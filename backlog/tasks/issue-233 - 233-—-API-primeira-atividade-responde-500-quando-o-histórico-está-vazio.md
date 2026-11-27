---
id: ISSUE-233
title: '233 — API: primeira atividade responde 500 quando o histórico está vazio'
status: Aberta
assignee: []
created_date: '2026-11-27T21:22:00Z'
updated_date: '2026-11-27T21:22:00Z'
labels:
  - api
milestone: m-13
dependencies: []
priority: low
type: bug
ordinal: 246000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: achado registrado na revisão da ISSUE-231 (27/11/2026) e pedido do usuário para abrir a issue. Since: alcançável na v0.81 (última publicada) e anterior à ISSUE-231. O handler engine/handlers/get_first_activity.go responde 500 (failed to get first listen activity) para qualquer erro de GetFirstListenUnix, inclusive quando o perfil ainda não tem escutas, caso em que o armazenamento devolve db.ErrNotFound. A rota é /first-activity, das leituras por perfil da API web (engine/routes.go, bindProfileReads); não encontrei chamador nos assets nem na documentação do projeto, mas clientes externos podem usá-la. Causa confirmada pela leitura do código; falta reproduzir pela API. A decidir antes de implementar a resposta do histórico vazio: (a) 200 com time nulo, que não é erro e mantém o formato; (b) 404; (c) 204 sem corpo. Falhas reais continuam 500, e vale registrá-las acima do nível de depuração.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A resposta para histórico vazio está decidida pelo usuário e registrada na tarefa
- [ ] #2 Perfil sem escutas recebe a resposta decidida, sem 500, com teste pela rota
- [ ] #3 Falha real do armazenamento continua respondendo 500, com teste, e é registrada no log
<!-- AC:END -->
