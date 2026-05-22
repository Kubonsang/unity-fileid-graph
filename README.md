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

## Usage

```bash
go run ./cmd/uyaml prefab blocks testdata/fixtures/simple_prefab.prefab
go run ./cmd/uyaml prefab graph testdata/fixtures/graph_prefab.prefab
```

Example warning output:

```text
WARN code=TAB_INDENT file_id=1000 message="tab indentation is unsupported in v0.2 field scanning"
```
