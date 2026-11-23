---
id: ISSUE-104
title: 104 — OpenAPI e métricas Prometheus
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-23T00:58:00Z'
labels:
  - api
  - operacao
milestone: m-15
dependencies:
  - ISSUE-082
priority: low
type: feature
ordinal: 104000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §8. OpenAPI para `/apis/web/v1` e `/apis/listenbrainz`; `/metrics` com rótulos de baixa cardinalidade, sem títulos nem perfis por padrão.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Especificação OpenAPI validada nos testes
- [ ] #2 Métricas sem dados pessoais
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:58:00Z

---

Ordem (22/11/2026): a ISSUE-196 (auditoria e classificação de todas as rotas) vem ANTES desta. Só rotas em uso e estáveis entram no OpenAPI; as de sessão e administração ficam marcadas como internas.

---
<!-- COMMENTS:END -->
