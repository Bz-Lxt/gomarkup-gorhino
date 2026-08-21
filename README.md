# GoRhino

分布式 HTTP 压测控制台。Master 聚合、Worker 用 fasthttp 发包，内置 Target 用于自证。

## 1. 如何启动

```bash
docker compose up --build -d --scale worker=2
```

控制台：http://localhost:37101

## 2. 使用说明

打开「任务配置」，默认 URL 为 `http://target:8088/echo`，填写版本标签后创建并下发。实时监控通过 WebSocket 推送 RPS / P95 / P99（HDR 近似）。结束后在「历史报告」查看。

## 3. 服务列表及 API 说明

| 服务 | 地址 |
|---|---|
| 控制台 | http://localhost:37101 |
| Master HTTP | http://localhost:37102 |
| Master gRPC | localhost:37103 |
| Target | http://localhost:37104 |

接口细节见 `docs/API.md`。

## 4. 测试账号

MVP 无登录。

## 5. 题目内容

Go 分布式压测平台（Mini Locust / JMeter）：Web 下发任务、多 Worker 并发、实时汇总。

## 6. 项目结构

`backend/` Master·Worker·Target；`frontend-admin/` Vue 控制台；`tests/` Playwright 与 API smoke。

## 7. API 模拟与切换指南

压测引擎始终是真实 fasthttp 发包。默认目标是 Compose 内的 Target 服务（真实 HTTP）。外部地址必须写入白名单（控制台只读展示 + `PUT /api/v1/whitelist`）。关闭外部目标：保持默认白名单即可。不存在静默 mock 替换引擎的开关。
