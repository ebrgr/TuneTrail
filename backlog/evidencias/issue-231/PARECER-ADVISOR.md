# ISSUE-231 — Parecer do Advisor

27/11/2026, Modo A sobre `git diff main...HEAD -- internal` (HEAD `1afe40c`). Opus, effort high, somente leitura, acionado pelo usuário.

**APROVADO.** Nenhuma falha crítica e nenhuma mudança de comportamento encontrada na fila durável do relay.

## Conferido

- O SQL de reivindicação, renovação, conclusão, falha de origem e leitura do conteúdo não mudou; só as assinaturas. `withIngestWriter` não repete a função, então o lote não acumula itens.
- A ordem do lote continua por vencimento e depois por ID; o orçamento de POSTs usa o mesmo tamanho.
- `RelayPayload` só traduz o `sql.ErrNoRows` da própria leitura; o worker toma o mesmo ramo nos mesmos casos. Erros de contexto ou de E/S não retiram o item.
- `RelayTarget` continua unindo `relay.ErrConfigInvalid` ao erro de validação; as mensagens dos dois erros são as mesmas.
- Nenhum outro chamador ou implementação foi afetado.
- `relayoutbox` não alcança `internal/db/sqlite`; o sqlite perdeu `catalog`, `images` e `memkv`.

## Melhorias, todas opcionais

1. **Deste diff (evidência):** `TestRelayClaimOrder` passa sem o `slices.SortFunc`. Confirmado pelo implementador; `VERIFICACAO.md` foi corrigido.
2. **Deste diff:** `internal/imageurl` está em `sqliteBase`, mas nada limita os imports dele; ele traz `imagecache` e dependências (que o sqlite já recebia via `catalog`). Sugestão: regra para os imports de `imageurl`, ou mover as constantes de tamanho. Não aplicada.
3. **Deste diff:** o comentário de `RelayPayload` não diz que `db.ErrNotFound` também aparece quando a reivindicação perdeu o bloqueio (lease, geração ou versão). Não aplicada.
4. **Anteriores, fora do escopo:** `TestWorkerPacesFailedClaims` trava até o limite do Go numa regressão; `/first-activity` responde 500 com histórico vazio; `RelayQueueStatus` devolve `sql.ErrNoRows` dentro do próprio pacote.

## Verificado pelo Advisor

`go build ./...`, `go vet` nos pacotes tocados, `go test -race` em `architecture`, `imageurl`, `relay`, `relayoutbox`, `summary` e `webui`, testes selecionados de relay no `sqlite` e no `engine`, `go list -deps`. Não repetiu o pacote `sqlite` inteiro com `-race`, o `make check` nem as demais mutações.
