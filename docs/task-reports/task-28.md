# Task 28 Report — Repo Intelligence

Task 28 production implementation and automated verification are complete. Human acceptance remains a separate final step.

Verified production checkpoint: `27f6c41e18b3ac8e0ea471f3e9cbbcb1802f594e`

Plan: `docs/superpowers/plans/2026-08-28-codea-v1.2-task28-repo-intelligence-plan.md`

Design baseline: `docs/superpowers/specs/2026-08-28-codea-v1.2-agent-intelligence-reliability-design.md`

## Scope delivered

Task 28 adds a Codea-owned, offline-safe repository intelligence layer without changing the AgentRuntime vendor boundary:

- safe repository walking with ignored/generated/binary/sensitive-path filtering;
- Java/Spring structural extraction for controllers, services, repositories, imports, methods, annotations and typed relations;
- Go AST extraction plus a deterministic generic fallback for unsupported source languages;
- conservative cross-file graph resolution where ambiguous relations remain unresolved instead of being promoted to facts;
- task-specific deterministic ranking using exact matches, changed files, type/reference/import evidence, graph neighbors and lexical tie-breaking;
- bounded deterministic Repo Map rendering with hard character budgets and sensitive environment files excluded;
- asynchronous Application-side Repo Context preparation before prompting;
- Repo Map injection as synthetic `runtime.TextPart` at `Parts[0]`, while the original user prompt remains unchanged at the following part;
- graceful degradation: indexing failure does not block the user prompt;
- production composition root injection through the existing project root, with no vendor DTO leakage into Application;
- dedicated Task 28 mechanical and three-platform CI Gates.

## Implementation commits

- `33fb170b8c2718cd1b80d7fa4b72187d88298330` — safe repo context walker
- `4764d2b1e68c42bbd031b149a40a16baffbbe2f6` — Java/Spring repository structure extraction
- `7fefab13d648d1f8c3be6cced28c11ab040b8cef` — Go AST and generic fallback indexing
- `6ced07cbc578d6316a546ce46509c5c433a78987` — task-relevant graph/ranking and Git signal
- `9cb5ebccef6ad05eb15519adc6eecdb4d4f90058` — full ranking-priority enforcement
- `c901c11bfc6f65bb3ce57418a7621f1a1ab3326f` — bounded deterministic Repo Map
- `9fbbda97341464a4582bc9af226b59fe128cdf26` — asynchronous Repo Context prompt injection
- `a84fefd373daf5ac449e2b483d9815949b316325` — Task 28 dedicated acceptance Gates
- `27f6c41e18b3ac8e0ea471f3e9cbbcb1802f594e` — Gate false-positive scope correction; production boundary remains unchanged

## TDD evidence

Task 28.6 Application injection was independently re-proven with RED/GREEN branches before the production commit:

- RED run `33218454153`: **expected failure** — `SetRepoContextService` was undefined, proving the new behavior test failed for the intended missing capability.
- GREEN run `33218514956`: **SUCCESS** — focused `RepoContext|RepoMapComposition` Application/composition tests passed and the architecture boundary test passed.

The RED/GREEN proof covered ordinary prompts, professional prompt commands, synthetic context ordering, unchanged user text, failure degradation, first-session ordering, and non-blocking prompt submission.

## Fresh automated acceptance

Verified checkpoint `27f6c41e18b3ac8e0ea471f3e9cbbcb1802f594e`, Task 28 Repo Intelligence Gates run `33218786421`: **SUCCESS**.

### Mechanical acceptance

The Linux Gate emitted every required Task 28 marker after the corresponding real tests/builds passed:

```text
JAVA_CHAIN OrderController#createOrder -> OrderService#createOrder -> OrderRepository#save
AMBIGUOUS_RELATION_NOT_PROMOTED PASS
REPO_MAP_DETERMINISTIC PASS
REPO_MAP_BUDGET PASS
WINDOWS_CGO0_BUILD PASS
TASK28_REPO_INTELLIGENCE PASS
```

It also passed `codea/tui/tests/architecture` before the Task 28 behavior checks.

### Full regression evidence

Run `33218786421` completed all three platform jobs successfully:

- Linux: mechanical Task 28 acceptance **PASS**; `go test ./... -count=1` **PASS**; enterprise plugin regression **247 pass / 0 fail**; plugin build **PASS** (`93 modules` bundled).
- Windows 2025: native Task 28 focused regression with `CGO_ENABLED=0` **PASS**; native full Go regression **PASS**; `go build ./cmd/codea` **PASS**.
- macOS 15: native Task 28 focused regression **PASS**; native full Go regression **PASS**; `go build ./cmd/codea` **PASS**.

The mechanical Gate additionally proves the pure-Go Windows cross-build contract with `CGO_ENABLED=0 GOOS=windows GOARCH=amd64`.

## Architecture and safety result

- Repo Intelligence and Application production code do not import OpenCode vendor implementation packages.
- `repoctx` does not introduce network libraries or CGO.
- no tree-sitter dependency is added to `tui/go.mod`.
- ambiguity is represented as unresolved evidence, not silently upgraded to a confirmed call relation.
- Repo Map output is deterministic and hard-budgeted.
- sensitive environment files are excluded from Repo Map context.
- repository indexing executes in an async Bubble Tea command rather than synchronously in `Update`/submit handling.

## Known limitations

These are intentional V1.2 boundaries, not failed acceptance items:

- deep typed structural extraction is implemented for Java/Spring and Go; other languages use the generic fallback and are **not** claimed to have AST-level precision;
- relevance ranking is deterministic evidence-based heuristic ranking, not embedding/semantic-search ranking;
- the hard Repo Map budget can omit lower-ranked repository context by design;
- ambiguous targets stay unresolved until stronger evidence exists, which favors correctness over recall.

## Result

- Automated verification: **PASS**
- Task Gate: **PASS**
- Verified checkpoint: `27f6c41e18b3ac8e0ea471f3e9cbbcb1802f594e`
- Automated Gate run: `33218786421` — **SUCCESS**
- Human acceptance: **PENDING**

Current state: **AWAITING_ACCEPTANCE**
