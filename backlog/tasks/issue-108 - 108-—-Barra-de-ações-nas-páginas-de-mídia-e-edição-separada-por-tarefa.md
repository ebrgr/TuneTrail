---
id: ISSUE-108
title: 108 — Barra de ações nas páginas de mídia e edição separada por tarefa
status: Aberta
assignee: []
created_date: '2026-11-14T01:06:00Z'
updated_date: '2026-11-14T01:12:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-107
  - ISSUE-109
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
priority: high
type: feature
ordinal: 108000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: comparação da página de faixa com o Koito React (`/track/23` × `/evertonberger/track/91`, evidência em backlog/evidencias/ui-responsiva/comparacao-koito-react.md). Objetivo: comportamento e aparência próximos do original, mantendo HTML do servidor e funcionamento sem JavaScript.

Situação atual: o cabeçalho de faixa, álbum e artista mostra só o link de texto "Manage this item", visível para todos os visitantes, que leva a `/{slug}/manage/{tipo}/{id}` — uma página com 42 formulários misturando apelidos, MBID, artistas, fuso, escutas, adicionar escuta, exclusão, mesclagem e capa. No Koito, o administrador vê ícones no topo do cabeçalho e cada um abre um modal com uma única tarefa.

Passos — visual:
1. Barra de ícones no canto superior direito do cabeçalho de mídia (acima da capa no celular), na ordem do Koito: MusicBrainz (link externo, para todos, quando houver MBID), Adicionar escuta (só faixa), Editar, Trocar imagem (álbum e artista), Mesclar, Excluir. Ícones Lucide (ISC, mesma licença já registrada em `icons/LICENSE`): acrescentar Pencil, Image, Merge, Plus e Trash2. Tamanho via tokens (equivalente a 30px no desktop e 22px no celular), alvo de toque de 44×44px, `aria-label` e `title` traduzidos em `langs/`.
2. Ícones de gestão só para sessão de administrador, decidido no servidor (sem o `fetch('/apis/web/v1/user')` tardio do `live.js`); visitantes não veem o link de gestão. Resposta com `Vary: Cookie` e cache privado (ISSUE-106). O `live.js` passa a usar o mesmo sinal para a exclusão rápida no histórico.

Passos — funcionamento:
3. Dividir a página de gestão em páginas por tarefa, cada uma com só o necessário: `…/manage/{tipo}/{id}/edit` (apelidos, artistas e MBID), `/add-listen`, `/image`, `/merge` (busca + revisão existente), `/delete` (revisão existente). A página antiga vira índice curto com links para cada tarefa, para não quebrar endereços salvos.
4. Sem JavaScript, cada ícone é um link para a página da tarefa; com JavaScript, abre um `<dialog>` nativo carregando o mesmo conteúdo como fragmento (`?partial=1`, mecanismo da ISSUE-107), com foco preso no diálogo, Esc para fechar e foco devolvido ao ícone.
5. Ações dentro do diálogo atualizam só a seção (ISSUE-107); depois de mesclar ou excluir, navegar para o destino ou para a lista, como hoje.
6. Trocar imagem: além do diálogo, aceitar arrastar uma imagem sobre a capa para administradores (o `admin.js` já tem o tratamento de arquivo), com prévia e confirmação.
7. Histórico: o link "Manage" por escuta passa a abrir o diálogo da escuta (excluir, ver detalhes) em vez da página misturada.

Segurança: páginas e fragmentos novos com as mesmas regras de sessão, origem, CSRF, token de uso único e escopo de perfil; testes de que visitantes não recebem ícones nem conseguem abrir as rotas. Revisão do Advisor recomendada (AGENTS.md: rotas → @Opus effort=medium, @Sol effort=low).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Cabeçalho de faixa, álbum e artista com barra de ícones na ordem e posição do Koito, nos dois tamanhos de tela
- [ ] #2 Visitantes não veem ícones de gestão; admin vê sem requisição extra nem atraso
- [ ] #3 Cada ícone abre só a sua tarefa em diálogo; sem JavaScript abre a página da tarefa
- [ ] #4 Página de gestão antiga vira índice; endereços antigos continuam funcionando
- [ ] #5 Arrastar imagem sobre a capa troca a imagem com confirmação (álbum e artista)
- [ ] #6 Testes de permissão, CSRF e perfil para páginas e fragmentos novos
- [ ] #7 Capturas lado a lado com o Koito (desktop e celular) e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:12:00Z

---

A detecção da sessão de administrador no servidor (passo 2) passa a vir da ISSUE-109; esta tarefa reaproveita o sinal .Admin para mostrar a barra de ações.

---
<!-- COMMENTS:END -->
