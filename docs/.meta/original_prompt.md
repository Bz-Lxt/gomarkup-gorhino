# Original Prompt

Date: 2026-08-21 (GMT+8)
Command: `/pm`
Project: GoRhino

## User Prompt (verbatim)

使用Go语言帮我实现一个分布式性能测试与压测平台 (类似Mini Locust / JMeter)
业务背景：一个可以通过 Web 界面下发压测任务，由多个 Go Worker 节点并发发起请求，并实时汇总统计报表的全栈系统。
前端页面 (Vue 3 + Tailwind)：任务配置页：配置请求 URL、Headers、并发用户数、压测持续时间。实时监控大屏：利用 WebSocket 实时展示每秒请求数 (RPS)、响应耗时 (P99/P95) 的折线图。历史报告列表：对比不同版本代码的压测数据差异。
Go 后端核心实现：Master-Worker 架构：实现节点间的自发现与心跳检测（基于 gRPC）。高压请求引擎：利用 fasthttp 或定制化 net 包实现极高性能的并发发包。数据聚合算法：在毫秒级处理成千上万个响应状态，汇总成统计摘要。
文件数与代码量：约 25~35 个 Go 文件，代码量 5000 - 7500 行。

## User Architecture Decisions (2026-08-21)

1. Target scope: builtin target service + configurable external URL whitelist. Default deny non-whitelist addresses. SSRF protection required (resolve then validate IP ranges, reject 169.254/metadata).
2. Version source for report comparison: manual version tag / Git SHA field on the task form. Reports grouped by tag. Zero external Git dependency.
3. Persistence: SQLite single file.

## PM Gate Result

ACCEPT. Scale 10k–40k LoC band. Phased Roadmap is mandatory before any code.
