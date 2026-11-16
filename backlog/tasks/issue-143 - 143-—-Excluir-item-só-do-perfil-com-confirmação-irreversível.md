---
id: ISSUE-143
title: 143 — Excluir item só do perfil, com confirmação irreversível
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T22:44:00Z'
updated_date: '2026-11-16T00:55:00Z'
labels:
  - admin
  - catalogo
milestone: m-10
dependencies:
  - ISSUE-140
references:
  - backlog/evidencias/issue-140/relatorio.md
  - backlog/evidencias/issue-143/relatorio.md
  - backlog/evidencias/issue-143/browser-checks.json
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
- [x] #1 Excluir abre uma janela pequena com o nome do item e o aviso de que não existe desfazer
- [x] #2 Confirmar apaga para o perfil todos os itens com o nome, em faixa, álbum e artista, e não altera nenhum outro perfil
- [x] #3 Clicar em confirmar funciona e leva ao destino definido; falhas mostram erro claro, sem apagar parcialmente
- [x] #4 Testes com dois perfis, nomes iguais e IDs forjados passam pelo roteador principal
- [x] #5 Exclusão remove só as escutas do perfil com igualdade de nome do artista e da faixa (álbum e artista, conforme o tipo); catálogo e outros perfis intactos
- [x] #6 Depois de excluir, vai para a página inicial do perfil (/{slug}), para faixa, álbum e artista
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

author: ebrgr
created: 2026-11-16T00:19:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-143` na branch `feat/issue-143-excluir-item-so-do-perfil` criada a partir de `main`.

---

author: codex
created: 2026-11-16T00:28:00Z

---

Decisão do usuário via coordenador: faixa/álbum exigem igualdade exata, sensível a caixa, do nome principal e do conjunto completo de nomes principais dos artistas, ignorando ordem, sem aliases. Excluir artista remove apenas escutas de faixas com um único crédito; colaborações são preservadas. O diálogo informa a contagem calculada no servidor com o mesmo critério da exclusão e explica a preservação das colaborações.

---

author: codex
created: 2026-11-16T00:44:00Z

---

Verificação: exclusão pessoal com igualdade exata e conjunto completo de artistas; colaborações preservadas ao excluir artista; contagem e DELETE pelo mesmo seletor transacional. Roteador principal, tickets, CSRF/origem/sessão, dois perfis, homônimos, créditos derivados, replay, IDs forjados e rollback aprovados; Chrome com foco/Esc/rede/sessão expirada no banco e navegação real, 12 capturas Yuu/Pearl desktop/celular. make check verde com 12/12 E2E. Critérios 2–6 verificados; #1 reservado ao aceite visual. Por decisão do usuário, Reviewer/Advisor ficam com o coordenador, sem chamada claude CLI pelo worker.

---

author: ebrgr
created: 2026-11-16T00:44:00Z

---

Verificação: make check verde (12/12 E2E); exclusão só de escutas do perfil em uma transação, contagem/revalidação, catálogo e outros perfis preservados; testes de segurança e rollback pelo roteador principal; Chrome real com foco/Esc, rede, sessão expirada no banco, resposta perdida sem retry e navegação; 12 capturas Yuu/Pearl desktop/celular. Critérios 2–6 verificados, #1 aguarda aceite visual; revisão do diff pelo coordenador conforme decisão do usuário.

---

author: claude
created: 2026-11-16T00:55:00Z

---

Aceite: aprovada pelo usuário em 16/11/2026.

---
<!-- COMMENTS:END -->
