---
id: ISSUE-124
title: 124 — Edição de faixas sem recarregar a página
status: Aberta
assignee: []
created_date: '2026-11-15T02:15:00Z'
updated_date: '2026-11-15T02:15:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-107
references:
  - backlog/evidencias/issue-107/advisor-inicial.md
type: feature
ordinal: 133000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: divisão da ISSUE-107 por decisão do usuário em 14/11/2026, seguindo o parecer do Advisor (backlog/evidencias/issue-107/advisor-inicial.md). Na gestão de faixa (assets/templates/admin/manage.html), cada ação refaz a página inteira (POST + 303 + GET de ~36 KB), volta ao topo e perde o foco. Passos: (1) sem JS, o 303 aponta para a âncora da seção editada; (2) carregadores e templates nomeados por seção (apelidos, artistas, metadados, escutas), reaproveitados pela página completa; (3) com JS (enhance.js), formulários data-enhance enviados por fetch com X-TuneTrail-Partial; o servidor devolve só a seção, com novo submission_token em X-TuneTrail-Submission que o JS aplica a todos os formulários; envios serializados; 409 recarrega; alias-primary atualiza h1 e title; aviso de escutas consumido no fragmento; (4) comboboxes de artista e destino de mesclagem por fragmento GET, sem criar ticket; (5) lista de escutas carregada à parte. O envio de capa (multipart) continua recarregando a página (decisão do usuário).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Adicionar/remover apelido ou artista atualiza só a seção, sem perder rolagem e foco
- [ ] #2 Sem JavaScript, as ações voltam à âncora da seção editada
- [ ] #3 Fragmentos respeitam sessão, CSRF, token de uso único, permissões e perfil (testes)
- [ ] #4 Consultas de artista e mesclagem por fragmento não criam ticket de envio
- [ ] #5 JS novo abaixo de 4 KB comprimido; tempos antes/depois registrados
<!-- AC:END -->
