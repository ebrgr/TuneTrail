---
id: ISSUE-155
title: '155 — Reestudar a segunda barra de administração: submenu dedicado ou menu (+)'
status: Aberta
assignee: []
created_date: '2026-11-17T11:31:00Z'
updated_date: '2026-11-17T11:31:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies: []
type: spike
ordinal: 168000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário durante a ISSUE-154. A segunda barra lateral (ISSUE-123) existe desde antes da barra inferior e da navegação responsiva (ISSUE-071). Hoje ela concentra administração e login, com o modo empurrar/sobrepor (painel de tema), e o ícone de entrar (LogIn) lembra sair. Agora que o celular tem barra inferior com o menu (+), reavaliar se a segunda barra continua fazendo sentido no desktop ou se administração, login/logout e demais funções devem ficar num submenu como o (+). Questões: (1) manter a segunda barra, trocá-la por um submenu/popover como o (+) ou unificar os dois; (2) o que fica visível sem sessão (entrar) e com sessão (conta, perfis, servidor, sair); (3) coerência entre celular, médio e desktop com a lateral só de ícones; (4) impacto no modo empurrar/sobrepor e na persistência das opções (ISSUE-153); (5) foco, teclado e leitor de tela; (6) ícones: entrar, sair, conta. Entregável: parecer do Advisor com opções, riscos e recomendação, e issues de implementação; sem código nesta issue.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Parecer com 2 ou mais alternativas (manter, submenu como o (+), unificar), com capturas ou esboços e impacto no código
- [ ] #2 Definição do que aparece sem sessão e com sessão, incluindo entrar e sair, e dos ícones
- [ ] #3 Recomendação aprovada pelo usuário e issues de implementação criadas
<!-- AC:END -->
