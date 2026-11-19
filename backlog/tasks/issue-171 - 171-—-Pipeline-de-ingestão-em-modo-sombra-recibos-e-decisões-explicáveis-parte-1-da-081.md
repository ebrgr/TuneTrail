---
id: ISSUE-171
title: >-
  171 — Pipeline de ingestão em modo sombra: recibos e decisões explicáveis
  (parte 1 da 081)
status: Aberta
assignee: []
created_date: '2026-11-19T16:18:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-080
documentation:
  - doc-043
priority: high
type: feature
ordinal: 184000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: divisão da ISSUE-081 em 2026-11-19, por decisão do usuário, a partir do estudo do Advisor (doc-043). Esta é a PARTE 1: grava o recibo e a decisão de deduplicação ao lado do caminho atual, SEM mudar nenhuma contagem de escutas, para medir a taxa real de Revisar no banco verdadeiro antes da ativação (ISSUE-081, parte 2). Fica antes da ISSUE-133 porque cabe nas guardas da v4: confirmado pelo usuário e pela leitura da migração 004 que a v4 aceita duplicate_decision same/different/review com event_id NULL (004:56 e :62; state só received, pending_identity, pending_duplicate e invalid; closed_at só para invalid; nenhum CHECK liga duplicate_decision a event_id). Consequência: linhas de sombra nunca ficam fechadas nem aceitas; o estado delas permanece received ou pending_*.

Escopo (commits ISSUE-171:): (1) internal/ingest/envelope.go: tipos, snapshot saneado, digest e precisão de tempo (second ou estimated); (2) internal/db/sqlite/ingest_receipts.go: ReceiveBatch em transação curta (BEGIN IMMEDIATE) com os índices únicos de idempotência da v4; (3) internal/dedup: Compare(policy, recibo, candidato) devolve Igual, Diferente ou Revisar com reason_code, evidência e rule_version; janela de ±2 s só seleciona candidatos, nunca prova identidade; (4) handler ListenBrainz em modo sombra: grava o recibo antes do caminho atual; o 200 só sai depois do commit do recibo e 503 em falha de disco; MusicBrainz e HTTP fora da trava; playing_now continua só em memória, sem recibo; (5) importadores em sombra: digest do arquivo mais ordinal, contador Pending e retomada por cursor (coordenar com a ISSUE-134); (6) relatório da taxa de Revisar sobre cópia de banco real, sem tocar o banco real.

Decisões adotadas (comentário do Advisor na ISSUE-081): mesma faixa duas vezes no mesmo segundo, pela API e sem ID externo, é registrada como Revisar (contrato §5); escuta sem listened_at é aceita com a hora de chegada marcada estimated; o modo Revisar nunca altera contagens nesta parte. Fora do escopo: criar evento ou escuta nova, outbox (082), tela de revisão (083), remover a escrita direta em listens, mexer nas guardas da v4 ou criar migração (se a parte 1 exigir migração, parar e registrar). O bug do mbid_mapping é a ISSUE-170.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 O recibo de cada escuta e a decisão (same, different ou review) com reason_code, evidência e rule_version são gravados em ingest_events sem alterar nenhuma contagem de escutas; o caminho atual continua sendo a fonte da verdade
- [ ] #2 O 200 da API só sai depois do commit do recibo e falha de disco dá 503, sem perder o lote; playing_now continua só em memória
- [ ] #3 Testes de concorrência (envios iguais de conexões distintas disputando a mesma chave) e de repetições legítimas, incluindo mesma faixa no mesmo segundo, fuso e horário de verão
- [ ] #4 Relatório da taxa de Revisar sobre cópia de banco real, sem tocar o banco real, registrado em backlog/evidencias/issue-171/
- [ ] #5 Confirmado em teste que linhas de sombra com duplicate_decision same e event_id NULL são aceitas pela v4 sem nova migração
<!-- AC:END -->
