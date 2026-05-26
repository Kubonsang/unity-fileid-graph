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

## Usage

```bash
go run ./cmd/uyaml prefab blocks testdata/fixtures/simple_prefab.prefab
go run ./cmd/uyaml prefab graph testdata/fixtures/graph_prefab.prefab
go run ./cmd/uyaml prefab check testdata/fixtures/check_ok.prefab
go run ./cmd/uyaml prefab roundtrip testdata/fixtures/check_ok.prefab --out /tmp/check_ok.copy.prefab
```

Example warning output:

```text
WARN code=TAB_INDENT file_id=1000 message="tab indentation is unsupported in v0.2 field scanning"
```

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
