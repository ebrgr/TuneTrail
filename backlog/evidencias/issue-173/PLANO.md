# ISSUE-173 — Retry e fila de falhas permanentes do resolvedor (plano para o Advisor)

Data: 2026-11-21. Base: v0.53. Fase 1 (plano), sem código. Revisão do Advisor (effort high) é obrigatória antes de implementar: concorrência, lease e migração.

## Decisões do usuário (21/11/2026)
- Retry: até 5 tentativas automáticas; esperas de 1, 2, 4 e 8 minutos entre elas.
- Onde ver: seção da central de saúde (`/admin/health`).
- Quem reprocessa: **dono do perfil e root** (o AC #2 original dizia só root; reescrito).
- Telas: a mesma central compartilha a fila de falhas (173) e a de revisão (181), em seções separadas, com **duas ou três colunas no desktop e uma sobre a outra no celular**, como o padrão do sistema. O reprocessamento é uma operação reaproveitada pelas duas.

- Estacionamento: **opção A** (colunas `parked_at` e `park_reason`, mantendo `state='retryable'`; sem reconstruir `ingest_events`), decidida em 21/11/2026.
- Advisor: não acionado até este ponto. O AC #4 (parecer do Advisor, effort high) segue pendente e não pode ser marcado sem o parecer.

## Situação atual (lida no código)
- `claimNextIngest` (`internal/db/sqlite/ingest_resolve.go:88`) pega recibos `mode='active'`, `closed_at IS NULL`, `state IN ('received','retryable')`, e incrementa `attempt_count` junto com o lease de 120 s.
- `retryIngest` (`:162`) devolve a `retryable` com `next_attempt_at = agora + 1 min`, sem olhar o contador. Falhas de leitura do instante ou do snapshot também só reagendam.
- `ingest_events.state` tem CHECK com `received, pending_identity, pending_duplicate, invalid, accepted, duplicate, retryable`. `closed_at` só pode existir em `invalid`, `accepted`, `duplicate`. `invalid` exige `rule_version` e ao menos um `reason_codes`.
- Índices parciais de fila: `idx_ingest_queue` e `idx_ingest_active_queue` (`state, next_attempt_at, received_at`).
- Esquema atual 12; há 13 arquivos em `db/migrations_sqlite`.

## Proposta
1. **Esgotamento.** Em `retryIngest`, usar `r.attempts` (já lido no claim): tentativas 1 a 4 reagendam com 1, 2, 4 e 8 min; a 5ª falha **estaciona** o recibo. Estacionar nunca descarta o recibo, a chave de idempotência, o snapshot original nem a contagem.
2. **Erros determinísticos.** Instante ilegível e snapshot JSON inválido não melhoram com o tempo: encerrar como `invalid` com `reason_codes` estável e `rule_version`, sem consumir 5 tentativas. Classificar por tipo de erro, não por texto.
3. **Como estacionar (decisão para o Advisor).**
   - Opção A: nova coluna `parked_at INTEGER` e `park_reason TEXT` (ALTER TABLE ADD COLUMN, sem reconstruir a tabela), mantendo `state='retryable'`; o claim passa a exigir `parked_at IS NULL`. Migração 013 pequena e reversível por backup.
   - Opção B: novo estado `failed` no CHECK. Exige reconstruir `ingest_events`, com triggers e FKs; maior risco de migração.
   - **Decidido pelo usuário: opção A.**
4. **Reprocessar.** Operação única `ReprocessIngest(profile, receipt)`: com CAS em `resolution_version` e `work_version`, volta a `retryable`, `next_attempt_at = agora`, limpa `parked_at`, e zera o orçamento de tentativas sem apagar o histórico (guardar `total_attempts` ou registrar em `reason_codes`). Idempotente: dois cliques seguidos reprocessam uma vez. Reaproveita `finishOccurrence` (mesma resolução transacional).
5. **Alerta.** Log WARN com motivo estável, no máximo uma vez por perfil e motivo por minuto (padrão do relay da 082), e contagem visível no topo da seção.
6. **Tela.** Seção "Falhas permanentes" na central de saúde: recibo, fonte, motivo, tentativas, quando estacionou, botão Reprocessar (POST, CSRF). Acesso do dono ao próprio perfil e root a todos, sem expor snapshot bruto, tokens nem chave de idempotência. Leitura limitada (LIMIT) e indexada. A central passa a organizar as seções em colunas no desktop (2 a 3) e empilhadas no celular, sem rolagem horizontal.
7. **R-b.** Com o interruptor mestre ligado, erro genérico antes de o modo do perfil ser conhecido segue pelo caminho legado, em vez de 500. Erros de armazenamento e BUSY continuam em 503 com `Retry-After`. Testar por classe de erro (`engine/handlers/lbz_submit_listen.go`, bloco de admissão).

## Pontos para o Advisor
- Concorrência: estacionar e reprocessar contra o claim com lease; recibo reprocessado durante uma tentativa em voo; crash entre estacionar e liberar o lease.
- Migração 013 (opção A ou B), compatibilidade com v0.52 e backup `.pre-*.bak`.
- Garantia de que nenhum recibo confirmado com 200 se perde, inclusive após reinício.
- Classificação de erros transitório vs determinístico, e o risco de classificar errado como determinístico.
- Fila indexada e limitada; custo da consulta da seção.
- Acoplamento com a 181 (mesma operação de reprocessar e mesma central) e ordem de entrega.

## Verificação prevista
- Ensaio sintético com falha persistente (trigger) e transitória, esperas 1/2/4/8 min (relógio injetável nos testes), crash e retomada por lease, sem escuta confirmada perdida.
- Testes de isolamento entre perfis na seção e na ação de reprocessar, CSRF, e ausência de segredos.
- `make check`.
