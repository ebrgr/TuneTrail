# Parecer do Reviewer — ISSUE-182 (commit cc304ef, schema v18)

Escopo: migração 018, `claimNextIngest`, `ingest_retry.go` (park por claims, `ReprocessIngest`, `ReprocessAllIngest`), handler/rotas de `/admin/health`.
Base: leitura do diff e dos arquivos centrais; testes focados disparados por mim (resultado no comentário da tarefa).

## Achados
- [RISCO] internal/db/sqlite/ingest_retry.go:ReprocessAllIngest — os lotes de 500 rodam dentro de uma única transação sob o escritor: o lock só cai no commit. Medido em 116 ms para 10 mil, mas cresce com a fila (um backlog de 100 mil seguraria o escritor da ingestão por ~1 s). Aceitável; se necessário, um commit por lote (a idempotência por `parked_at` torna isso seguro).
- [RISCO] internal/db/sqlite/ingest_retry.go:ReprocessIngest — o reprocessamento unitário deixou de ler `resolution_version`/`work_version` antes de liberar o recibo e passou a um UPDATE simples fora do `withIngestWriter`. Seguro, pois um recibo estacionado não é reivindicável e o predicado `parked_at IS NOT NULL` torna a repetição um no-op, mas muda a garantia documentada na 173; registrar.
- [RISCO] db/migrations_sqlite/018_ingest_reprocess.sql — o UPDATE da migração reinicia o orçamento de claims de todos os recibos ativos abertos com tentativas (O(n) sobre o conjunto aberto, normalmente pequeno). Há guarda de downgrade; o backup do v17 é o caminho de volta.
- [OK] Teto vitalício decidido dentro do mesmo UPDATE do claim (`attempt_count-claim_attempt_base>=25` → estaciona com `lease_exhausted`, lease nula, estado `retryable`, sem emitir novo claim), respeitando o CHECK da migração 013; `ResolveNextIngest` trata a linha estacionada como progresso sem lease; alerta WARN por perfil e motivo; reprocessar todos valida que o `profile` do formulário é o autorizado, grava o ID da conta e o horário (`reprocessed_by>0`), é idempotente e perfil diferente é no-op; comentário de `warnParkedIngest` corrigido.

Sem BUG. Os riscos são de operação, não de correção.

## Reauditoria após as correções (commit cea899b)

Escopo: só o que mudou desde cc304ef (`resolveNextIngest`/`resolveIngestClaim`, `requeueParkedIngest`, testes). Item 3 do Advisor (commit por lote) ignorado por decisão do usuário. Testes disparados por mim; resultado no comentário da tarefa.

- [RISCO] internal/db/sqlite/ingest_resolve.go:resolveIngestClaim — depois de `verifyOccurrence`, `proofCtx.Err()` passou a abortar a tentativa. Antes, qualquer falha do MusicBrainz (inclusive timeout de 20 s) devolvia prova vazia e a escuta seguia com identidade degradada; agora só o **timeout** reagenda o recibo (1/2/4/8 min e estaciona), enquanto erros rápidos (recusa de conexão, 5xx) continuam degradando. Mudança de comportamento não pedida e com classificação inconsistente: um MusicBrainz lento atrasa e pode estacionar escutas, um MusicBrainz fora do ar não. Decidir: voltar a degradar no timeout (recomendado, preserva a ingestão sem o MusicBrainz) ou aceitar a nova regra e documentá-la.
- [RISCO] ingest_retry.go:requeueParkedIngest — a revisão de reprocessamento usa `decision='review'` para caber no CHECK; qualquer leitor que liste decisões por `ingest_revisions` (por exemplo "por que esta escuta está aqui") passará a ver uma linha `review/reprocessed`. Inofensivo, mas semanticamente imprecisa.
- [RISCO] resolveNextIngest — cancelamento do worker deixa a lease para recuperação e consome um claim a cada reinício; irrelevante frente ao teto de 25.
- [OK] Prazo por tentativa de 100 s (lease 120 s) cobre toda a resolução; a gravação do retry usa contexto próprio de 5 s; `rejectIngest` agora devolve o erro e o `retryIngest` central o trata; `lease_exhausted` recebe 5 claims após reprocessar (cálculo `attempt_count-20`), `retry_exhausted` recebe o orçamento cheio; revisão `reprocessed` (id da conta, `resolution_version+1`) e UPDATE leem o mesmo conjunto (CTE materializada), então unitário e em massa geram uma linha por recibo.

Sem BUG executado. O primeiro risco precisa de decisão do usuário.
