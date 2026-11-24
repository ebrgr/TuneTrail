---
id: ISSUE-209
title: >-
  209 — Aposentar o relay legado por variáveis de ambiente e remover
  legacy_relay_binding
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-24T00:51:00Z'
labels:
  - ingestao
  - config
  - docs
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: low
type: chore
ordinal: 222000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-046 §5.2 e doc-047. ENABLE_LBZ_RELAY, LBZ_RELAY_URL e LBZ_RELAY_TOKEN são o bootstrap legado: associam uma vez o relay do ambiente a um único perfil (initializeLegacyRelay em profile_store.go), e a tabela legacy_relay_binding só marca que isso já foi feito. A ISSUE-197 os mantinha até a ISSUE-164, que já foi publicada. Decidir e, se aprovado: remover as três variáveis do código e da referência de configuração, a função initializeLegacyRelay, a tabela legacy_relay_binding e o texto de relay legado do guia; migrar quem ainda usa para a tela de relay do perfil. Efeito: uma tabela a menos e menos código de ambiente. Depende de decisão do usuário (doc-046 §8, decisão 1) e conversa com a ISSUE-197.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Decisão registrada: o bootstrap de relay por variáveis de ambiente é aposentado, ou não
- [ ] #2 Se aprovado: variáveis, função de bootstrap e tabela removidas, referência e guia atualizados, com aviso para quem ainda as define
- [ ] #3 Testes de relay e de upgrade verdes
<!-- AC:END -->
