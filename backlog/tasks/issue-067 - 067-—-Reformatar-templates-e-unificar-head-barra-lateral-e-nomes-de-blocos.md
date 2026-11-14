---
id: ISSUE-067
title: 067 — Reformatar templates e unificar head, barra lateral e nomes de blocos
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-066
priority: high
type: chore
ordinal: 67000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 fase 2 e §Organização dos templates.

Passos:
1. Uma tag estrutural por linha, indentação de 2 espaços, atributos longos quebrados; `{{-`/`-}}` onde a quebra mudaria espaço entre elementos em linha.
2. Um único `partials/head.html` para prévia, páginas reais e administração; barra lateral única.
3. Renomear blocos para `page/...`, `partial/...`, `admin/...`.
4. `scripts/check_web_style.py` em `make ui.check`: nenhuma linha acima de 160 caracteres em `templates/` e `assets/css/`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 HTML renderizado equivalente à referência após normalizar espaços
- [ ] #2 Nenhuma linha acima de 160 caracteres em templates/ (verificado no ui.check)
- [ ] #3 Head e barra lateral definidos uma única vez
- [ ] #4 Capturas sem diferença visual nas seis larguras
<!-- AC:END -->
