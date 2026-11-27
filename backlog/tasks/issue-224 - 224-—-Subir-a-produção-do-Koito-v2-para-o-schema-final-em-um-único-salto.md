---
id: ISSUE-224
title: 224 — Subir a produção do Koito v2 para o schema final em um único salto
status: Aberta
assignee: []
created_date: '2026-11-27T15:21:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - banco
  - operacao
milestone: m-15
dependencies:
  - ISSUE-195
priority: high
type: chore
ordinal: 237000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decisão do usuário em 27/11/2026 (B1): decidir tudo, consolidar as migrações (ISSUE-195) e subir a produção uma única vez, direto do Koito v2 para o schema final. Plano e medições no doc-046 §7: backup verificado, restauração ensaiada, janela de manutenção e conferência por perfil. Depois desta issue entra a ISSUE-210.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Backup verificado e restauração ensaiada antes da janela
- [ ] #2 Salto v2 para o schema final com digest e contagens por perfil iguais
- [ ] #3 Janela de manutenção com tempo e espaço em disco previstos pelo ensaio da ISSUE-195 (300 mil e 1 milhão de escutas)
- [ ] #4 Plano de volta documentado e aceite do usuário
<!-- AC:END -->
