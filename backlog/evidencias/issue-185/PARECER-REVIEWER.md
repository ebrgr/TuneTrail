# Parecer do Reviewer — ISSUE-185 (commit 2560826, schema v20)

Escopo: migração 020, `ingest_discard.go`, rota `/admin/health/discard`, pré-visualização e confirmação na central.
Base: leitura do diff e dos arquivos centrais; testes focados disparados por mim (resultado no comentário da tarefa).

## Achados
- [BUG provável, não executado] internal/db/sqlite/ingest_discard.go:discardIngestRow + db/migrations_sqlite/020_ingest_discards.sql — `ingest_discards` tem chave primária `(profile_id, receipt_id)` e o INSERT de rastro não trata conflito. O recibo de revisão de reenvio de ID externo confiável (`reviewTrustedReplay`) tem ID **determinístico** (UUID SHA1 do recibo original), então depois de descartado ele é recriado por um novo reenvio com o mesmo ID. Descartar de novo viola a chave primária, o erro SQL sobe sem tratamento (500) e o lote inteiro é desfeito, deixando o recibo indescartável. Correção: `INSERT ... ON CONFLICT(profile_id,receipt_id) DO UPDATE` (atualiza quem/quando) ou guardar um contador/histórico; teste com reenvio do mesmo ID, descarte, novo reenvio e novo descarte.
- [RISCO] ingest_discard.go — lotes de 25 com confirmação a cada lote: 10 mil estacionados exigem 400 confirmações. A segurança da confirmação por digest é correta, mas a operação em massa real fica impraticável. Sugestão: confirmação única com contagem digitada (ou o digest do conjunto total) e lotes maiores no servidor (como o `reprocess-all` da 182 usa 500).
- [RISCO] a medição de 2,5 s para 10 mil recibos é em 400 transações; cada lote segura o escritor por poucos ms, então não afeta o ACK. Sem problema.
- [RISCO] `ingest_discards` não tem política de retenção; cada linha é pequena (IDs e códigos), mas cresce sem limite; alinhar com a retenção dos recibos numa issue futura.
- [OK] escopo: só `event_id IS NULL` e estados `received/retryable/pending_identity/pending_duplicate/invalid` sem lease ativa; aceitos, duplicados e ligados a escuta ficam protegidos e orientam para a exclusão de escuta; sem evento não há intenção de relay (a outbox é por `event_id`); CAS por `resolution_version`, `work_version`, estado e `parked_at`; o descarte do lote é atômico (qualquer divergência desfaz tudo); POST + CSRF + confirmação; rastro sem snapshot nem corpo bruto; FKs em cascata removem observações e revisões, preservando só o rastro.

Um provável bug (reenvio de ID confiável). Sem outros achados bloqueantes.
