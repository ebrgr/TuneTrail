---
id: ISSUE-107
title: 107 — Busca instantânea
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:55:00Z'
updated_date: '2026-11-15T11:46:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-106
  - ISSUE-067
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
  - backlog/evidencias/issue-107/advisor-inicial.md
  - backlog/evidencias/issue-107/reviewer.md
  - backlog/evidencias/issue-107/busca_instantanea.log
  - backlog/evidencias/ui-responsiva/issue-107-tempos.md
priority: high
type: feature
ordinal: 107000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: comparação com o Koito React (evidência em backlog/evidencias/ui-responsiva/comparacao-koito-react.md); ROADMAP item 8 e ISSUE-056. Assume a parte de busca da ISSUE-075.

Diagnóstico: o servidor Go responde em 1–4 ms, como o do Koito. A lentidão percebida vem de cada busca e cada ação de edição refazer a página inteira: a busca só roda ao pressionar Enter e navega para uma página completa; a edição de faixa é uma página com 42 formulários em que adicionar um apelido custa POST + 303 + GET de 36 KB, volta ao topo e perde o foco; procurar artista ou destino de mesclagem também recarrega tudo.

Princípio: sem framework. O servidor continua renderizando HTML; um script pequeno (meta: menos de 4 KB comprimido) troca só o trecho afetado. Sem JavaScript, tudo funciona como hoje.

Passos — busca:
1. `/{slug}/search?q=…&partial=1` devolve só o bloco de resultados (artistas, álbuns, faixas), reaproveitando `listening.ReadSearch`.
2. No diálogo de busca: resultados enquanto digita, com debounce de ~120 ms, `AbortController` para descartar respostas antigas, mínimo de 2 caracteres e limite de itens.
3. Combobox ARIA: setas para navegar, Enter abre o item, Esc fecha; link "ver todos" para a página completa; anúncio da contagem para leitor de tela.

Passos — edição de faixa:
4. Sem JS, vitória imediata: o 303 depois de cada ação aponta para a âncora da seção editada (`#aliases`, `#artists`, `#metadata`), preservando a posição.
5. Com JS: formulários marcados com `data-enhance` são enviados por `fetch` com o mesmo corpo (CSRF incluído). O servidor responde só a seção alterada (apelidos, artistas, MBID, escutas), já com novo `submission_token` de uso único; erros 4xx voltam como a seção com a mensagem, no mesmo lugar.
6. Procurar artista para adicionar e destino de mesclagem: combobox com fragmento (`?artist_query=…&partial=1`, `?merge_query=…&partial=1`) no lugar do recarregamento.
7. Botão em estado de carregamento e desabilitado durante o envio; foco devolvido ao campo ou ao item alterado.
8. A lista de escutas da faixa (25 por página) passa a ser uma seção carregada separadamente, para não ser recalculada a cada ação de apelido ou artista.

Segurança e isolamento: fragmentos seguem as mesmas regras de sessão, origem, CSRF, permissões e escopo de perfil das páginas completas; testes cobrindo cada um. Revisão do Advisor recomendada (AGENTS.md: rotas → @Opus effort=medium, @Sol effort=low).

Recomendação de ordem: depois da ISSUE-106. Pode vir antes da reorganização dos arquivos (066–068), porque mexe sobretudo em handlers e JS; se a 066 estiver em andamento, fazer depois dela para não editar templates em linha única.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Busca mostra resultados enquanto digita, sem recarregar, descartando respostas antigas
- [x] #2 Busca utilizável só por teclado (combobox) e anunciada por leitor de tela
- [x] #3 Até 5 itens por tipo e link "ver todos" para a página completa; sem JavaScript a busca funciona como hoje
- [x] #4 Fragmento respeita perfil e sessão, recusa pedido fora de contexto e usa cache privado (testes)
- [x] #5 JS novo abaixo de 4 KB comprimido; tempos antes/depois registrados em backlog/evidencias/ui-responsiva
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:06:00Z

---

A edição sem recarregar é consumida pelos diálogos da ISSUE-108 (barra de ações nas páginas de mídia): esta tarefa entrega o mecanismo de fragmentos e o envio de formulários por seção; a 108 divide a página de gestão por tarefa e abre cada uma em diálogo.

---

author: claude
created: 2026-11-14T15:10:00Z

---

Nova ordem da Fase 1 aprovada pelo usuário em 14/11/2026: os assets passam a ser servidos do disco pela ISSUE-066 e os templates são reformatados pela ISSUE-067, então as tarefas de fluidez vêm depois delas para evitar retrabalho e conflitos.

---

author: ebrgr
created: 2026-11-15T02:12:00Z

---

Início: branch `feat/issue-107-busca-instantanea-e-edicao-de` criada a partir de `main`.

---

author: claude
created: 2026-11-15T02:14:00Z

---

Parecer do Advisor (Opus, medium): REPROVADO COM RESSALVAS — token de uso único por página, limite global de tickets, serialização de envios, nome no cabeçalho em alias-primary e aviso consumido no fragmento; recomenda dividir em busca (107a) e edição (107b). Aguardando decisões do usuário.

---

author: claude
created: 2026-11-15T02:15:00Z

---

Decisões do usuário em 14/11/2026: dividir conforme o Advisor — a 107 fica só com a busca instantânea (5 itens por tipo + ver todos); a edição sem recarregar vai para a ISSUE-124 (envio de capa continua recarregando); a ISSUE-108 passa a depender da 124.

---

author: ebrgr
created: 2026-11-15T02:31:00Z

---

Verificação: make check 12/12 e referência sem diferenças; validador de cache Chrome/Firefox/HTTPS PASS; testes do fragmento (5 por tipo, curto/longo, perfil inexistente, navegação direta recusada, cabeçalhos); roteiro busca_instantanea.py PASS em 5 execuções (resposta atrasada descartada, setas, Enter, Esc duplo, anúncio, foco); fragmento 0,85 ms/1,8 KB contra 1,81 ms/25 KB da página; search.js 1,6 KB gzip; Reviewer APROVADO. Falta o aceite visual (capturas em backlog/evidencias/issue-107/).

---

author: claude
created: 2026-11-15T11:46:00Z

---

Ajustes pedidos pelo usuário antes do aceite, comparando com o Koito original (capturas em backlog/evidencias/issue-107/koito/): diálogo ancorado no topo com o campo fixo (posição constante com e sem resultados e ao rolar), resultados em cartões por tipo com título acima (artistas em grade com imagem e nome, álbuns e faixas com miniatura e artistas), botão Buscar escondido com JavaScript; mantidos 5 por tipo e ver todos (decisão do usuário). Reviewer: item obrigatório (opção ativa coberta pelo cabeçalho) corrigido e coberto pelo roteiro.

---
<!-- COMMENTS:END -->
