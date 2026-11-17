---
id: ISSUE-155
title: '155 — Reestudar a segunda barra de administração: submenu dedicado ou menu (+)'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T11:31:00Z'
updated_date: '2026-11-17T12:47:00Z'
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
- [x] #1 Parecer com 2 ou mais alternativas (manter, submenu como o (+), unificar), com capturas ou esboços e impacto no código
- [x] #2 Definição do que aparece sem sessão e com sessão, incluindo entrar e sair, e dos ícones
- [x] #3 Recomendação aprovada pelo usuário e issues de implementação criadas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-17T12:22:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-155` na branch `spike/issue-155-reestudar-a-segunda-barra-de` criada a partir de `main`.

---

author: advisor
created: 2026-11-17T12:41:00Z

---

Advisor — Modo B concluído por análise estática do HEAD 066734d, sem alteração de código ou execução de testes. [PLANO ISSUE-155] salvo no ai-memory: 01a1e987-1d45-7cc1-ac8a-4e3b2e72e65d (plans/issue-155-segunda-barra-advisor.md). Inclui inventário por sessão/viewport, esboços e comparação A manter/B reunir no (+)/C gaveta única, impactos em arquivos/JS/testes e ISSUE-153, logout HTML/API, acessibilidade, fallback sem JS, riscos e sequência de tarefas propostas (sem criar issues). Recomendação: B, ações agrupadas e rotuladas no (+), removendo a segunda barra e push/overlay após aprovação; Sair… abre confirmação HTML com ticket próprio e POST protegido existente. Decisões pendentes: (1) aprovar B e retirada de push/overlay; (2) confirmação de saída ou exigência de um clique; (3) rótulos globais no conf.toml ou padrão global com override por navegador na153. Nenhum AC marcado nem status alterado; aguardando decisão do usuário sobre o desenho.

---

author: ebrgr
created: 2026-11-17T12:47:00Z

---

Aceite do usuário (2026-11-17): aprovada a alternativa B (remover a segunda barra e a opção empurrar/sobrepor; ações em grupos no menu (+)); Sair abre uma página de confirmação e depois o botão Sair; refinar a ISSUE-153. Criadas as ISSUE-156 (unificar no menu (+)) e ISSUE-157 (sair com confirmação); ISSUE-153 refinada e dependente da 156. Pergunta 3 do Advisor (rótulos globais ou com escolha por navegador) segue para o critério #1 da ISSUE-153. Merge da branch depende de autorização à parte.

---
<!-- COMMENTS:END -->
