---
id: ISSUE-156
title: 156 — Unificar as ações de administração no menu (+) e remover a segunda barra
status: Aberta
assignee: []
created_date: '2026-11-17T12:46:00Z'
updated_date: '2026-11-17T12:46:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-155
type: feature
ordinal: 169000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: parecer do Advisor na ISSUE-155 (plano ai-memory 01a1e987-1d45-7cc1-ac8a-4e3b2e72e65d), alternativa B aprovada pelo usuário em 2026-11-17. Remover a segunda barra lateral (ISSUE-123) e reunir as ações no menu (+) existente, em uma única superfície details nativa com grupos de título textual: Escuta (Escolher perfil de escuta), Aparência (Tema e idioma), Administração (sem sessão: só Entrar com a chave; com sessão: Conta, Gerenciar perfis, Configuração do servidor) e Sobre TuneTrail. Na prévia: texto 'Prévia: administração indisponível', sem botão inativo focalizável. Retirar a opção empurrar/sobrepor (nav-layout) do painel de tema e do JS. Sair fica para a ISSUE-157; até lá a saída continua pelo formulário da página de administração. Arquivos: assets/templates/shared/nav.html (partial/admin-actions; remover admin-sidebar e data-nav-toggle), assets/css/ui/layout/shell.css (sidebar-secondary, :has(#nav-admin), offsets), assets/js/nav.js (e referência em shared/head.html e manifest.json depois de conferir dependentes), assets/js/ui.js e theme-picker.js/theme.js (nav-layout), Nav.Open se sem outro consumidor, testes validate_navigation_071.cjs, navigation_071_test.go e engine/admin_sidebar_123_test.go (migrar as garantias de papel, seleção e erro pré-autenticação). Atenção: foco ao fechar o tema com o (+) fechado, pointerdown fora, painel com altura limitada e rolagem em 320px, sem JS, textos longos e i18n, preservar period e tz nos destinos que já os carregam.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A segunda barra e a opção empurrar/sobrepor foram removidas; o conteúdo não é deslocado ao abrir a administração
- [ ] #2 O menu (+) mostra grupos Escuta, Aparência, Administração e Sobre conforme o papel (visitante, admin, prévia), com nomes completos em todas as larguras
- [ ] #3 Funciona sem JavaScript (abrir o menu e seguir Entrar, Conta, Perfis e Servidor) e com foco, teclado e leitor de tela verificados
- [ ] #4 Garantias do admin_sidebar_123_test migradas para o novo grupo; validate_navigation_071 e make check verdes
- [ ] #5 Capturas em celular, médio e desktop e aceite visual do usuário
<!-- AC:END -->
