---
id: ISSUE-154
title: 154 — Last played em largura total e lateral desktop só com ícones por padrão
status: Aberta
assignee: []
created_date: '2026-11-16T19:25:00Z'
updated_date: '2026-11-16T19:25:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
type: bug
ordinal: 167000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Correções de layout pedidas pelo usuário após as ISSUE-069 e 071. (1) A seção 'Last played' da Home ocupa só uma coluna (cerca de 352px em 1440px) porque .home-blocks virou grade auto-fill na 069; deve voltar a esticar na largura toda do conteúdo, como antes. (2) A lateral desktop com ícones e rótulos não agradou: o padrão acima de 64rem passa a ser a lateral só de ícones (formato intermediário), com o rótulo no tooltip antigo; a opção 'Rótulos da navegação' do painel de tema continua permitindo ativar os rótulos. Persistência no conf.toml segue na ISSUE-153.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Last played ocupa toda a largura do conteúdo na Home em 1024, 1440 e 1920px, sem rolagem horizontal
- [ ] #2 Acima de 64rem a lateral é só de ícones por padrão, com tooltip no rótulo
- [ ] #3 A opção do painel de tema ativa os rótulos e volta ao padrão
- [ ] #4 Testes, ui.check e capturas do usuário (Home e lateral) conferidos
<!-- AC:END -->
