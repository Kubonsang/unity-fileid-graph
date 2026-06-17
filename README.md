# unity-fileid-graph

Experimental Unity fileID graph analyzer and native mutation safety lab.

This project explores whether limited native Unity YAML mutation can be made safe through lossless block preservation, graph integrity checks, and transaction-based write pipelines.

Parser is infrastructure. Safety planner is the product.

## v0.1 Scope

- Read-only parser only
- Supports `prefab`, `scene`, `asset`, and `mat` namespaces
- Supports `blocks` command only
- Preserves `HeaderRaw`, `BodyRaw`, block order, LF, CRLF, and document end markers
- Does not reject duplicate fileIDs during parsing

## v0.2 Scope

- Adds read-only `graph` extraction on top of the v0.1 block parser
- Extracts `GameObject`, `Transform`, and `MonoBehaviour.m_Script` relationships
- Preserves duplicate fileID evidence for later graph checks
- Prints `WARN` lines for unsupported shapes instead of crashing
- May print partial nodes together with `WARN` lines when extraction is incomplete
- Does not mutate files or resolve project-level GUIDs

## v0.3 Scope

- Adds read-only `check` validation on top of the v0.2 parser and graph pipeline
- Runs local integrity checks for duplicate fileIDs, missing component blocks, missing `m_GameObject` targets, GameObject/component back-reference mismatches, Transform parent/child mismatches, and missing Transform components
- Flags suspicious `MonoBehaviour.m_Script` metadata shapes without resolving project-level GUIDs
- Preserves parser and graph warnings as `WARN` output during `check`
- Returns exit code `1` when integrity errors are found, and exit code `0` for `OK` or `WARN` results
- Does not mutate YAML files or perform any write-back behavior

## v0.4 Scope

- Adds `roundtrip` for no-op lossless block copy experiments
- Writes a copy using preserved `PreambleRaw`, `HeaderRaw`, `BodyRaw`, block order, and `TrailerRaw`
- Verifies byte equality, reparse success, block-sequence equality, graph-check status, and line-ending preservation
- Reports `editor_open=NOT_CHECKED` by default because no Unity Editor harness is wired in this milestone
- Implements only `lossless-block-copy`
- Does not implement mutation, scalar set, or generic YAML serialization

## v0.5 Scope

- Adds `set` for safe mutation of existing top-level scalar fields only
- Supports scalar replacements for `bool`, `int`, `float`, and `string`
- Uses a transactional write pipeline with pre-check, temp-file verification, backup, atomic rename, and final re-read
- Preserves parser/graph `WARN` results without treating them as fatal
- Emits replacement strings as quoted YAML strings in `v0.5`
- Blocks duplicate fileID, MonoBehaviour, stripped-object, nested-field, list, and inline-object mutations
- Does not implement add/remove/reparent or generic YAML rewriting

## v0.6 Scope

- Adds experimental `remove-component` for a small built-in component allowlist only
- Requires both `--experimental` and `--write`
- Restricts the command to the `prefab` namespace
- Preserves the transactional write pipeline from `v0.5`
- Does not implement add-child, reparent, or generic structural editing

## v0.8 Scope

- Accepts negative top-level Unity header `fileID` values
- Hardens `restore_failed=true` coverage for the shared write pipeline
- Keeps `remove-component` prefab-only
- Keeps the built-in remove allowlist limited to `BoxCollider` and `Rigidbody`
- Adds dependency-aware blocked reasons for `MeshRenderer` and `MeshFilter`
- Adds richer `REMOVE_COMPONENT` error output when restore metadata is present

## v0.9a Scope

