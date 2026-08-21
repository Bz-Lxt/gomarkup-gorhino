package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorhino/internal/shared/model"
	"gorhino/internal/timeutil"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := path
	if path != ":memory:" {
		dsn = "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	} else {
		dsn = "file:memdb1?mode=memory&cache=shared&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	const q = `
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  method TEXT NOT NULL,
  url TEXT NOT NULL,
  headers TEXT NOT NULL,
  body TEXT NOT NULL,
  vu INTEGER NOT NULL,
  duration_sec INTEGER NOT NULL,
  qps INTEGER NOT NULL,
  version_tag TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  started_at TEXT,
  ended_at TEXT
);
CREATE TABLE IF NOT EXISTS snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL,
  ts TEXT NOT NULL,
  rps REAL NOT NULL,
  p50_ms REAL NOT NULL,
  p95_ms REAL NOT NULL,
  p99_ms REAL NOT NULL,
  avg_ms REAL NOT NULL,
  error_rate REAL NOT NULL,
  codes TEXT NOT NULL,
  workers INTEGER NOT NULL,
  elapsed_sec INTEGER NOT NULL,
  remaining_sec INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snap_task ON snapshots(task_id);
CREATE TABLE IF NOT EXISTS reports (
  task_id TEXT PRIMARY KEY,
  final_rps REAL NOT NULL,
  p50_ms REAL NOT NULL,
  p95_ms REAL NOT NULL,
  p99_ms REAL NOT NULL,
  avg_ms REAL NOT NULL,
  error_rate REAL NOT NULL,
  total_requests INTEGER NOT NULL,
  total_errors INTEGER NOT NULL,
  codes TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS whitelist (
  pattern TEXT PRIMARY KEY
);
`
	_, err := s.db.Exec(q)
	return err
}

func NewID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

func (s *Store) SeedWhitelist(patterns []string) error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM whitelist`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	st, err := tx.Prepare(`INSERT OR IGNORE INTO whitelist(pattern) VALUES (?)`)
	if err != nil {
		return err
	}
	defer st.Close()
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := st.Exec(p); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListWhitelist() ([]string, error) {
	rows, err := s.db.Query(`SELECT pattern FROM whitelist ORDER BY pattern`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceWhitelist(patterns []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM whitelist`); err != nil {
		return err
	}
	st, err := tx.Prepare(`INSERT INTO whitelist(pattern) VALUES (?)`)
	if err != nil {
		return err
	}
	defer st.Close()
	for _, p := range patterns {
		if _, err := st.Exec(p); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CreateTask(spec model.TaskSpec) (*model.Task, error) {
	t := &model.Task{
		ID:          NewID("tsk_"),
		Method:      spec.Method,
		URL:         spec.URL,
		Headers:     spec.Headers,
		Body:        spec.Body,
		VU:          spec.VU,
		DurationSec: spec.DurationSec,
		QPS:         spec.QPS,
		VersionTag:  spec.VersionTag,
		Status:      model.StatusDraft,
		CreatedAt:   timeutil.NowString(),
	}
	_, err := s.db.Exec(
		`INSERT INTO tasks(id,method,url,headers,body,vu,duration_sec,qps,version_tag,status,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Method, t.URL, model.HeadersJSON(t.Headers), t.Body, t.VU, t.DurationSec, t.QPS, t.VersionTag, t.Status, t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func scanTask(sc interface{ Scan(dest ...any) error }) (*model.Task, error) {
	var t model.Task
	var headers string
	var started, ended sql.NullString
	if err := sc.Scan(&t.ID, &t.Method, &t.URL, &headers, &t.Body, &t.VU, &t.DurationSec, &t.QPS, &t.VersionTag, &t.Status, &t.CreatedAt, &started, &ended); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Headers = model.ParseHeaders(headers)
	t.StartedAt = started.String
	t.EndedAt = ended.String
	return &t, nil
}

const taskCols = `id,method,url,headers,body,vu,duration_sec,qps,version_tag,status,created_at,started_at,ended_at`

func (s *Store) GetTask(id string) (*model.Task, error) {
	return scanTask(s.db.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE id=?`, id))
}

func (s *Store) ListTasks() ([]model.Task, error) {
	rows, err := s.db.Query(`SELECT ` + taskCols + ` FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) FailStaleRunning() error {
	now := timeutil.NowString()
	_, err := s.db.Exec(`UPDATE tasks SET status=?, ended_at=? WHERE status=?`, model.StatusFailed, now, model.StatusRunning)
	return err
}

func (s *Store) RunningTask() (*model.Task, error) {
	t, err := scanTask(s.db.QueryRow(`SELECT `+taskCols+` FROM tasks WHERE status=? LIMIT 1`, model.StatusRunning))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return t, err
}

func (s *Store) MarkRunning(id string) error {
	now := timeutil.NowString()
	res, err := s.db.Exec(`UPDATE tasks SET status=?, started_at=? WHERE id=? AND status=?`, model.StatusRunning, now, id, model.StatusDraft)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: task not draft", ErrConflict)
	}
	return nil
}

func (s *Store) FinishTask(id, status string) error {
	_, err := s.db.Exec(`UPDATE tasks SET status=?, ended_at=? WHERE id=?`, status, timeutil.NowString(), id)
	return err
}

func (s *Store) InsertSnapshot(sn model.Snapshot) error {
	_, err := s.db.Exec(
		`INSERT INTO snapshots(task_id,ts,rps,p50_ms,p95_ms,p99_ms,avg_ms,error_rate,codes,workers,elapsed_sec,remaining_sec)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		sn.TaskID, sn.TS, sn.RPS, sn.P50MS, sn.P95MS, sn.P99MS, sn.AvgMS, sn.ErrorRate, model.CodesJSON(sn.Codes),
		sn.Workers, sn.ElapsedSec, sn.RemainingSec,
	)
	return err
}

func (s *Store) ListSnapshots(taskID string) ([]model.Snapshot, error) {
	rows, err := s.db.Query(
		`SELECT task_id,ts,rps,p50_ms,p95_ms,p99_ms,avg_ms,error_rate,codes,workers,elapsed_sec,remaining_sec
		 FROM snapshots WHERE task_id=? ORDER BY id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Snapshot
	for rows.Next() {
		var sn model.Snapshot
		var codes string
		if err := rows.Scan(&sn.TaskID, &sn.TS, &sn.RPS, &sn.P50MS, &sn.P95MS, &sn.P99MS, &sn.AvgMS, &sn.ErrorRate, &codes, &sn.Workers, &sn.ElapsedSec, &sn.RemainingSec); err != nil {
			return nil, err
		}
		sn.Codes = model.ParseCodes(codes)
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *Store) UpsertReport(r model.Report) error {
	_, err := s.db.Exec(
		`INSERT INTO reports(task_id,final_rps,p50_ms,p95_ms,p99_ms,avg_ms,error_rate,total_requests,total_errors,codes)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(task_id) DO UPDATE SET
		   final_rps=excluded.final_rps, p50_ms=excluded.p50_ms, p95_ms=excluded.p95_ms, p99_ms=excluded.p99_ms,
		   avg_ms=excluded.avg_ms, error_rate=excluded.error_rate, total_requests=excluded.total_requests,
		   total_errors=excluded.total_errors, codes=excluded.codes`,
		r.ID, r.FinalRPS, r.P50MS, r.P95MS, r.P99MS, r.AvgMS, r.ErrorRate, r.TotalRequests, r.TotalErrors, model.CodesJSON(r.Codes),
	)
	return err
}

func (s *Store) GetReport(id string) (*model.Report, error) {
	t, err := s.GetTask(id)
	if err != nil {
		return nil, err
	}
	var r model.Report
	r.Task = *t
	var codes string
	err = s.db.QueryRow(
		`SELECT final_rps,p50_ms,p95_ms,p99_ms,avg_ms,error_rate,total_requests,total_errors,codes FROM reports WHERE task_id=?`, id,
	).Scan(&r.FinalRPS, &r.P50MS, &r.P95MS, &r.P99MS, &r.AvgMS, &r.ErrorRate, &r.TotalRequests, &r.TotalErrors, &codes)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	r.Codes = model.ParseCodes(codes)
	series, err := s.ListSnapshots(id)
	if err != nil {
		return nil, err
	}
	r.Series = series
	return &r, nil
}

func (s *Store) ListReports() ([]model.Report, error) {
	q := `SELECT t.id,t.method,t.url,t.headers,t.body,t.vu,t.duration_sec,t.qps,t.version_tag,t.status,t.created_at,t.started_at,t.ended_at,
		COALESCE(r.final_rps,0),COALESCE(r.p50_ms,0),COALESCE(r.p95_ms,0),COALESCE(r.p99_ms,0),COALESCE(r.avg_ms,0),
		COALESCE(r.error_rate,0),COALESCE(r.total_requests,0),COALESCE(r.total_errors,0),COALESCE(r.codes,'{}')
		FROM tasks t LEFT JOIN reports r ON r.task_id=t.id
		WHERE t.status IN ('completed','stopped','failed') ORDER BY t.created_at DESC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Report
	for rows.Next() {
		var t model.Task
		var headers, codes string
		var started, ended sql.NullString
		var r model.Report
		if err := rows.Scan(&t.ID, &t.Method, &t.URL, &headers, &t.Body, &t.VU, &t.DurationSec, &t.QPS, &t.VersionTag, &t.Status, &t.CreatedAt, &started, &ended,
			&r.FinalRPS, &r.P50MS, &r.P95MS, &r.P99MS, &r.AvgMS, &r.ErrorRate, &r.TotalRequests, &r.TotalErrors, &codes); err != nil {
			return nil, err
		}
		t.Headers = model.ParseHeaders(headers)
		t.StartedAt = started.String
		t.EndedAt = ended.String
		r.Task = t
		r.Codes = model.ParseCodes(codes)
		out = append(out, r)
	}
	return out, rows.Err()
}

func MustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
