# 报表导出服务

这是一个用于演示异步报表导出流程的 Go Web 后端服务。服务提供创建、查询、取消和重试导出任务的 HTTP 接口，后台 worker 负责生成 CSV 文件地址。

## 接口

- `POST /v1/exports`：创建导出任务，参数为 `{"report":"daily-sales"}`。
- `GET /v1/exports/{id}`：查询导出任务状态。
- `DELETE /v1/exports/{id}`：取消正在执行的导出任务。
- `POST /v1/exports/{id}/retry`：重试已取消或失败的任务。

## 本地运行

运行环境：

- Go 工具链：`go1.26.2`
- `go.mod` 语言版本：`go 1.26.2`
- `go.mod` 不包含 `toolchain` 指令
- 必须设置 `GOTOOLCHAIN=local`，避免 Go 自动切换工具链

```sh
export GOTOOLCHAIN=local
export GOCACHE=/private/tmp/report-export-gocache
go run ./cmd/server
```

## 测试

```sh
export GOTOOLCHAIN=local
export GOCACHE=/private/tmp/report-export-gocache
go test -race ./...
```

## 约束

任务取消后允许重试。不同执行批次的异步结果不能覆盖当前批次已经产生的状态和文件地址。

## 提交修复

完成修复并通过测试后，提交一次中文 Git commit。提交前应确认工作区只包含与当前问题相关的源码改动，不修改或跳过既有测试用例。
