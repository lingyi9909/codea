# Task 28 Report — Repo Intelligence

Task 28 production implementation, reviewer-requested remediation, and automated verification are complete. Human acceptance remains a separate final step.

Verified production checkpoint: `2dbdc75e90dcc959b62160b035ac9233d96b6cf9`

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
- `27f6c41e18b3ac8e0ea471f3e9cbbcb1802f594e` — Gate false-positive scope correction; production boundary unchanged
- `7ced3e8e4ab0ef97e53e134e03091d1e391a7d78` — reviewer remediation: package/import-aware relation resolution, generic type parsing, Spring DI evidence, and `.mvn/wrapper` exclusion
- `2dbdc75e90dcc959b62160b035ac9233d96b6cf9` — final Go relation correctness fix: exact root-module canonical import identity and external-module collision regression

## TDD evidence

Task 28.6 Application injection was independently re-proven with RED/GREEN branches before the production commit:

- RED run `33218454153`: **expected failure** — `SetRepoContextService` was undefined.
- GREEN run `33218514956`: **SUCCESS** — focused Repo Context/Application/composition tests and architecture boundary passed.

Reviewer remediation was also executed RED → GREEN before the production commit:

- RED run `33230939358`: **expected failure**. Seven negative tests reproduced the reported Java/Go relation, generic parsing, constructor DI, and `.mvn/wrapper` defects.
- GREEN run `33231261488`: **SUCCESS** under Go 1.26.5. The same regressions passed after the scoped fixes.
- The temporary TDD workflow was not copied to `develop`; the regression tests themselves are permanent production tests.

The final Go import-identity blocker was independently reproduced and fixed with another RED → GREEN proof:

- RED run `33232024503`: **expected failure**. `github.com/external/project/internal/good` was incorrectly promoted to the local `internal/good` package because the old resolver accepted a directory suffix match.
- GREEN run `33232101753`: **SUCCESS** under Go 1.26.5. The external collision test, all `TestTask28Remediation*` tests, and the whole `internal/repoctx` package passed.
- Production now reads a reliable root `go.mod` module directive and confirms qualified Go relations only when the import path exactly equals `<module-path>/<repository-relative-directory>`. Missing or ambiguous module identity degrades to unresolved.

## Reviewer remediation

The remediation scope remained intentionally limited to the reported correctness items:

1. Java relation resolution uses package/import evidence; unresolved/ambiguous targets are not promoted.
2. Java generic field/parameter parsing preserves the outer declared type.
3. Constructor `RelationInjects` requires reliable Spring DI evidence; ordinary POJO constructors do not generate confirmed injection relations.
4. `.mvn/wrapper` is excluded by path-aware matching without excluding unrelated `.mvn` files.
5. Go qualified relation resolution now requires exact canonical module import identity. Directory suffix similarity is never sufficient evidence.

The permanent `TestTask28Remediation*` regressions are executed by `tests/task28-repo-intelligence.sh` and by the native Windows/macOS Task 28 focused regex. The external-module collision has its own mechanical marker: `EXTERNAL_GO_IMPORT_NOT_PROMOTED PASS`.

## Fresh automated acceptance after final remediation

Verified checkpoint `2dbdc75e90dcc959b62160b035ac9233d96b6cf9`, Task 28 Repo Intelligence Gates run `33232361094`: **SUCCESS**.

### Mechanical acceptance

The Linux Gate emitted the required markers only after the corresponding real tests/builds passed:

```text
TASK28_REMEDIATION_REGRESSIONS PASS
EXTERNAL_GO_IMPORT_NOT_PROMOTED PASS
JAVA_CHAIN OrderController#createOrder -> OrderService#createOrder -> OrderRepository#save
AMBIGUOUS_RELATION_NOT_PROMOTED PASS
REPO_MAP_DETERMINISTIC PASS
REPO_MAP_BUDGET PASS
WINDOWS_CGO0_BUILD PASS
TASK28_REPO_INTELLIGENCE PASS
```

### Full regression evidence

Run `33232361094` completed all three platform jobs successfully:

- Linux: execution-state validation **PASS**; remediation regressions **PASS**; external Go import collision **PASS**; mechanical Task 28 acceptance **PASS**; `go test ./... -count=1` **PASS**; enterprise plugin regression **247 pass / 0 fail**; plugin build **PASS** (`93 modules` bundled).
- Windows Server 2025: Go 1.26.5; native Task 28 focused regression with `CGO_ENABLED=0` **PASS**; `go build ./cmd/codea` with `CGO_ENABLED=0` **PASS**; native full Go regression with `CGO_ENABLED=0` **PASS**.
- macOS 15 arm64: Go 1.26.5; native Task 28 focused regression **PASS**; native full Go regression **PASS**; `go build ./cmd/codea` **PASS**.

The mechanical Gate additionally proves the pure-Go Windows cross-build contract with `CGO_ENABLED=0 GOOS=windows GOARCH=amd64`.

## Architecture and safety result

- Repo Intelligence and Application production code do not import OpenCode vendor implementation packages.
- `repoctx` does not introduce network libraries or CGO.
- no tree-sitter dependency is added to `tui/go.mod`.
- Java ambiguity and Go module/import ambiguity are represented as unresolved evidence, not silently upgraded to confirmed relations.
- Repo Map output remains deterministic and hard-budgeted.
- sensitive environment files remain excluded from Repo Map context.
- repository indexing remains asynchronous and outside Bubble Tea synchronous submit/update handling.

## Known limitations

These remain intentional V1.2 boundaries, not failed acceptance items:

- deep typed structural extraction is implemented for Java/Spring and Go; other languages use the generic fallback and are not claimed to have AST-level precision;
- multi-module or repositories without a reliable root `go.mod` do not receive qualified cross-package Go confirmation; those candidates stay unresolved by design;
- relevance ranking is deterministic evidence-based heuristic ranking, not embedding/semantic-search ranking;
- the hard Repo Map budget can omit lower-ranked repository context by design;
- ambiguous targets stay unresolved until stronger evidence exists, favoring correctness over recall.

## Result

- Reviewer remediation: **PASS**
- Final Go import-identity remediation: **PASS**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Verified checkpoint: `2dbdc75e90dcc959b62160b035ac9233d96b6cf9`
- Final blocker RED run: `33232024503` — **EXPECTED FAILURE**
- Final blocker GREEN run: `33232101753` — **SUCCESS**
- Automated Gate run: `33232361094` — **SUCCESS**
- Human acceptance: **PENDING**

Current state: **AWAITING_ACCEPTANCE**
