---
id: ISSUE-125
title: 125 — Remodelar a página de conta do administrador
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
ordinal: 138000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Parte da ISSUE-074 (remodelação responsiva da administração). Origem: pedido do usuário em 15/11/2026. Página /admin?section=account (ícone Conta da segunda barra, ISSUE-123). Hoje: título Administração, link de configuração, botão Sair solto no topo e um único bloco com nome de usuário e troca de senha. Proposta: cartão Identidade (nome de usuário), cartão Senha (nova senha e confirmação, requisitos visíveis), cartão Sessão (sessão atual e Sair com peso visual secundário) e zona de perigo reservada para ações futuras. Padrão visual comum (ISSUE-074): o mesmo estilo responsivo já usado na busca (ISSUE-107) e no painel de Tema (ISSUE-123) — título da seção acima de um cartão com borda fina e cantos arredondados (bg-secondary, borda bg-tertiary, raio 8px), celular primeiro (uma coluna até 40rem, duas colunas a partir de 64rem quando couber), alvos de toque de 44px, rótulos acima dos campos, mensagens de sucesso e erro junto do cartão afetado e ações destrutivas numa zona de perigo separada e confirmada. Funciona sem JavaScript; com JavaScript as ações atualizam só o cartão, usando o mecanismo da ISSUE-124. Capturas antes e depois em 390, 768 e 1440 px, nos temas Yuu e Pearl, com aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cartões Identidade, Senha e Sessão no padrão comum, em uma ou duas colunas conforme a largura
- [ ] #2 Sair com peso secundário; mensagens junto do cartão
- [ ] #3 Funciona sem JavaScript; com JavaScript atualiza só o cartão
- [ ] #4 Capturas 390/768/1440 em Yuu e Pearl com aceite visual
<!-- AC:END -->
