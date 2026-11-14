---
id: ISSUE-075
title: 075 — Fragmentos parciais, View Transitions e Speculation Rules
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T15:10:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-073
  - ISSUE-107
priority: medium
type: feature
ordinal: 75000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 8, ISSUE-056 e doc-010 §3.5. Sem framework; JS próprio e mínimo. A busca instantânea e a edição sem recarregar foram antecipadas para a ISSUE-107; esta tarefa generaliza o mesmo mecanismo de fragmentos para o resto da interface.

Passos:
1. Estender o `?partial=1` da ISSUE-107 aos rankings, ao histórico e às páginas de mídia.
2. Filtros e paginação: o script intercepta formulários GET, troca o `<main>` e mantém a URL com `history.pushState`.
3. `@view-transition { navigation: auto; }` para suavizar as navegações completas restantes, e Speculation Rules para pré-carregar links do mesmo perfil.
4. Gráficos com detalhe por `<title>` e tooltip CSS, mantendo as tabelas acessíveis.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Todas as jornadas continuam funcionando com JavaScript desligado
- [ ] #2 JS total abaixo de 25 KB comprimido
- [ ] #3 ISSUE-056 encerrada com as decisões registradas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:56:00Z

---

Busca com sugestões (passo e critério 3) movida para a ISSUE-107, após a comparação com o Koito React (backlog/evidencias/ui-responsiva/comparacao-koito-react.md). Esta tarefa passa a depender da 107 para reaproveitar o mecanismo de fragmentos.

---

author: claude
created: 2026-11-14T15:10:00Z

---

ISSUE-056 encerrada como estudo; dependência retirada.

---
<!-- COMMENTS:END -->
