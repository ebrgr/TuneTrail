---
id: ISSUE-126
title: 126 — Remodelar a gestão de perfis
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T22:53:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-124
  - ISSUE-163
references:
  - backlog/evidencias/issue-163/ESTUDO.md
  - backlog/evidencias/issue-126/ESTUDO.md
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
- [ ] #5 Meu perfil para o usuário comum e lista root de contas e perfis, com alvo derivado da sessão
- [ ] #6 Aviso junto ao checkbox de proteção: o relay configurado continua enviando escutas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-20T18:33:00Z

---

Reescopo (decisão do usuário em 20/11/2026, estudo da 163): absorve «Meu perfil» (gestão própria, sem informar ID ou slug) e a lista root de contas e perfis, reaproveitando os serviços da 160/162, com o aviso de que o relay segue ativo junto à opção de proteção. A exclusão de conta é zona root.

---

author: claude
created: 2026-11-22T22:51:00Z

---

Decisão do usuário (22/11/2026): despachar a 126 a um agente Claude em worktree própria, com FASE DE DESENHO primeiro. Hoje 'Minha conta' (/account) e 'Meu perfil' (/admin?profile=N) parecem a mesma coisa e há dois caminhos para a gestão de perfil e conta. O agente deve avaliar a proposta original da 126 e alternativas (por exemplo um local único com abas Conta, Perfil, Chaves de API, Histórico, Relay, Saúde, e uma área root separada de Administração), apresentar opções com prós e contras e recomendação e PARAR para decisão do usuário antes de implementar. Sem capturas; o usuário testa no servidor.

---

author: ebrgr
created: 2026-11-22T22:51:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-126` na branch `feat/issue-126-remodelar-a-gestao-de-perfis` criada a partir de `main`.

---

author: claude
created: 2026-11-22T22:53:00Z

---

Fase 1 (desenho) concluída, sem implementação. Estudo em backlog/evidencias/issue-126/ESTUDO.md. Recomendação A′: um único 'Minha conta' em /account com abas (Conta | Perfil | Chaves de API | Histórico | Relay | Saúde), alvo derivado da sessão e sem 'Meu perfil' separado; área root 'Administração' (Contas e perfis, Saúde, Servidor); root edita outra pessoa no mesmo shell em modo contexto (/admin?profile=N, cabeçalho 'Você x Maria') com aba Conta root. /admin?profile=<próprio> redireciona (303) para /account/profile; todas as URLs atuais e todos os POST permanecem. Descartadas: proposta original isolada (mantém dois caminhos), B (duas áreas), C (centrar no perfil), D (página única). Plano em 6 cortes com make check verde. Aguardando decisão do usuário (5 perguntas no §12) antes de implementar.

---
<!-- COMMENTS:END -->
