# ISSUE-232 — Levantamento inicial dos subsistemas

Medido em 27/11/2026 sobre a v0.79 (schema 25), com `go list -json ./...` e contagem de linhas dos arquivos `.go`.

## Tamanho (Go de produção)

42 pacotes, 41.712 linhas de produção e 34.494 de testes; frontend com 27 CSS, 19 HTML e 9 JS (7.947 linhas).

| Pacote | Linhas | % |
| --- | --- | --- |
| internal/db/sqlite | 12.685 | 30 |
| internal/adminui | 4.477 | 11 |
| engine/handlers | 3.472 | 8 |
| internal/webui | 2.092 | 5 |
| internal/cfg | 1.918 | 5 |

## internal/db/sqlite por assunto (agrupado pelo nome do arquivo, aproximado)

| Assunto | Linhas | Arquivos |
| --- | --- | --- |
| Ingestão (recibos, resolução, revisão, saúde) | 3.027 | 15 |
| Catálogo | 2.734 | 8 |
| Relay | 1.730 | 8 |
| Contas, perfis, sessões e chaves | 1.240 | 8 |
| Metadados e enriquecimento | 1.149 | 6 |
| Infraestrutura (abertura, migração, backup) | 950 | 3 |
| Escutas e estatísticas | 839 | 4 |
| Exportação e importação | 521 | 2 |
| Outros (planos de mutação, leituras por escopo, sessão root) | 495 | 3 |

## Dependências que importam

- Nenhum SQL de produção fora de `internal/db/sqlite`.
- Importam `internal/db/sqlite`: `engine` e `internal/relayoutbox`. Importam `internal/db` (interfaces): 16 pacotes.
- `internal/db/sqlite` importa: `catalog` (5 arquivos, só `BuildImageList`), `mbz` (2 arquivos), `enrichment` (3), `relay` (8) e `dedup` (3).
- `internal/relayoutbox` usa de `sqlite`: `RelayBatch`, `RelayItem`, `ErrRelayConfigInvalid`, `ErrRelaySourceInvalid`.
- `webui` importa `adminui`; `adminui` usa `mbz`, `importer`, `relay` e `ingest`.
- Funções com mais de 120 linhas em `sqlite`, `handlers` e `adminui`: 12; as maiores têm 337 e 334 linhas (`internal/adminui/actions.go` e `media.go`).
