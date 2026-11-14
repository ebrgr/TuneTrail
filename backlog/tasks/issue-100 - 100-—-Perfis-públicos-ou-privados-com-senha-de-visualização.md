---
id: ISSUE-100
title: 100 — Perfis públicos ou privados com senha de visualização
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
labels:
  - admin
  - ui
milestone: m-14
dependencies:
  - ISSUE-074
priority: high
type: feature
ordinal: 100000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ROADMAP item 2 e doc-010 §7. Senha simples por perfil só para visualização; somente a conta de administração apaga dados e cria ou remove perfis. Revisão obrigatória do Advisor antes de implementar (AGENTS.md: mensageria/concorrência/criptografia → @Opus effort=high no Claude, @Sol effort=medium no Codex).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Senha armazenada com hash adequado e limite de tentativas
- [ ] #2 Perfis privados não vazam em páginas, API, widget nem busca
- [ ] #3 Somente admin cria, remove e apaga dados (testes de permissão)
- [ ] #4 Parecer do Advisor registrado
<!-- AC:END -->
