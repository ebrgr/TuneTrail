---
id: ISSUE-198
title: 198 — Log estruturado na configuração do servidor
status: Aberta
assignee: []
created_date: '2026-11-23T16:32:00Z'
updated_date: '2026-11-23T16:33:00Z'
labels:
  - config
  - ui
milestone: m-10
dependencies:
  - ISSUE-127
references:
  - backlog/evidencias/issue-127/ESTUDO.md
type: feature
ordinal: 211000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-127. Acrescentar ENABLE_STRUCTURED_LOGGING ao cartão «Geral e log» da página do servidor, com a chave logging.structured no conf.toml (bool, valor padrão atual, reinício para valer). Exige a chave no leitor (internal/cfg/toml.go), no editor (settings_edit.go) e em managedDefaults, o campo em settingsFields com grupo general, textos pt-BR e en, a referência de configuração e o changelog. Os testes de correspondência página × conf.toml da 127 devem continuar verdes. Relacionada à ISSUE-112 (diagnóstico e visualização de logs).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Campo «Log estruturado» no cartão Geral e log, com a chave logging.structured no conf.toml e valor padrão igual ao atual
- [ ] #2 Leitor, editor e padrões do cfg conhecem a chave; testes de correspondência página × conf.toml continuam verdes
- [ ] #3 Textos pt-BR e en, referência de configuração e changelog atualizados
<!-- AC:END -->
