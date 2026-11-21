# Parecer do Reviewer — ISSUE-173 (commit b881df9)

Escopo: diff `main...HEAD` (migração 013, retry/park, rejectIngest, ReprocessIngest, central de saúde, R-b).
Verificação: `go test -race` em `internal/db/sqlite`, `internal/adminui`, `engine/...` — todos OK; Codex reportou `make check` 13/13.

## Achados
- [RISCO] internal/db/sqlite/ingest_retry.go:58 — o alerta WARN é limitado por perfil+motivo por 1 minuto, mas o comentário diz "lifetime of the worker". Ajustar o comentário (não afeta comportamento).
- [RISCO] internal/db/sqlite/ingest_retry.go:ReprocessIngest — não há trilha de auditoria de quem reprocessou (apenas zera park e incrementa `work_version`). Aceitável nesta issue; considerar registrar revisão na 181.
- [RISCO] engine/handlers/lbz_submit_listen.go:223 — `Retry-After: 1` agora também é enviado ao modo legado (antes só ativo). Coerente com R-b (modo desconhecido), mas é mudança de cabeçalho no legado.
- [OK] Backoff 1/2/4/8 min e park na 5ª falha; lease recuperada retoma a tentativa (`retry_attempt_base`); claim exige `parked_at IS NULL`; CAS por `resolution_version`/`work_version`/lease; erros determinísticos → `invalid` com reason_code e rule_version; POST+CSRF e isolamento por perfil; migração aditiva com guarda de downgrade.

Sem BUG obrigatório. Matemática/lease/migração tocam mensageria: **AC #4 segue pendente do Advisor (@Opus effort=high), acionado pelo usuário.**
