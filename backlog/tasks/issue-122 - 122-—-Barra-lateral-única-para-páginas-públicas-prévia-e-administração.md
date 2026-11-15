---
id: ISSUE-122
title: 122 — Barra lateral única para páginas públicas, prévia e administração
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T00:57:00Z'
updated_date: '2026-11-15T01:06:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
references:
  - backlog/evidencias/issue-110/advisor-inicial.md
  - backlog/evidencias/issue-122/verificacao.md
  - backlog/evidencias/issue-122/reviewer.md
type: chore
ordinal: 131000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor na ISSUE-110 (backlog/evidencias/issue-110/advisor-inicial.md) e decisão do usuário em 14/11/2026 de separar a refatoração. Hoje há duas barras: partial/sidebar (público e prévia, assets/templates/public/page.html) e admin/sidebar (assets/templates/admin/shell.html, com SVG inline e máscaras CSS no avatar). Unificar sem mudar comportamento: struct webassets.Nav com método Nav() em Page, LivePage e no view do admin; assets/templates/shared/nav.html montado nos dois conjuntos pelo manifesto (sidebar, avatar, language-picker); funções icon, initials e avatarIcon em webassets para os dois conjuntos. A busca continua diálogo no público e link na administração; o botão de configurações continua abrindo o diálogo de cada lado. A ISSUE-110 depende desta.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Barra lateral definida uma única vez em assets/templates/shared/nav.html
- [x] #2 Administração usa a função icon; sem SVG inline nem máscaras no avatar
- [x] #3 HTML equivalente à referência, salvo diferenças justificadas nos ícones da administração
- [x] #4 Capturas sem diferença visual e make check aprovado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-15T00:57:00Z

---

Início: branch `chore/issue-122-barra-lateral-unica-para-paginas` criada a partir de `main`.

---
<!-- COMMENTS:END -->
