---
id: ISSUE-143
title: 143 — Excluir item só do perfil, com confirmação irreversível
status: Aberta
assignee: []
created_date: '2026-11-15T22:44:00Z'
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
Pedido do usuário em 15/11/2026, ao revisar a ISSUE-140. Hoje Excluir no diálogo de gestão é revisão e exclusão do catálogo compartilhado, para todos os perfis, e clicar na confirmação não apaga nada. O pedido: o diálogo mostra uma janela pequena 'Você realmente deseja excluir o item NOME DO ITEM?', com aviso de que não existe desfazer; ao confirmar, apaga para este perfil todos os itens com esse nome (faixa, álbum ou artista), sempre respeitando os perfis, sem tocar nos dados de outros perfis. Decisões pendentes antes de implementar: o que significa 'apagar o item' para o perfil (remover as escuta do perfil e deixar o catálogo compartilhado intacto, ou também remover entradas do catálogo que nenhum perfil use), como o nome é comparado (igualdade exata, caixa, e se o artista conta) e o que acontece com a página do item depois. Revisão do Advisor recomendada: operação destrutiva e de escopo por perfil.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Excluir abre uma janela pequena com o nome do item e o aviso de que não existe desfazer
- [ ] #2 Confirmar apaga para o perfil todos os itens com o nome, em faixa, álbum e artista, e não altera nenhum outro perfil
- [ ] #3 Clicar em confirmar funciona e leva ao destino definido; falhas mostram erro claro, sem apagar parcialmente
- [ ] #4 Testes com dois perfis, nomes iguais e IDs forjados passam pelo roteador principal
<!-- AC:END -->
