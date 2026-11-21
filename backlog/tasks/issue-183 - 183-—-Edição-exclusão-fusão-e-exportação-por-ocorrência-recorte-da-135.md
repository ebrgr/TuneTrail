---
id: ISSUE-183
title: 183 — Edição, exclusão, fusão e exportação por ocorrência (recorte da 135)
status: Aberta
assignee: []
created_date: '2026-11-21T22:02:00Z'
labels:
  - catalogo
milestone: m-13
dependencies:
  - ISSUE-135
type: feature
ordinal: 196000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte da ISSUE-135 (decisão do usuário em 21/11/2026). Edição, exclusão, fusão e exportação/reimportação passam a operar por ocorrência (doc-036, doc-039, decision-008): editar ou remover um lançamento não altera outro; fusão não deduplica por gravação/segundo; exportação versionada com cursor por event_id preserva eventos simultâneos, ocorrência e proveniência. Entradas obrigatórias da ISSUE-133 (comentário #2 da 135): merge reversível de gravações para ocorrências equivalentes em lançamentos distintos, reimportação e compatibilidade integrada do corte, comparação HTTP do AC3 versionada e reaplicada aos leitores ativados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Editar ou remover um lançamento não muda outro; fusão não deduplica eventos por gravação/segundo
- [ ] #2 Exportar e reimportar preserva ocorrência e proveniência; exportação versionada com cursor por event_id
- [ ] #3 Duas escutas simultâneas de perfis distintos sobrevivem; planos de consulta medidos em fixtures grandes
<!-- AC:END -->
