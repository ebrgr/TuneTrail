# ISSUE-231 — Verificação (27/11/2026)

## URLs de imagem

`internal/imageurl/imageurl_test.go` traz os endereços literais. O mesmo teste passou antes contra a função original `catalog.BuildImageList`, por meio de um apontamento temporário, e depois contra `imageurl.List`.

## Mutações

Cada alteração foi aplicada sozinha e desfeita depois.

| Mutação | Resultado |
| --- | --- |
| Histórico vazio sem `db.ErrNotFound` | detectada: `TestEmptyHistoryIsNotAFirstActivityError`, `TestRewindSQLiteAndSummaryAPI` |
| Origem ausente do relay com outro erro | detectada: `TestRelayHistoryDeletion` |
| Worker não reconhece origem ausente | detectada: `TestWorkerBisectsAndSettles` |
| Worker não reconhece configuração inválida | detectada só pelo limite de 10 minutos: o teste existente fica esperando um canal |
| Lista de páginas registra histórico vazio como falha | detectada: `TestEmptyHistoryIsNotAFirstActivityError` |
| Rewind falha com histórico vazio | detectada: `TestEmptyHistoryIsNotAFirstActivityError`, `TestRewindSQLiteAndSummaryAPI` |
| Resumo falha com histórico vazio | detectada: `TestSummaryGenericAndAllTimeIntervals` |
| Ordem do lote invertida | detectada: `TestRelayClaimOrder` |
| Lote sem ordenação por vencimento | detectada: `TestRelayClaimOrder` |
| Empate sem desempate por ID | detectada: `TestRelayClaimOrder` |
| URL de imagem sem perfil | detectada: `TestList` |

Na primeira rodada, as duas mutações de ordem do lote **não** eram detectadas: nenhum teste conferia a ordem. `TestRelayClaimOrder` foi criado para isso. Duas outras mutações da primeira rodada apenas quebravam a compilação e foram refeitas.

## Detector de corridas

`CGO_ENABLED=1 go test -race -count=1 -timeout 30m`, todos `ok`: `internal/architecture`, `internal/catalog`, `internal/db/sqlite` (1391 s), `internal/imageurl`, `internal/relay`, `internal/relayoutbox`, `internal/summary`, `internal/webui`.

## Dependências do armazenamento

`go list -deps ./internal/db/sqlite` deixou de incluir `internal/catalog`, `internal/images` e `internal/memkv`.

## Teste de arquitetura

As exceções passaram de 13 para 8. `internal/imageurl` entrou na lista `sqliteBase`.
