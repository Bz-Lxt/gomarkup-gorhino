# GoRhino Design Spec

Aesthetic name: **Range Telemetry**
Date: 2026-08-21

A load-test console should feel like a ballistic-range instrument wall, not a SaaS marketing page. Operators stare at numbers under pressure. The UI is dark, dense, and keyed to phosphor / amber signal lamps.

---

## Direction

- Tone: industrial telemetry. Carbon chassis, engraved labels, live lamps.
- Remembered detail: amber corner brackets on every panel, phosphor digits for RPS, a single sweeping scanline on the live chart card.
- Not used: purple gradients, Inter, glassmorphism blobs, generic card grids with soft shadows.

---

## Palette

| Token | Hex | Use |
|---|---|---|
| `--bg` | `#07090c` | page |
| `--bg-panel` | `#10141b` | cards |
| `--bg-inset` | `#0b0e13` | inputs, tables |
| `--line` | `#243044` | borders |
| `--line-hot` | `#f6a53b` | active / focus |
| `--ink` | `#e8edf5` | primary text |
| `--muted` | `#8b9bb4` | labels |
| `--amber` | `#f6a53b` | primary action, P99 |
| `--phosphor` | `#7cffb2` | RPS, live, success |
| `--cyan` | `#5ec8ff` | P95, links |
| `--danger` | `#ff5a5a` | errors, stop |
| `--warn` | `#ffd166` | warnings |

CSS variables live in `src/style.css`. No ad-hoc hex in components except chart series.

---

## Typography

- Display / metrics: **Oxanium** (400 / 600 / 700)
- UI / Chinese: **Noto Sans SC** (400 / 500 / 700)
- Mono: **IBM Plex Mono** (labels, IDs, JSON)
- Dates: always `yyyy-MM-dd HH:mm:ss` (Beijing)

---

## Layout

- Full-bleed shell. No `max-w-*` on page containers.
- Top bar: logomark (rhino chevron) + nav + live worker count lamp.
- Main: 12-column grid. Task form left 5 / helper right 7 on desktop; stack at 768px.
- Live wall: 4 metric tiles on top, dual charts below, code histogram rail.
- 480px: hide side labels, tiles 2×2, tables scroll-x.

---

## Components

- **Button**: square-ish, 2px amber border, fill on primary. Hover lifts brightness. Disabled = 40% + not-allowed.
- **Input / Select**: inset field, custom chevron on all selects (`appearance: none`).
- **Toast**: top-right, close ×, auto-dismiss 5s. Types: ok / err / info.
- **Modal**: custom overlay, no `alert` / `confirm`. Danger actions require typed confirm or explicit 取消 / 确认.
- **Tiles**: label (muted, 11px tracking), value (Oxanium 36px), delta caption.
- **Table**: sticky header, row hover `#161c26`, status pills.
- **Charts**: ECharts dark, no toolbox. RPS phosphor area; latency dual-line cyan/amber. Empty state: "WAITING FOR FIRE".

---

## Pages

1. `/` 任务配置 — method, url, headers, body, VU, duration, QPS, version tag. Client validate before POST. Default URL `http://target:8088/echo`.
2. `/live` 实时监控 — WS tiles + charts. Banner if no running task, with link to create.
3. `/reports` 历史报告 — list, open detail. No compare UI (V1).
4. `/reports/:id` 报告详情 — summary + replay of 1s series.
5. `/nodes` 节点 — worker table, last heartbeat.

Percentile captions include `≈` and a footnote: HDR 近似，误差 ≤ 1%.

---

## Motion

- Lamp pulse 1.6s on `.is-live`
- Panel enter: 180ms fade + 6px rise
- Chart card scanline: 8s linear loop, 8% opacity
- No page-wide parallax

---

## Copy

Chinese UI. Error text from API `error.message`. Field errors render under the field in `--danger`. Save/start blocked until `validate()` passes, plus a toast summary.
