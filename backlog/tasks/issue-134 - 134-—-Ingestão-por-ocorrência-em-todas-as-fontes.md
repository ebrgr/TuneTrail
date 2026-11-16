---
id: ISSUE-134
title: 134 — Ingestão por ocorrência em todas as fontes
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-16T17:06:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-133
  - ISSUE-081
documentation:
  - doc-036
  - doc-039
type: feature
ordinal: 147000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 4 do estudo da ISSUE-116 (doc-036). Ingestão grava a ocorrência ouvida em todas as fontes (ListenBrainz, Last.fm, importadores).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mesmo MBID em A/B gera uma gravação e duas ocorrências
- [ ] #2 Duas faixas 'Intro' com IDs ou posições diferentes continuam separadas
- [ ] #3 Sem desambiguador, o evento é preservado para revisão
- [ ] #4 Reimportação idempotente e repetição legítima testadas conforme a ISSUE-081
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Implementar recebimentos duráveis, conflitos preservados para revisão por perfil, idempotência por fonte/ID ou digest/ordinal, Pending na importação e retomada sem depender do staging. Reenvios indistinguíveis sem ID confiável não permitem exatamente uma vez automático. Coordenar revisão/consulta com ISSUE-083 e relay elegível com ISSUE-082.

---
<!-- COMMENTS:END -->
