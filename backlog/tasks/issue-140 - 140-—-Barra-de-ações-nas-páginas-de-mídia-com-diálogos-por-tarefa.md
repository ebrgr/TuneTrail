---
id: ISSUE-140
title: 140 — Barra de ações nas páginas de mídia com diálogos por tarefa
status: Aberta
assignee: []
created_date: '2026-11-15T20:04:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-139
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
priority: high
type: feature
ordinal: 153000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte B da ISSUE-108 (avaliação do Codex em backlog/evidencias/issue-108/avaliacao-codex.md, §5). Barra de ícones no cabeçalho de faixa, álbum e artista na ordem do Koito original: MusicBrainz (link externo, para todos, com MBID), Adicionar escuta (só faixa), Editar, Trocar imagem (álbum e artista), Mesclar, Excluir. Ícones Lucide Pencil, Image, Merge, Plus e Trash2 registrados no manifesto e em icons/LICENSE; tamanhos por tokens (30px desktop, 22px celular), alvo de 44×44px, aria-label e title traduzidos. Ícones de gestão só para administrador, usando o sinal .Admin da ISSUE-109, sem requisição extra. Com JavaScript, cada ícone abre um dialog nativo com o fragmento da página da tarefa (recorte A); sem JavaScript é um link. O primeiro fragmento emite o ticket de uso único; o enhance.js ganha escopo ao diálogo e não troca a URL nem o h1 da página pública. Trocar imagem atualiza a capa sem recarregar a página. Depois de editar, o cabeçalho (nome, MBID, artistas, contagens) é atualizado. Posição da barra alinhada com a ISSUE-073.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Barra na ordem e posição do Koito original em faixa, álbum e artista, com e sem MBID, em desktop e celular
- [ ] #2 Visitantes não recebem ícones de gestão no HTML; administrador os vê na primeira pintura, sem consulta de papel
- [ ] #3 Cada ícone abre só a sua tarefa em diálogo (foco preso, Esc, foco devolvido) e, sem JavaScript, a página da tarefa; erro, rede e sessão expirada são recuperáveis
- [ ] #4 Trocar imagem atualiza a capa sem recarregar; editar atualiza o cabeçalho; resposta antiga não preenche diálogo novo e o botão Voltar não restaura formulário ativo
- [ ] #5 Capturas lado a lado com o Koito original (desktop e celular) e aceite visual do usuário
<!-- AC:END -->
