---
id: ISSUE-200
title: 200 — Mostrar a chave do conf.toml em cada campo da configuração do servidor
status: Aberta
assignee: []
created_date: '2026-11-23T16:32:00Z'
updated_date: '2026-11-23T16:33:00Z'
labels:
  - ui
  - config
milestone: m-10
dependencies:
  - ISSUE-127
references:
  - backlog/evidencias/issue-127/ESTUDO.md
type: enhancement
ordinal: 213000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-127. Cada um dos 12 campos tem uma chave no conf.toml (por exemplo logging.level, subsonic.params), provado pelos testes da 127 (cfg.TOMLKey). Exibir essa chave dentro dos detalhes recolhidos do campo, ao lado dos valores salvo, efetivo e em execução, para que quem edita o arquivo à mão encontre a chave certa. Funciona sem JavaScript, texto traduzido e teste que confira a chave de todos os campos.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A chave do conf.toml de cada um dos 12 campos aparece nos detalhes do campo, sem JavaScript
- [ ] #2 Texto traduzido e teste que confere a chave de todos os campos
<!-- AC:END -->