- Adds `check --json` as the first stable machine-readable safety-kernel output.
- Adds read-only `refs` and `refs --json` for targeted Unity PPtr/GUID evidence.
- `refs` includes local fileID-only references and external GUID-backed references.
- `refs` supports inline PPtr values only in v0.9a; malformed inline values may produce warning-only extraction issues, while multiline PPtr values are deferred.
- `refs` field paths are best-effort evidence labels and are not a full YAML AST path contract.
- `refs` is read-only evidence extraction; `status=WARN` means warning-only extraction issues and still exits `0`.
- `refs` may include `file_id=0` null PPtr values outside skipped graph-structural fields; consumers should treat them as null evidence, not object edges.
- `refs` skips graph-structural Transform/GameObject fields already modeled by `graph`, including `m_GameObject`, `m_Father`, and `m_Children`.
- JSON `file` fields preserve the input path exactly as provided.
- `check --json` emits `ERROR` issues first and `WARN` issues second.
- Keeps `blocks` and `graph` text-only in v0.9a.
- Does not add new mutation commands or expand structural write support.
- Does not add a generic YAML parser, generic serializer, multiline PPtr parser, or structural mutation.
- Intended consumer: `unity-ctx` can call `uyaml ... check --json` and `uyaml ... refs --json` before integrating graph safety into write paths.

## v0.9.1 Scope (gap1 — filled m_Children + symmetry skip visibility)

- Parses Unity's real `m_Children` serialization where the child dash sits at the
  **same indent** as the `m_Children:` key (F3), not only the deeper-indented form.
  Accepts empty `[]`, deeper-dash, same-indent dash, and key-only (empty) blocks;
  a non-empty inline `[{...}]` array stays a warning-only `UNKNOWN_FIELD_SHAPE`.
- Because filled `m_Children` now parses, the existing (ERROR-severity)
  `TRANSFORM_PARENT_CHILD_MISMATCH` symmetry check is no longer silently bypassed
  by an `UNKNOWN_FIELD_SHAPE` skip; genuine parent/child asymmetry now surfaces.
- **Conservative symmetry skip:** a parent/child link is asserted only when both
  endpoints have locally authoritative data. A link is **skipped** (never silently
  — counted and reported) when the other endpoint is a **stripped** nested
  prefab-instance block, or a **present-but-unmodeled class** (e.g. `RectTransform`
  224). A reference to a fileID with **no block at all** is still a genuine
  dangling `TRANSFORM_PARENT_CHILD_MISMATCH` ERROR. The premise "asymmetry =
  broken graph" holds only within the kernel's modeled scope: skipping stripped /
  unmodeled endpoints is honest modeling, not false-positive hiding.
- `check` reports `skipped=N` on the summary line (and `skip_reasons=stripped:M,unmodeled_class:K`
  when `N>0`); `check --json` adds `skipped_links`, `skipped_stripped`,
  `skipped_unmodeled_class` to `summary`. A passing check therefore never silently
  means "skipped everything" — it means clean within the checkable scope.
- The write path carries the same visibility: `set` reports `pre_check_skipped=N`
  (and `pre_check_skip_reasons=...` when `N>0`), and `core.SetResult` /
  `core.RemoveComponentResult` expose the pre_check skip counts, so a committed
  write is never read as fully symmetry-checked when stripped/unmodeled endpoints
  were skipped.
- **Deferred (future slice):** model `RectTransform` (class 224) in transform
  symmetry so UI hierarchies are validated rather than skipped, and re-survey any
  further unmodeled transform classes — deferred pending a UI-heavy corpus, to be
  revisited before the reparent slice. Until then, 224 endpoints are skipped-and-counted.

## Library Surface (pkg/)

`pkg/` is the supported Go library surface for external consumers such as `unity-ctx`:

- `pkg/parser` — lossless block parsing (`parser.Parse`)
- `pkg/graph` — fileID graph extraction (`graph.Build`)
- `pkg/check` — graph integrity validation (`check.Run`)
- `pkg/refs` — PPtr/GUID reference evidence extraction (`refs.Extract`)
- `pkg/core` — shared data model (`Block`, `Graph`, `CheckResult`, `RefsResult`, ...)

```go
parsed, err := parser.Parse(data)
g, err := graph.Build(parsed)
result := check.Run(g) // result.Status: OK | WARN | ERROR
```

