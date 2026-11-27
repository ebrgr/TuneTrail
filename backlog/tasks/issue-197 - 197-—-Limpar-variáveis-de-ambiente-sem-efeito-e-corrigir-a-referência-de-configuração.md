---
id: ISSUE-197
title: >-
  197 — Limpar variáveis de ambiente sem efeito e corrigir a referência de
  configuração
status: Aberta
assignee: []
created_date: '2026-11-23T16:32:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - config
  - docs
dependencies: []
references:
  - backlog/evidencias/issue-127/ESTUDO.md
type: chore
ordinal: 210000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: auditoria de variáveis de ambiente da ISSUE-127 (backlog/evidencias/issue-127/ESTUDO.md §7.3). Variáveis lidas sem efeito ou documentação que não corresponde ao código. (1) ALLOWED_HOSTS: o motor só registra que não é mais usada (engine/engine.go:225) e a referência nem a cita. (2) ENABLE_FULL_IMAGE_CACHE: só aparece no mapa de nomes (internal/cfg/env_alias.go); nenhum código a lê. (3) LOGIN_GATE: lida em cfg.go e documentada como «não mostra estatísticas sem login», mas LoginGate() não tem chamador; a proteção hoje é por perfil. Estudar o comportamento atual e decidir com o usuário: corrigir a referência ou remover a variável. (4) A referência de configuração não cita os nomes de variável RELAY_ALLOW_HTTP e RELAY_ALLOW_PRIVATE_NETWORK, só as chaves TOML do relay. Manter ENABLE_LBZ_RELAY, LBZ_RELAY_URL e LBZ_RELAY_TOKEN (associação única legada) até a ISSUE-164.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão do usuário registrada sobre LOGIN_GATE (corrigir a referência ou remover), após estudo do comportamento atual
- [ ] #2 A leitura de ALLOWED_HOSTS e ENABLE_FULL_IMAGE_CACHE é removida sem mudar a subida do servidor; testes de env_alias e cfg atualizados
- [ ] #3 A referência de configuração documenta RELAY_ALLOW_HTTP e RELAY_ALLOW_PRIVATE_NETWORK e não promete comportamento inexistente
- [ ] #4 LOGIN_GATE removida do código e da referência de configuração, com aviso para quem ainda a define
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-24T00:51:00Z

---

23/11/2026: a decisão sobre aposentar o bootstrap de relay por variáveis de ambiente (ENABLE_LBZ_RELAY, LBZ_RELAY_URL, LBZ_RELAY_TOKEN) e a tabela legacy_relay_binding foi separada na ISSUE-209.

---

author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026 (C2): remover LOGIN_GATE e corrigir a referência de configuração.

---
<!-- COMMENTS:END -->
