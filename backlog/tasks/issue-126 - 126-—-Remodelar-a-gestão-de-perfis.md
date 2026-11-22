---
id: ISSUE-126
title: 126 — Remodelar a gestão de perfis
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T15:18:00Z'
updated_date: '2026-11-22T23:37:00Z'
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

author: claude
created: 2026-11-22T23:04:00Z

---

Decisões do usuário (22/11/2026) sobre o estudo: (1) aprovada a Opção A′: um único 'Minha conta' com abas, sem 'Meu perfil' separado; o root opera outras contas na Administração, com o mesmo conjunto de abas em modo contexto; (2) abas: Conta | Perfil | Chaves de API | Histórico | Relay; a SAÚDE sai das abas e permanece no menu (+) principal, como hoje (dono e root); (3) /admin?profile=<próprio> redireciona (303) para /account/profile; as demais URLs atuais continuam funcionando; (4) default conservador: root exclui a própria conta pela Administração; zona de perigo da Conta própria segue vazia (125).

---

author: claude
created: 2026-11-22T23:23:00Z

---

Fase 2 implementada (make check verde; sem push/merge/publish/Done; sem capturas, usuário testa no servidor). Entrega: /account vira 'Minha conta' com cinco abas (Conta | Perfil | Chaves de API | Histórico | Relay) em /account, /account/profile, /account/keys, /account/history, /account/relay (alvo derivado da sessão, sem ID nem query; entram em IsPage/authorizePage e no retorno pós-login). /admin?profile=<próprio> e section=profiles|keys|relay|import|export redirecionam 303 para a aba própria; section=account&profile=<próprio> para /account. Root em outra conta: /admin?profile=N (e section=account|keys|import|relay) no mesmo shell em modo contexto (banner 'Nome (@slug) · você está operando como administrador', Você: <root>, Voltar), aba Conta com token, papel e revisão de exclusão (nunca para comum ou dono: 403). Administração: lista de contas e perfis em cartões + criar conta e perfil (/admin e ?section=profiles). Menu: sem 'Meu perfil'; Saúde fica no menu (dono e root); /admin/health só teve o link 'Configurar perfil' ajustado. Todos os POST, CSRF, ticket único, SameOrigin e autorização intactos; só muda o destino do 303 (dono -> /account/<aba>, root em outra conta -> contexto). Com JS: identidade/avatar/chaves/relay/papel atualizam só o cartão (enhance.js: formaction do submitter, multipart e data-slug); importação segue página inteira. Testes alterados: internal/adminui/{account_tokens,admin}_test.go; engine/{account_issue125,admin_badge_109,adminui_import,auth_issue159,authorization_issue160,i18n_admin_052,import_permissions_issue160,navigation_issue163,visibility_issue161}_test.go; scripts/validate_admin_webui.py e validate_isolation.py (e2e); referência backlog/evidencias/ISSUE-075/reference regenerada (28 arquivos). Novos: internal/webassets/account_tabs_test.go e engine/account_tabs_126_test.go (comum, dono, root, root em contexto, redirecionamentos, fragmentos, 403 sem vazamento). Pendente: capturas e aceite visual do usuário (AC4).

---

author: claude
created: 2026-11-22T23:31:00Z

---

Reviewer: APROVADO, sem BUG. Sonda HTTP em binário real com três papéis (26 verificações OK): comum sem controles de root e 403 nas rotas root; root com lista e contexto; redirecionamento do próprio perfil; POST em perfil alheio 403. Riscos baixos: aceite visual pendente, cartão único por ação com JS, N consultas na lista root. Removidos dois arquivos de trava de teste commitados por engano.

---

author: ebrgr
created: 2026-11-22T23:37:00Z

---

Verificação: Reviewer APROVADO (sonda HTTP 26 verificações nos três papéis); make check 13/13 E2E; aceite visual pelo usuário no servidor

---

author: evertonberger
created: 2026-11-22T23:37:00Z

---

Aceite do usuário em 22/11/2026 ('aceito o 126' e 'publique').

---
<!-- COMMENTS:END -->
