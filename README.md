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

## Usage

```bash
go run ./cmd/uyaml prefab blocks testdata/fixtures/simple_prefab.prefab
```
