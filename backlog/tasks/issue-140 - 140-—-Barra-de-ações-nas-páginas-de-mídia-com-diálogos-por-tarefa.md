---
id: ISSUE-140
title: 140 — Barra de ações nas páginas de mídia com diálogos por tarefa
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T20:04:00Z'
updated_date: '2026-11-15T22:44:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-139
references:
  - backlog/evidencias/issue-108/avaliacao-codex.md
  - backlog/evidencias/issue-140/relatorio.md
  - backlog/evidencias/issue-140/comparacao.html
  - backlog/evidencias/issue-140/browser-checks.json
  - backlog/evidencias/issue-140/reviewer-complementar.md
priority: high
type: feature
ordinal: 153000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte B da ISSUE-108 (avaliação do Codex em backlog/evidencias/issue-108/avaliacao-codex.md, §5). Barra de ícones no cabeçalho de faixa, álbum e artista na ordem do Koito mais recente (referência http://127.0.0.1:4111/artist/1, /album/1 e /track/1), posicionada inicialmente abaixo da barra superior, no canto superior direito: MusicBrainz (link externo, para todos, com MBID), Adicionar escuta (só faixa), Editar, Trocar imagem (álbum e artista), Mesclar, Excluir. Ícones Lucide Pencil, Image, Merge, Plus e Trash2 registrados no manifesto e em icons/LICENSE; tamanhos por tokens (30px desktop, 22px celular), alvo de 44×44px, aria-label e title traduzidos. Ícones de gestão só para administrador, usando o sinal .Admin da ISSUE-109, sem requisição extra. Com JavaScript, cada ícone abre um dialog nativo com o fragmento da página da tarefa (recorte A); sem JavaScript é um link. O primeiro fragmento emite o ticket de uso único; o enhance.js ganha escopo ao diálogo e não troca a URL nem o h1 da página pública. Trocar imagem atualiza a capa sem recarregar a página. Depois de editar, o cabeçalho (nome, MBID, artistas, contagens) é atualizado. Posição da barra alinhada com a ISSUE-073.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Barra na ordem e posição do Koito original em faixa, álbum e artista, com e sem MBID, em desktop e celular
- [x] #2 Visitantes não recebem ícones de gestão no HTML; administrador os vê na primeira pintura, sem consulta de papel
- [x] #3 Cada ícone abre só a sua tarefa em diálogo (foco preso, Esc, foco devolvido) e, sem JavaScript, a página da tarefa; erro, rede e sessão expirada são recuperáveis
- [x] #4 Trocar imagem atualiza a capa sem recarregar; editar atualiza o cabeçalho; resposta antiga não preenche diálogo novo e o botão Voltar não restaura formulário ativo
- [ ] #5 Capturas lado a lado com o Koito original (desktop e celular) e aceite visual do usuário
- [ ] #6 Todos os formulários dos diálogos usam botões e campos do tema
- [ ] #7 Adicionar escuta: data e hora, botão responsivo e fuso horário opcional abaixo
- [ ] #8 Editar item e Mesclar no padrão do Koito, com lista dinâmica e responsiva na mescla
- [ ] #9 Tooltip nos ícones da barra, no mesmo estilo da barra lateral
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-15T20:13:00Z

---

Referência visual definida pelo usuário em 15/11/2026: a última versão do Koito em http://127.0.0.1:4111/artist/1, /album/1 e /track/1 (instância local do usuário). A barra de ícones fica inicialmente abaixo da barra superior, no canto superior direito. Alinhar o padrão (ordem, tamanho, espaçamento, posição) com essa versão, não com a comparação histórica de 13/11.

---

author: ebrgr
created: 2026-11-15T20:45:00Z

---

Início: branch `feat/issue-140-barra-de-acoes-nas-paginas` criada a partir de `main`.

---

author: codex
created: 2026-11-15T21:58:00Z

---

Verificação: critérios 1–4 exercitados por roteador real e Chrome em fixture descartável; make check com 12/12 E2E. Reviewer sem itens impeditivos após correções de geometria, carregamento condicional de assets e recuperação; aceite visual #5 fica com o usuário. Atualização automática abrange cabeçalho/capa/contagens; histórico, atividade e gráficos conservam snapshot até navegação. Sem push, merge ou publicação.

---

author: ebrgr
created: 2026-11-15T22:00:00Z

---

Verificação: make check aprovado com 12/12 jornadas E2E; roteador real valida sessão (incluindo expiração no banco), origem, CSRF, perfil, tickets e plano de confirmação; Chrome valida faixa/álbum/artista em desktop/celular, sem JS e clique modificado, foco/Tab/Esc, nome/MBID/contagens/capa sem recarga, rede, respostas antigas e bfcache. Capturas com Koito atual e relatório em backlog/evidencias/issue-140; Reviewer complementar sem itens impeditivos. Critérios 1–4 verificados; #5 aguarda aceite visual do usuário.

---

author: claude
created: 2026-11-15T22:44:00Z

---

Aceite adiado: o usuário pediu ajustes antes do aceite final (15/11/2026). Voltam para esta tarefa: (3a) todos os formulários dos diálogos com botões e campos no tema do TuneTrail; (3b) Adicionar escuta: campo de data e hora, botão de adicionar responsivo e, abaixo, o fuso horário como campo opcional; (3c) Editar item com a identidade visual e o funcionamento do Koito, botões no mesmo estilo; (3d) Mesclar no padrão do Koito, com lista dinâmica e responsiva; (4) tooltip nos ícones da barra, no mesmo estilo da barra lateral. Fora desta tarefa: Excluir por perfil (ISSUE-143) e o × do histórico (ISSUE-141).

---
<!-- COMMENTS:END -->
