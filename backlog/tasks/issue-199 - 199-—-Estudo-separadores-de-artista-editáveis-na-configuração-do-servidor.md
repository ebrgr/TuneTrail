---
id: ISSUE-199
title: '199 — Estudo: separadores de artista editáveis na configuração do servidor'
status: Aberta
assignee: []
created_date: '2026-11-23T16:32:00Z'
updated_date: '2026-11-23T16:33:00Z'
labels:
  - config
  - catalogo
milestone: m-10
dependencies:
  - ISSUE-127
references:
  - backlog/evidencias/issue-127/ESTUDO.md
type: spike
ordinal: 212000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-127 (ESTUDO.md §7.4). ARTIST_SEPARATORS_REGEX define como nomes de artista são separados ao associar artistas (padrão: espaço, ponto mediano, espaço; vários padrões separados por ;;). Hoje uma expressão inválida impede o servidor de subir. Estudar: onde a expressão é usada, se altera só associações novas ou também dados existentes, como validar ao salvar sem aceitar expressão catastrófica, e se merece um cartão «Catálogo». Entregar parecer e, se aprovado pelo usuário, abrir a issue de implementação.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Parecer sobre uso, alcance (novas associações ou dados existentes), validação segura da expressão e cartão Catálogo
- [ ] #2 Decisão do usuário registrada; se aprovada, issue de implementação aberta
<!-- AC:END -->
