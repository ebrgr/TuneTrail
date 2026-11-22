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

## Reauditoria após as correções (commit 6981a53)

Escopo: só o que mudou desde 2560826 (`ingest_discard.go`, migração 020, handler). Testes disparados por mim; resultado no comentário da tarefa.

- [OK] BUG resolvido: `ingest_discards` ganhou `id AUTOINCREMENT` (uma linha por descarte, índice por recibo), e o CAS passou a incluir `received_at` e a "geração" do recibo (último id de descarte), de modo que um recibo recriado com o mesmo ID, as mesmas versões e o mesmo segundo é distinguido. A v20 ainda está só na branch (a v0.59 é v19), então editar a migração é seguro.
- [OK] Operação em massa: uma confirmação cobre o conjunto elegível inteiro (digest sobre o conjunto ordenado, sem expor IDs no formulário); o servidor executa lotes de até 500, relendo e comparando o conjunto restante com a seleção original antes de cada lote; qualquer divergência devolve conflito e preserva os lotes já concluídos, com mensagem explícita na tela.
- [OK] Retenção: 90 dias, com limpeza de até 500 linhas vencidas do perfil a cada descarte bem-sucedido.
- [RISCO] discardIngestSet relê todo o conjunto restante a cada lote (custo O(n²/500)): 20 transações para 10 mil leem 200 mil linhas (2,8 s medidos), mas uma fila de centenas de milhares de recibos ficaria lenta; aceitável hoje, a medir de novo se as filas crescerem.
- [RISCO] qualquer recibo novo que entre no conjunto elegível durante um descarte longo (nova falha, por exemplo) provoca conflito e interrompe o descarte nos lotes seguintes (seguro, mas exige reabrir a página).

Sem BUG. Os riscos são de escala e usabilidade.
