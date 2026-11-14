---
id: ISSUE-107
title: 107 — Busca instantânea e edição de faixas sem recarregar a página
status: Aberta
assignee: []
created_date: '2026-11-14T00:55:00Z'
updated_date: '2026-11-14T15:10:00Z'
labels:
  - ui
  - admin
milestone: m-10
dependencies:
  - ISSUE-106
  - ISSUE-067
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
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
- [ ] #1 Busca mostra resultados enquanto digita, sem recarregar, descartando respostas antigas
- [ ] #2 Busca utilizável só por teclado e anunciada por leitor de tela
- [ ] #3 Adicionar/remover apelido ou artista atualiza só a seção, sem perder rolagem e foco
- [ ] #4 Sem JavaScript, as ações voltam à âncora da seção editada
- [ ] #5 Fragmentos respeitam sessão, CSRF, token de uso único, permissões e perfil (testes)
- [ ] #6 JS novo abaixo de 4 KB comprimido; tempos antes/depois registrados em backlog/evidencias/ui-responsiva
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
<!-- COMMENTS:END -->
