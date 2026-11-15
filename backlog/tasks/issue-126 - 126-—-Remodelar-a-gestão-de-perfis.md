---
id: ISSUE-126
title: 126 — Remodelar a gestão de perfis
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
ordinal: 139000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Páginas /admin?section=profiles e /admin?profile=N (ícone Perfis da segunda barra). Hoje: lista simples de links, criação de perfil num <details> e, por perfil, uma página longa com detalhes, avatar, chaves de API, importação/exportação e relay em sequência. Proposta: grade de cartões de perfil com avatar, nome, /slug e total de escutas, mais um cartão Criar perfil; página do perfil com cartões Perfil e avatar, Chaves de API, Importar e exportar, Relay ListenBrainz e zona de perigo (excluir perfil, que já tem página de revisão). Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Grade de cartões de perfil e cartão Criar perfil no padrão comum
- [ ] #2 Página do perfil dividida em cartões, com zona de perigo separada
- [ ] #3 Funciona sem JavaScript; com JavaScript atualiza só o cartão, preservando o token de uso único
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
<!-- AC:END -->
