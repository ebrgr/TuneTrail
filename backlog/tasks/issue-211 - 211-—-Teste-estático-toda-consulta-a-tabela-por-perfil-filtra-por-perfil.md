---
id: ISSUE-211
title: '211 — Teste estático: toda consulta a tabela por perfil filtra por perfil'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-24T00:51:00Z'
labels:
  - validacao
  - banco
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: low
type: enhancement
ordinal: 224000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-046 §4.4. O SQLite não tem segurança por linha: o isolamento entre perfis depende de cada consulta filtrar por profile_id, e nada no banco impede uma consulta nova de esquecer o filtro. A matriz da ISSUE-164 validou o comportamento por papel, mas não há auditoria estática. Criar um teste que percorra o SQL do Go de produção e falhe quando uma consulta a uma tabela com profile_id não filtrar por ele nem constar de uma lista explícita de exceções justificadas (catálogo, administração, rotinas de retenção e de saúde por origem). Não substitui segurança por linha, mas torna a lacuna visível a cada mudança. Reaproveitar o analisador de consultas de backlog/evidencias/db-auditoria/audit_db_usage.py. Depende de decisão do usuário (doc-046 §8, decisão 8).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Teste que lista toda consulta a tabela com profile_id sem filtro por perfil e falha fora da lista de exceções
- [ ] #2 Lista de exceções justificada, uma por uma, e revisada pelo Reviewer
- [ ] #3 Teste integrado ao make check
<!-- AC:END -->
