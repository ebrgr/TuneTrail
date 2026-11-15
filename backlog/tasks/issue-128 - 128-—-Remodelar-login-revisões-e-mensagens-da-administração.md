---
id: ISSUE-128
title: 128 — Remodelar login, revisões e mensagens da administração
status: Aberta
assignee: []
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-15T15:18:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
priority: medium
type: feature
ordinal: 141000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Telas de apoio da administração: login (/admin sem sessão), revisão de mesclagem e exclusão do catálogo (catalog-review), revisão de exclusão de perfil (profile-review) e mensagens de erro e aviso. Proposta: login num cartão central no padrão comum; revisões com resumo do impacto em cartão, lista do que será alterado e confirmação na zona de perigo; erros e avisos com um único componente de mensagem, o mesmo dos demais cartões. Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Login em cartão no padrão comum
- [ ] #2 Revisões com resumo do impacto e confirmação na zona de perigo
- [ ] #3 Mensagens de erro e aviso com um componente único
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
<!-- AC:END -->
