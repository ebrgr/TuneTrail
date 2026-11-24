---
id: ISSUE-205
title: '205 — Banco: fundir profile_relays e relay_destinations'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-24T00:51:00Z'
labels:
  - banco
  - ingestao
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: medium
type: chore
ordinal: 218000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-046 §5.2 (pacote B). profile_relays (url e token) e relay_destinations (estado da fila, geração, pausa, lease) são uma relação 1:1: cada perfil tem no máximo um relay. Cinco gatilhos (trg_relay_destination_insert, _update, _delete, trg_relay_draft_insert, trg_relay_draft_change) só existem para manter uma tabela em sincronia com a outra, e profile_relays é lida em 13 pontos do Go de produção. Fundir em uma tabela e remover os cinco gatilhos, preservando o comportamento do rascunho aguardando aprovação, da geração e do fencing da fila. Efeito: uma tabela e cinco gatilhos a menos. RISCO: é o relay durável, coberto por muitos testes e por regra de segurança de rede (relay_allowlist); exige Reviewer e o ensaio de upgrade com um relay configurado (a cópia real do usuário tem um). Entra no arquivo único da ISSUE-195. Depende de decisão do usuário (doc-046 §8, decisão 3).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Uma só tabela guarda URL, token e estado do relay; os 5 gatilhos de sincronia removidos
- [ ] #2 Rascunho, aprovação de rede, geração e fencing da fila com o mesmo comportamento, provados pelos testes de relay existentes
- [ ] #3 Upgrade v2 → atual com um relay configurado preserva URL, token e fila
<!-- AC:END -->
