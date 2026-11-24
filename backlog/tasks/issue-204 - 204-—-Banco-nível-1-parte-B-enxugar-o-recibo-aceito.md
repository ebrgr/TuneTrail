---
id: ISSUE-204
title: '204 — Banco, nível 1 (parte B): enxugar o recibo aceito'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-24T00:51:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-203
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: medium
type: chore
ordinal: 217000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-047, repetições R1, R2, R3 e R5. Cada recibo aceito grava hoje o que já está em outro lugar. (1) ingest_revisions: as 5.000 revisões do ensaio são 100% idênticas à decisão do recibo (reason_codes, evidence, rule_version) e nenhuma consulta as lê; parar de gravar a revisão automática do recibo aceito e manter só as decisões de usuário e de fusões de catálogo (194 bytes por escuta). Pontos de escrita: ingest_occurrence.go, ingest_retry.go, ingest_review.go, ingest_trusted_replay.go. (2) ingest_events.listened_at é texto RFC3339 de 30 bytes que repete listens.listened_at (inteiro, 5.000 de 5.000 iguais): guardar como inteiro. (3) closed_at repete resolved_at em todos os recibos aceitos: uma coluna de tempo de decisão, depois de conferir os demais estados (estacionado, inválido, duplicado). (4) reason_codes, evidence e rule_version têm o mesmo valor em todos os recibos do caminho normal: gravar só quando fugir do padrão. Estimativa: 247 a 311 bytes a menos por escuta (de 2.382), -10% a -13%. CUIDADO: a página por que esta escuta está aqui (ISSUE-083), a fila de revisão e a central de saúde leem o recibo; não pode haver mudança visível. Medir antes e depois com backlog/evidencias/db-auditoria. Depende da ISSUE da parte A (mesmos arquivos) e da decisão do usuário (doc-047 §7, decisões 1 e 2).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Recibo aceito não grava mais revisão automática; decisões de usuário e de fusões continuam gravadas
- [ ] #2 Horário do recibo como inteiro e uma só coluna de tempo de decisão, com os estados não aceitos conferidos
- [ ] #3 Resultado padrão do caminho normal não é gravado; páginas de saúde, revisão e por que esta escuta está aqui sem mudança
- [ ] #4 Bytes por escuta medidos antes e depois com 5.000 escutas pela API, e a economia registrada
<!-- AC:END -->
