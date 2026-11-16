---
id: ISSUE-135
title: >-
  135 — Consultas, edição, exclusão e exportação por ocorrência, com ranking por
  gravação ou lançamento
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-16T17:06:00Z'
labels:
  - catalogo
  - ui
milestone: m-13
dependencies:
  - ISSUE-133
  - ISSUE-134
  - ISSUE-092
documentation:
  - doc-036
  - doc-039
type: feature
ordinal: 148000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 5 do estudo da ISSUE-116 (doc-036). Consultas, edição, fusão, exclusão e exportação passam a usar ocorrências. Decisão do usuário: a tela de faixas mais ouvidas ganha uma chave 'por gravação' / 'por lançamento', com padrão por gravação.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Chave 'por gravação' / 'por lançamento' no ranking de faixas, padrão por gravação; soma por lançamento igual ao total por gravação
- [ ] #2 Rankings e novidades por perfil seguem o contrato da ISSUE-132
- [ ] #3 Editar ou remover um lançamento não muda outro; exportar e reimportar preserva ocorrência e proveniência
- [ ] #4 Duas escutas simultâneas de perfis distintos sobrevivem; planos de consulta medidos em fixtures grandes
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Reconciliação por lançamento inclui o balde sem lançamento; pendências não contam antes de resolução. Exportação versionada e cursor por event_id preservam eventos simultâneos; edição/exclusão/fusão não deduplicam por gravação/segundo. Manter filtros de perfil e URLs antigas de ocorrência.

---
<!-- COMMENTS:END -->
