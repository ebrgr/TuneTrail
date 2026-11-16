---
id: ISSUE-145
title: 145 — Contrato de API e autenticação por dispositivo para apps nativos
status: Aberta
assignee: []
created_date: '2026-11-16T01:05:00Z'
labels:
  - api
milestone: m-12
dependencies:
  - ISSUE-104
  - ISSUE-121
  - ISSUE-100
documentation:
  - doc-038
priority: low
type: spike
ordinal: 158000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: estudo doc-038 (16/11/2026). Antes de qualquer app nativo, definir o contrato que ele vai consumir: o que a API web JSON /apis/web/v1 expõe como estável (OpenAPI da ISSUE-104), como um dispositivo se autentica (token por dispositivo, com escopos, revogação e renovação; ISSUE-121), e como as leituras mudam com perfis privados (ISSUE-100). Saída: proposta de contrato e de autenticação, sem implementação; decidir se o token convive com o cookie de administrador e como o app descobre e valida o endereço do servidor (TLS, subcaminho). Inclui avaliar o 'tocando agora' em tempo real (ISSUE-084).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Contrato de leitura documentado (rotas, formatos, versionamento e política de mudança)
- [ ] #2 Modelo de autenticação por dispositivo decidido: token, escopos, revogação e coexistência com o cookie de administrador
- [ ] #3 Impacto dos perfis privados nas leituras do app descrito
- [ ] #4 Decisão registrada em backlog/decisions após o aceite
<!-- AC:END -->
