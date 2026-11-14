---
id: ISSUE-063
title: 063 — Consolidar os scripts de validação por issue em uma suíte e2e no CI
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:25:00Z'
labels:
  - validacao
  - processo
milestone: m-9
dependencies: []
priority: high
type: chore
ordinal: 63000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §1.3 e §8 — 37 dos 50 scripts em `scripts/` têm sufixo de issue (`validate_*_0NN`, `capture_*_0NN`) e não rodam como suíte.

Passos:
1. Inventariar os 37 scripts: o que verificam, se já existe teste Go equivalente, se dependem de React removido.
2. Classificar cada um: coberto (remover), converter (nova jornada), histórico (mover para `scripts/historico/` com nota).
3. Criar a suíte de jornadas em Go (`httptest` + SQLite descartável) cobrindo: login/sessão, perfis, ingestão ListenBrainz, páginas públicas, administração, importação/exportação, permissões e isolamento entre perfis.
4. Ligar a suíte ao `make check` e ao workflow `test.yml`; capturas com navegador ficam opcionais (`make ui.capture`).
5. Atualizar `scripts/README.md`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Inventário dos 37 scripts com destino de cada um
- [ ] #2 Suíte de jornadas executa em `make check` e no CI sem rede externa
- [ ] #3 Jornadas cobrem perfis, ingestão, páginas públicas, administração, importação e isolamento
- [ ] #4 Scripts de issue removidos ou arquivados como históricos, sem quebrar evidências citadas no backlog
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T01:25:00Z

---

Início: branch `chore/issue-063-consolidar-os-scripts-de-validacao` criada a partir de `main`.

---
<!-- COMMENTS:END -->
