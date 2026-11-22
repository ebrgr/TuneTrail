# Parecer do Reviewer — ISSUE-182 (commit cc304ef, schema v18)

Escopo: migração 018, `claimNextIngest`, `ingest_retry.go` (park por claims, `ReprocessIngest`, `ReprocessAllIngest`), handler/rotas de `/admin/health`.
Base: leitura do diff e dos arquivos centrais; testes focados disparados por mim (resultado no comentário da tarefa).

## Achados
- [RISCO] internal/db/sqlite/ingest_retry.go:ReprocessAllIngest — os lotes de 500 rodam dentro de uma única transação sob o escritor: o lock só cai no commit. Medido em 116 ms para 10 mil, mas cresce com a fila (um backlog de 100 mil seguraria o escritor da ingestão por ~1 s). Aceitável; se necessário, um commit por lote (a idempotência por `parked_at` torna isso seguro).
- [RISCO] internal/db/sqlite/ingest_retry.go:ReprocessIngest — o reprocessamento unitário deixou de ler `resolution_version`/`work_version` antes de liberar o recibo e passou a um UPDATE simples fora do `withIngestWriter`. Seguro, pois um recibo estacionado não é reivindicável e o predicado `parked_at IS NOT NULL` torna a repetição um no-op, mas muda a garantia documentada na 173; registrar.
- [RISCO] db/migrations_sqlite/018_ingest_reprocess.sql — o UPDATE da migração reinicia o orçamento de claims de todos os recibos ativos abertos com tentativas (O(n) sobre o conjunto aberto, normalmente pequeno). Há guarda de downgrade; o backup do v17 é o caminho de volta.
- [OK] Teto vitalício decidido dentro do mesmo UPDATE do claim (`attempt_count-claim_attempt_base>=25` → estaciona com `lease_exhausted`, lease nula, estado `retryable`, sem emitir novo claim), respeitando o CHECK da migração 013; `ResolveNextIngest` trata a linha estacionada como progresso sem lease; alerta WARN por perfil e motivo; reprocessar todos valida que o `profile` do formulário é o autorizado, grava o ID da conta e o horário (`reprocessed_by>0`), é idempotente e perfil diferente é no-op; comentário de `warnParkedIngest` corrigido.

Sem BUG. Os riscos são de operação, não de correção.
