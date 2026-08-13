# Task 6 Report — Reasoning Processing

## Overview

Checkpoint: `715c0b50e12982e4460952db89e5c32486d23b0d`

建立 Codea 自己的 Reasoning 处理层：把 Runtime 输出的推理内容统一整理成稳定的 Reasoning 状态，供 Task 7 TUI 直接消费。实现结构化 Reasoning 识别、`<think>` fallback 解析、Reasoning 正规化、生命周期管理、streaming 增量合并、duration、reasoning 与 Answer 分离、多 block 顺序、malformed/不完整标签安全处理。仅消费现有 `runtime.Event`，零 Vendor DTO 依赖。

## Step 1 — Reasoning Domain + Normalizer

- Created `tui/internal/reasoning/domain.go`：
  - `BlockState`：`BlockActive` / `BlockCompleted` / `BlockInterrupted`
  - `Block`：`Index` / `Content` / `State` / `StartedAt` / `EndedAt` / `Duration`
  - `Snapshot`：有序 block 视图 + `Active()` / `HasActive()`
- Created `tui/internal/reasoning/normalizer.go`：
  - `Kind`：`KindReasoning` / `KindText` / `KindOther`
  - `Normalizer.Normalize(runtime.Event)` → `Normalized{Kind, Content}`
  - 只 import `codea/tui/internal/runtime`，零 OpenCode vendor 依赖
- 识别语义：`reasoning.delta` → KindReasoning（结构化主路径，不扫描 `<think>`）；`answer.delta` → KindText（可能含 `<think>`，交给 fallback parser）；tool/raw/session/error → KindOther
- Tests（`normalizer_test.go`）：结构化 reasoning delta、answer delta、tool/approval/step、raw、unknown、空 reasoning 均正确分类

## Step 2 — Structured Reasoning 主路径

结构化 reasoning（`reasoning.delta`）直接消费，不扫描 `<think>`。streaming 增量合并由 Tracker 承担。

- `Tracker.Start` → Active；`Append` → 内容增量；`End` → Completed
- 多个 reasoning delta 连续到达时增量合并，不漏 delta

## Step 3 — `<think>` Fallback Parser

- Created `tui/internal/reasoning/tag_parser.go`：stateful streaming FSM
  - 状态：`ThinkStateAnswer` / `ThinkStateReasoning`
  - `Feed(chunk) []ParserEvent` + `Flush()`（安全 finalize）+ `IsInReasoning()`
  - 事件：`ParserEventAnswerDelta` / `ParserEventReasoningStart` / `ParserEventReasoningDelta` / `ParserEventReasoningEnd`
- 关键能力：
  - 跨 chunk 标签拆分（`<thi` + `nk>`、`</th` + `ink>`、逐字符）
  - pending buffer 只保留标签前缀（≤7 字节），不缓存整段 Answer，无界缓存为 0
  - 未闭合 `<think>` 在 `Flush` 时安全 finalize，不 panic
  - 嵌套 `<think>`、`<<think>`、`<thinkx>` 等 malformed 输入不 panic/不死循环/不吞 answer
- 测试矩阵（`tag_parser_test.go`）：标准/跨 chunk/answer 前置文本/无 think/未闭合/空 think/多 block/嵌套/双 `<`/malformed/`<` 非标签/跨 chunk `<` 保留

## Step 4 — Reasoning Tracker

- `tui/internal/reasoning/tracker.go`：核心状态机
  - `Start` / `Append` / `End` / `Interrupt` / `Reset` / `Active` / `Snapshot`
  - `Clock` 接口 + `WithClock` 注入，测试不真实 sleep，duration 确定性
  - `sync.Mutex` 并发安全（`go test -race` 通过）
- 语义：
  - Start → Active；Append → 增量；End → Completed（duration = EndedAt−StartedAt）
  - Interrupt → BlockInterrupted（error/abort）
  - 多 block 顺序稳定（Index 递增）；新 Start 隐式完成前一个 block（不污染）
  - Append before Start / duplicate End / End without Start → 受控 no-op
  - 空 append 忽略；Reset 清空并重置 Index

## Step 5 — Structured + Fallback 集成契约

- Created `tui/internal/reasoning/processor.go`：组合 Normalizer + TagParser + Tracker，Task 7 唯一入口
  - `Process(runtime.Event) []Event` / `Flush()` / `Snapshot()` / `Reset()`
  - 输出 `Event{Kind, Content, Duration, Interrupted}`：`EventReasoningStart` / `EventReasoningDelta` / `EventReasoningEnd` / `EventAnswerDelta`
  - 去重：结构化 reasoning 已提供时，fallback `<think>` 内容剥离但不重复输出（遇到非空纯 answer 文本后重置去重标志，保证多 block 正确）
  - 空 reasoning delta / 空 `<think></think>` 不产生空 block
- Created `tui/tests/contract/reasoning_event_test.go`：4 条完整链路契约
  - Case A：结构化 reasoning → `[start, delta, end, answer]`
  - Case B：`<think>` fallback 输出与 Case A 用户语义一致
  - Case C：结构化 + think 同时存在 → 去重，answer 不含重复推理标签
  - Case D：无 reasoning 模型 → 纯 answer，完全正常

## Full Gate Verification

针对 Final Implementation Commit `715c0b50e12982e4460952db89e5c32486d23b0d`：

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（16 packages） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（无 vendor DTO 泄漏） |
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | valid |

## Test Summary

| Package | Tests |
|---------|-------|
| internal/reasoning | normalizer（6）+ tracker（15）+ tag_parser（13）+ processor（12） |
| tests/contract | 1（`reasoning_event_test.go`，4 Case + 类型映射一致性） |

覆盖清单：Structured reasoning start/delta/completed、普通 Answer 不被识别、Tool/Raw 不误判、空 reasoning 无垃圾状态、Vendor DTO 零依赖、`<think>` 单 chunk/跨 chunk/malformed、Structured+fallback 去重、Reasoning/Answer 分离、duration、多 block、no-reasoning model。

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/reasoning/domain.go` | Create |
| `tui/internal/reasoning/normalizer.go` | Create |
| `tui/internal/reasoning/tracker.go` | Create |
| `tui/internal/reasoning/tag_parser.go` | Create |
| `tui/internal/reasoning/processor.go` | Create |
| `tui/internal/reasoning/normalizer_test.go` | Create |
| `tui/internal/reasoning/tracker_test.go` | Create |
| `tui/internal/reasoning/tag_parser_test.go` | Create |
| `tui/internal/reasoning/processor_test.go` | Create |
| `tui/tests/contract/reasoning_event_test.go` | Create |

## 提交记录

| Commit | Step |
|--------|------|
| `ac286a6` | Step 1 — Reasoning Domain + Normalizer |
| `d7fff83` | Step 2/4 — Reasoning Tracker 状态机 |
| `f289dba` | Step 3 — `<think>` fallback streaming parser |
| `715c0b5` | Step 5 — Processor + 集成契约（Final Implementation Commit） |

## 与计划偏差

无。Task 6 完全局限于 `tui/internal/reasoning/` 与 `tui/tests/contract/reasoning_event_test.go`，未修改 Runtime Contract，未新增事件到 application/TUI，未重构 opencode/runtime 层。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不启动 Task 7。
