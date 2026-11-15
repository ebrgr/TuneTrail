---
id: ISSUE-139
title: 139 — Páginas de gestão por tarefa e compatibilidade dos endereços antigos
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T20:04:00Z'
updated_date: '2026-11-15T20:44:00Z'
labels:
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
  - backlog/evidencias/issue-139/relatorio.md
  - backlog/evidencias/issue-139/reviewer-final.md
  - backlog/evidencias/issue-139/reviewer-complementar.md
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
- [x] #1 Cada página de tarefa contém só a sua tarefa; tarefas que não valem para o tipo não são ofertadas nem aceitas
- [x] #2 Página base vira índice com links; URLs antigas com consulta e ações levam à tarefa correspondente sem perder fuso, escuta e retorno
- [x] #3 Merge ou exclusão confirmados levam ao item que sobrou; sem sobrevivente, ao perfil
- [x] #4 Testes de sessão, origem, CSRF, ticket de uso único e escopo de perfil passam pelo roteador principal, com ticket obrigatório como em produção
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T20:13:00Z

---

Decisão do usuário em 15/11/2026: preservar endereços antigos, incluindo consultas (listen_at, history_return, artist_query, merge_query, tz), âncoras, /review e ações POST, e não só fazer a página base responder.

---

author: ebrgr
created: 2026-11-15T20:15:00Z

---

Início: branch `feat/issue-139-paginas-de-gestao-por-tarefa` criada a partir de `main`.

---

author: Codex
created: 2026-11-15T20:35:00Z

---

Verificação dos critérios #1–#4: páginas separadas e restritas por tipo, índice e endereços antigos preservados, destinos de merge normal/invertido com fallback administrativo aprovado para sobrevivente sem escutas, exclusão pessoal isolada e matriz de sessão/origem/CSRF/ticket/perfil pelo roteador principal com configuração de produção; make check aprovado com 12/12 E2E. Decisão do coordenador: manter leitura pública restrita e navegar à gestão do sobrevivente quando ele não foi ouvido no perfil atual.

---

author: Codex
created: 2026-11-15T20:38:00Z

---

Reviewer: Claude Code Opus, esforço medium, aprovou o diff final sem itens obrigatórios (reviewer-final.md); correção adicional da sugestão sobre erro POST com retorno filtrado também APROVADA (reviewer-complementar.md). make check final aprovado, incluindo 12/12 E2E; graphify atualizado pelo script AST oficial.

---

author: ebrgr
created: 2026-11-15T20:39:00Z

---

Verificação: critérios 1–4 verificados pelo roteador principal com tickets obrigatórios; páginas por tarefa, compatibilidade de URLs e POST, login e retornos seguros, isolamento de perfis, merge normal/invertido e sobrevivente não ouvido; make check aprovado com 12/12 E2E, Reviewer Opus APROVADO sem itens obrigatórios e graphify atualizado; evidências em backlog/evidencias/issue-139/relatorio.md

---

author: claude
created: 2026-11-15T20:44:00Z

---

Aceite: aprovada pelo usuário em 15/11/2026, com autorização para push e publicação.

---
<!-- COMMENTS:END -->
