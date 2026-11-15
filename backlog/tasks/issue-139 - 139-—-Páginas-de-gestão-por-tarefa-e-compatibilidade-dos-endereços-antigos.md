---
id: ISSUE-139
title: 139 — Páginas de gestão por tarefa e compatibilidade dos endereços antigos
status: Aberta
assignee: []
created_date: '2026-11-15T20:04:00Z'
labels:
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
priority: high
type: feature
ordinal: 152000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte A da ISSUE-108 (avaliação do Codex em backlog/evidencias/issue-108/avaliacao-codex.md, §5). Separa a página de gestão em páginas por tarefa: /edit (apelidos, artistas, MBID), /add-listen, /image, /merge, /delete e a página da escuta; a página antiga vira índice curto. Funciona sem JavaScript e prepara os blocos para os fragmentos do recorte B. Os endereços antigos (listen_at, history_return, artist_query, merge_query, tz, /review e ações POST) continuam funcionando e levam à tarefa correspondente. A imagem da faixa continua alterando a capa do álbum e segue acessível pelo índice. Merge e exclusão confirmados levam ao item que sobrou; se nenhum sobrar, ao perfil. Rotas, allowlists, SafeReturn e return_to evoluem juntos (internal/adminui/routing.go, engine/routes.go, internal/adminweb/return.go).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cada página de tarefa contém só a sua tarefa; tarefas que não valem para o tipo não são ofertadas nem aceitas
- [ ] #2 Página base vira índice com links; URLs antigas com consulta e ações levam à tarefa correspondente sem perder fuso, escuta e retorno
- [ ] #3 Merge ou exclusão confirmados levam ao item que sobrou; sem sobrevivente, ao perfil
- [ ] #4 Testes de sessão, origem, CSRF, ticket de uso único e escopo de perfil passam pelo roteador principal, com ticket obrigatório como em produção
<!-- AC:END -->
