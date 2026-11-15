---
id: ISSUE-143
title: 143 — Excluir item só do perfil, com confirmação irreversível
status: Aberta
assignee: []
created_date: '2026-11-15T22:44:00Z'
updated_date: '2026-11-15T22:49:00Z'
labels:
  - admin
  - catalogo
milestone: m-10
dependencies:
  - ISSUE-140
references:
  - backlog/evidencias/issue-140/relatorio.md
priority: high
type: feature
ordinal: 156000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Pedido do usuário em 15/11/2026, ao revisar a ISSUE-140. Hoje Excluir no diálogo de gestão é revisão e exclusão do catálogo compartilhado, para todos os perfis, e clicar na confirmação não apaga nada. O pedido: o diálogo mostra uma janela pequena 'Você realmente deseja excluir o item NOME DO ITEM?', com aviso de que não existe desfazer; ao confirmar, apaga para este perfil todos os itens com esse nome (faixa, álbum ou artista), sempre respeitando os perfis, sem tocar nos dados de outros perfis. Decisões do usuário em 15/11/2026: (1) apagar significa remover só as escutas do perfil; o catálogo compartilhado e os outros perfis ficam intactos. (2) A correspondência é por igualdade do nome: para faixa, nome da faixa e do artista; para álbum, nome do álbum e do artista; para artista, nome do artista; sempre dentro do perfil. (3) Depois de excluir, vai para a página inicial do perfil (/{slug}); isso vale também quando o item removido é um artista (confirmado pelo usuário: a tela principal é a página inicial do perfil). Revisão do Advisor recomendada: operação destrutiva e de escopo por perfil.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Excluir abre uma janela pequena com o nome do item e o aviso de que não existe desfazer
- [ ] #2 Confirmar apaga para o perfil todos os itens com o nome, em faixa, álbum e artista, e não altera nenhum outro perfil
- [ ] #3 Clicar em confirmar funciona e leva ao destino definido; falhas mostram erro claro, sem apagar parcialmente
- [ ] #4 Testes com dois perfis, nomes iguais e IDs forjados passam pelo roteador principal
- [ ] #5 Exclusão remove só as escutas do perfil com igualdade de nome do artista e da faixa (álbum e artista, conforme o tipo); catálogo e outros perfis intactos
- [ ] #6 Depois de excluir, vai para a página inicial do perfil (/{slug}), para faixa, álbum e artista
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T22:48:00Z

---

Decisões do usuário em 15/11/2026: (1) remover só as escutas do perfil, sem tocar no catálogo compartilhado nem em outros perfis; (2) igualdade de nome do artista e da faixa (álbum: nome do álbum e do artista; artista: nome do artista), sempre dentro do perfil; (3) depois de excluir, ir para o perfil e, se for removido o artista, para a tela principal. Ponto a confirmar: 'tela principal' como a página inicial /.

---

author: claude
created: 2026-11-15T22:49:00Z

---

Esclarecimento do usuário em 15/11/2026: a 'tela principal' é a página inicial do perfil (/{slug}). Faixa, álbum e artista excluídos levam todos a ela.

---
<!-- COMMENTS:END -->
