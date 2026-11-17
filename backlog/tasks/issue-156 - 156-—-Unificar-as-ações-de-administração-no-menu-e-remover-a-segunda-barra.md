---
id: ISSUE-156
title: 156 — Unificar as ações de administração no menu (+) e remover a segunda barra
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-17T12:46:00Z'
updated_date: '2026-11-17T14:26:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-155
references:
  - backlog/evidencias/issue-156/verificacao.md
  - backlog/evidencias/issue-156/review.md
type: feature
ordinal: 169000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor na ISSUE-155 (plano ai-memory 01a1e987-1d45-7cc1-ac8a-4e3b2e72e65d), alternativa B aprovada pelo usuário em 2026-11-17. Remover a segunda barra lateral (ISSUE-123) e reunir as ações no menu (+) existente, em uma única superfície details nativa com grupos de título textual: Escuta (Escolher perfil de escuta), Aparência (Tema e idioma), Administração (sem sessão: só Entrar com a chave; com sessão: Conta, Gerenciar perfis, Configuração do servidor) e Sobre TuneTrail. Na prévia: texto 'Prévia: administração indisponível', sem botão inativo focalizável. Retirar a opção empurrar/sobrepor (nav-layout) do painel de tema e do JS. Sair fica para a ISSUE-157; até lá a saída continua pelo formulário da página de administração. Arquivos: assets/templates/shared/nav.html (partial/admin-actions; remover admin-sidebar e data-nav-toggle), assets/css/ui/layout/shell.css (sidebar-secondary, :has(#nav-admin), offsets), assets/js/nav.js (e referência em shared/head.html e manifest.json depois de conferir dependentes), assets/js/ui.js e theme-picker.js/theme.js (nav-layout), Nav.Open se sem outro consumidor, testes validate_navigation_071.cjs, navigation_071_test.go e engine/admin_sidebar_123_test.go (migrar as garantias de papel, seleção e erro pré-autenticação). Atenção: foco ao fechar o tema com o (+) fechado, pointerdown fora, painel com altura limitada e rolagem em 320px, sem JS, textos longos e i18n, preservar period e tz nos destinos que já os carregam.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A segunda barra e a opção empurrar/sobrepor foram removidas; o conteúdo não é deslocado ao abrir a administração
- [x] #2 O menu (+) mostra grupos Escuta, Aparência, Administração e Sobre conforme o papel (visitante, admin, prévia), com nomes completos em todas as larguras
- [ ] #3 Funciona sem JavaScript (abrir o menu e seguir Entrar, Conta, Perfis e Servidor) e com foco, teclado e leitor de tela verificados
- [x] #4 Garantias do admin_sidebar_123_test migradas para o novo grupo; validate_navigation_071 e make check verdes
- [ ] #5 Capturas em celular, médio e desktop e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-17T12:49:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-156` na branch `feat/issue-156-unificar-as-acoes-de-administracao` criada a partir de `main`.

---

author: codex
created: 2026-11-17T13:00:00Z

---

Revisão própria: APROVADO; diff auditado com os critérios reviewer/deslop, sem delegação. Evidência: backlog/evidencias/issue-156/review.md. AC3 permanece aberto por faltar leitura manual com leitor de tela; foco, teclado, sem JS e AX automatizado passaram. AC5 aguarda aceite visual.

---

author: codex
created: 2026-11-17T13:00:00Z

---

Verificação: make check e make ui.check verdes; 270 vistas validate_navigation_071 com visitante/admin/prévia, com/sem JS, Yuu/Pearl, 320–1920px e paisagem; papéis, seleção, erro pré-auth, foco/teclado, AX, alvos 44px, rolagem, margens 58/224 e footer verificados. Referência regenerada e seis capturas em backlog/evidencias/issue-156/. ACs 1, 2 e 4 marcados. AC3 aguarda leitura manual com leitor de tela; AC5 aguarda aceite visual do usuário. Revisão própria APROVADO; sem push, merge, publish ou Done.

---

author: claude
created: 2026-11-17T14:26:00Z

---

Ajuste pedido pelo usuário (2026-11-17): removido o item 'Time period' do menu (+) (details .nav-periods-menu, regras de CSS e asserts), por ser redundante com o seletor de cada página; o grupo .nav-periods da lateral com rótulos permanece. Referência e as 6 capturas do menu regeneradas (as capturas desta rodada vêm de um servidor estático da referência, por isso capas aparecem como imagem quebrada; só o menu importa). 270 vistas de navegação e make check verdes.

---
<!-- COMMENTS:END -->
