---
id: ISSUE-154
title: 154 — Last played em largura total e lateral desktop só com ícones por padrão
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-16T19:25:00Z'
updated_date: '2026-11-17T11:35:00Z'
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
- [x] #1 Last played ocupa toda a largura do conteúdo na Home em 1024, 1440 e 1920px, sem rolagem horizontal
- [x] #2 Acima de 64rem a lateral é só de ícones por padrão, com tooltip no rótulo
- [x] #3 A opção do painel de tema ativa os rótulos e volta ao padrão
- [ ] #4 Testes, ui.check e capturas do usuário (Home e lateral) conferidos
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-16T19:25:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-154` na branch `fix/issue-154-last-played-em-largura-total` criada a partir de `main`.

---

author: claude
created: 2026-11-16T19:28:00Z

---

Verificação: Last played mede a mesma largura da grade da Home (886px em 1024, 1302px em 1440, 1360px em 1920) sem rolagem horizontal; acima de 64rem a lateral é só de ícones por padrão, com tooltip, e a opção do painel de tema ativa os rótulos; make check verde (12/12 e2e), validate_navigation_071 com 108 vistas. Referência ui.reference.check passou a apontar para backlog/evidencias/issue-154/reference. Aceite visual do usuário pendente (AC #4).

---

author: claude
created: 2026-11-16T20:02:00Z

---

Ajuste pedido pelo usuário (2026-11-16): o menu (+) fecha ao clicar fora (pointerdown fora do .nav-more) e a Retrospectiva substitui o Histórico na barra, saindo do menu (+); o histórico já aparece na Home. Verificado no navegador com os dados reais (cópia do banco): abre ao clicar, fecha ao clicar fora, alterna pelo próprio botão; Rewind aparece uma vez na barra e 0 vezes no menu. make check verde (12/12), referência regenerada.

---

author: claude
created: 2026-11-16T20:23:00Z

---

Ajuste pedido pelo usuário (2026-11-16): o gráfico de atividade passa a ter largura máxima de 35rem (igual ao gráfico de interesse ao longo do tempo) e as células escalam com o cartão (36 semanas, 26 em cartões estreitos), sem barras de rolagem nem semanas cortadas. Verificado na página /evertonberger/track/37 e na Home em 320 a 1920px: sem rolagem interna nem da página. make check verde.

---

author: claude
created: 2026-11-17T11:08:00Z

---

Ajuste pedido pelo usuário (2026-11-17): nomes e textos dos cards de destaque truncavam porque a lista virava duas colunas de cerca de 200px com capa de 90px (texto de 77px em 1440px). Agora a segunda coluna só aparece com 22rem por item, o texto ocupa a largura restante ao lado da capa e '(N) reproduções' não quebra de linha. Verificado na Home com dados reais em 320 a 1920px: sem palavras partidas nem rolagem. make check verde.

---

author: claude
created: 2026-11-17T11:35:00Z

---

Ajuste pedido pelo usuário (2026-11-17): o ícone de entrar (LogIn) lembrava sair; trocado pelo ícone de chave (KeyRound, Lucide, ISC), escolhido pelo usuário entre cinco opções. LogIn.svg removido (sem outros usos) e manifest atualizado. Reestudo da segunda barra aberto na ISSUE-155. make check verde.

---
<!-- COMMENTS:END -->
