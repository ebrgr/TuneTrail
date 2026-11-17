---
id: ISSUE-134
title: 134 — Ingestão por ocorrência em todas as fontes
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-17T16:18:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-133
  - ISSUE-081
references:
  - backlog/decisions/decision-009
documentation:
  - doc-036
  - doc-039
  - doc-040
type: feature
ordinal: 147000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 4 do estudo da ISSUE-116 (doc-036). Ingestão grava a ocorrência ouvida em todas as fontes (ListenBrainz, Last.fm, importadores). Contrato normativo: doc-040 e decision-009 (ISSUE-079, aceitas em 2026-11-17). Acrescenta (doc-040 §3–§5 e §10): adotar o envelope e o resolvedor comuns, junto da ISSUE-081; manter ocorrência ouvida, digest e ordinal, precisão temporal, Pending e retomada independente do staging; deduplicação conservadora (Igual, Diferente, Revisar) com janelas apenas para selecionar candidatos e evidência persistida; testar idempotência enquanto os recibos existem e a nova importação após a exclusão (evento novo, sem bloqueio), inclusive arquivos e lotes parcialmente excluídos: a chave de lote não bloqueia os itens excluídos e itens vivos reaproveitam seus recibos; coordenar a revisão por perfil com a ISSUE-083 e a outbox elegível com a ISSUE-082; sem ativação geral antes de ser possível resolver as pendências.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mesmo MBID em A/B gera uma gravação e duas ocorrências
- [ ] #2 Duas faixas 'Intro' com IDs ou posições diferentes continuam separadas
- [ ] #3 Sem desambiguador, o evento é preservado para revisão
- [ ] #4 Reimportação idempotente e repetição legítima testadas conforme a ISSUE-081
- [ ] #5 Reimportação após a exclusão cria evento novo; arquivo ou lote parcialmente excluído reaproveita os recibos vivos e recria só os excluídos
- [ ] #6 Ambiguidade não prova igualdade: candidatos indistinguíveis vão para Revisar com evidência e regra versionadas, sem apagar nem contar uma segunda escuta
- [ ] #7 Fontes e importadores usam o envelope comum, com precisão temporal e retomada independente do staging
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Implementar recebimentos duráveis, conflitos preservados para revisão por perfil, idempotência por fonte/ID ou digest/ordinal, Pending na importação e retomada sem depender do staging. Reenvios indistinguíveis sem ID confiável não permitem exatamente uma vez automático. Coordenar revisão/consulta com ISSUE-083 e relay elegível com ISSUE-082.

---
<!-- COMMENTS:END -->
