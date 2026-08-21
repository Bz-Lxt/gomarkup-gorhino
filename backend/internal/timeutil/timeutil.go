package timeutil

import (
	"time"
)

const Layout = "2006-01-02 15:04:05"

// Beijing is GMT+8 without relying on tzdata being present (still install tzdata in images).
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format(Layout)
}

func NowString() string {
	return Format(Now())
}

func Parse(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation(Layout, s, Beijing)
}

func UnixMS() int64 {
	return Now().UnixMilli()
}