`internal/` (`cli`, `mutate`, `roundtrip`) remains private to the `uyaml` CLI.
Consumers should pin a tagged release; until `v0.9.0` is tagged, a local
`replace` directive is required.

## Usage

```bash
go run ./cmd/uyaml prefab blocks testdata/fixtures/simple_prefab.prefab
go run ./cmd/uyaml prefab graph testdata/fixtures/graph_prefab.prefab
go run ./cmd/uyaml prefab check testdata/fixtures/check_ok.prefab
go run ./cmd/uyaml prefab check testdata/fixtures/check_ok.prefab --json
go run ./cmd/uyaml prefab refs testdata/fixtures/refs_prefab.prefab
go run ./cmd/uyaml prefab refs testdata/fixtures/refs_prefab.prefab --json
go run ./cmd/uyaml prefab roundtrip testdata/fixtures/check_ok.prefab --out /tmp/check_ok.copy.prefab
go run ./cmd/uyaml prefab set testdata/fixtures/set_prefab.prefab --id 1000 --field m_IsActive --value 0
cp testdata/fixtures/remove_component_ok.prefab /tmp/remove_component_ok.prefab
go run ./cmd/uyaml prefab remove-component /tmp/remove_component_ok.prefab --id 65000 --experimental --write
```

The `set` command modifies files in place after creating a backup.
Use it on version-controlled files and review the diff.

The `remove-component` command is intentionally experimental and allowlist-only in `v0.6`.
It is limited to the `prefab` namespace, and `WARN` is reflected through the `pre_check`, `temp_check`, and `final_check` fields rather than a top-level `WARN` status.
Use it on version-controlled files, review the diff, and treat blocked results as expected safety outcomes rather than command failures.

`v0.8` keeps `remove-component` prefab-only and does not expand the built-in allowlist yet. `MeshRenderer` and `MeshFilter` remain explicitly blocked with dependency-aware messages because sibling-pair safety rules are not implemented.

Scene-file structural mutation was evaluated in `v0.8` and remains out of scope. `remove-component` stays limited to `prefab` because scene-scale object graphs and cross-object blast radius need a separate safety design before write support is allowed.

Example warning output:

```text
WARN code=TAB_INDENT file_id=1000 message="tab indentation is unsupported in v0.2 field scanning"
```

Example refs warning output:

```text
WARN code=UNKNOWN_FIELD_SHAPE file_id=11400000 message="unsupported PPtr fileID"
```

The same condition appears as `status: "WARN"` in `refs --json` and still exits `0`.

Example integrity error output:

```text
GRAPH_CHECK status=ERROR blocks=6 gameobjects=2 components=4 transforms=2
ERROR code=DUPLICATE_FILE_ID file_id=900 duplicates=2
ERROR code=DUPLICATE_FILE_ID file_id=1000 duplicates=2
```

Example roundtrip output:

```text
ROUNDTRIP status=OK mode=lossless-block-copy bytes_equal=1 reparsed=1 block_sequence_equal=1 graph_check=OK line_endings=LF editor_open=NOT_CHECKED out=/tmp/check_ok.copy.prefab
```

Example scalar set output:

```text
SET status=OK file_id=1000 field=m_IsActive old=1 new=0 pre_check=OK temp_check=OK final_check=OK backup=testdata/fixtures/set_prefab.prefab.bak
SET status=BLOCKED code=MONOBEHAVIOUR_NATIVE_WRITE_BLOCKED file_id=11400000 field=m_Enabled message="native scalar writes to MonoBehaviour are blocked in v0.5"
SET status=WARN file_id=2100000 field=m_Name old=Body new="Helmet" pre_check=WARN temp_check=WARN final_check=WARN backup=testdata/fixtures/set_material.mat.bak
```

Example remove-component output:

```text
REMOVE_COMPONENT status=EXPERIMENTAL file_id=65000 class_id=65 type=BoxCollider game_object=1000 pre_check=OK temp_check=OK final_check=OK backup=/tmp/remove_component_ok.prefab.bak
```
