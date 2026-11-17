---
id: ISSUE-072
title: '072 — Home e painel: KPIs, tocando agora, heatmap rolável e grade de destaques'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-17T20:48:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
references:
  - backlog/evidencias/issue-072/verification.md
  - backlog/evidencias/issue-072/review.md
  - backlog/evidencias/issue-072/browser-matrix.json
  - backlog/evidencias/issue-072/browser-adjustments.json
  - backlog/evidencias/issue-072/kpis-browser.json
priority: medium
type: feature
ordinal: 72000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.4 (Home).

Passos:
1. Linha de KPIs em cards (minutos, escutas, artistas, sequência) com variação contra o período anterior.
2. "Tocando agora" no topo, quando houver.
3. Heatmap que rola horizontalmente em telas pequenas (`overflow-x`, `scroll-snap`), ancorado na semana atual.
4. Top artistas/álbuns em grade `auto-fill`: 2 colunas no celular, 4–6 no desktop.
5. Seletor de período sticky.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 KPIs com variação calculada no servidor e testada
- [x] #2 Heatmap legível em 320px sem esconder semanas
- [x] #3 Grade de destaques adaptável sem largura fixa
- [x] #4 Capturas e aceite visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-17T18:13:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-072` na branch `feat/issue-072-home-e-painel-kpis-tocando` criada a partir de `main`.

---

author: codex
created: 2026-11-17T18:20:00Z

---

Escopo reconciliado: as decisões aceitas na ISSUE-154 prevalecem sobre doc-010 §3.4 e os passos antigos da 072. Heatmap mantido fluido, máximo 35rem, 36/26 semanas conforme o cartão, sem rolagem; grade auto-fill preserva textos em uma coluna por card e só permite duas quando cada item tem 22rem. Mantidas lateral de ícones, menu (+) agrupado e últimas reproduções em toda a grade. KPIs com intervalos móveis locais atuais; comparação com intervalo anterior adjacente de igual duração em segundos, sem sobreposição; sequência é a maior sequência de dias civis dentro de cada intervalo. Todo o tempo não compara; base zero com atividade mostra Nova atividade, ambos vazios 0%.

---

author: codex
created: 2026-11-17T18:22:00Z

---

Revisão local executada diretamente por codex, sem delegação: APROVADO; diff e impacto estrutural conferidos via graphify. Limitação de legibilidade do heatmap registrada nas evidências; nenhuma rolagem implementada. AC2 e AC4 permanecem pendentes.

---

author: codex
created: 2026-11-17T18:22:00Z

---

Verificação: make check e make ui.check verdes; 12/12 jornadas e2e; 105 combinações de perfis/períodos/larguras entre 320 e 1920px sem rolagem horizontal. Seis capturas Yuu antes/depois e referência em backlog/evidencias/issue-072/. AC1 e AC3 verificados; AC2 pendente pela leitura individual das células de ~6,6px em 320px, AC4 reservado ao aceite visual. Pergunta: manter essa leitura ou planejar visão compacta opcional de 13 semanas, sem rolagem? Conflitos com doc-010/passos antigos resolvidos conforme decisões da ISSUE-154. Servidor de teste encerrado; banco original preservado.

---

author: codex
created: 2026-11-17T19:39:00Z

---

Decisão do usuário aplicada: restaurar seis totais sempre históricos; a ordem final em telas largas é TOTAIS À ESQUERDA e HEATMAP À DIREITA (substitui a primeira instrução deste ajuste). Em telas menores mantém heatmap seguido dos totais. Tamanho do heatmap aprovado pelo usuário; AC2 marcado após nova verificação em 320px sem rolagem. Tooltips data-tip em CSS puro, sem title nativo, alinhados para dentro; tabela acessível mantida. Variações zero ocultas; rótulos en/pt-BR existentes reutilizados. AC4 permanece desmarcado.

---

author: codex
created: 2026-11-17T19:39:00Z

---

Verificação: ajustes implementados e revisados diretamente por codex, sem delegação. make check e make ui.check verdes, 12/12 jornadas e2e; referência regenerada em backlog/evidencias/issue-072/reference. Capturas finais depois2-390.png, depois2-768.png e depois2-1440.png em backlog/evidencias/issue-072/. Browser confirmou ausência de rolagem e tooltips dentro do cartão em 320/390/768/1440px; totais à esquerda no desktop. AC1/AC2/AC3 verificados; AC4 reservado ao aceite visual. Banco original preservado e servidor encerrado.

---

author: codex
created: 2026-11-17T20:35:00Z

---

Verificação: decisão mais recente aplicada — variação em texto PEQUENO ao lado do número, alinhada pela base, com espaço de 12px e quebra apenas quando necessário. Textos en/pt-BR: N% More/Mais, N% Less/Menos, Same/Igual e New/Novo; sem sinais e sem vs. período anterior. Zero agora mostra Igual, substituindo a regra anterior de ocultar; Tudo não compara. Novo mantido quando a base anterior é zero e há atividade atual, aguardando confirmação do rótulo. make check e make ui.check verdes, 12/12 jornadas e2e; referência regenerada e grafo atualizado. Em 390/1440px, os quatro cards mantêm comparação na mesma linha e não transbordam. Novas capturas dispensadas conforme última instrução do usuário. AC4 permanece desmarcado. Servidor temporário encerrado e banco original preservado.

---

author: ebrgr
created: 2026-11-17T20:47:00Z

---

Aceite do usuário (2026-11-17) após ver a Home em instância local com dados reais (cópia) e em banco descartável, com os ajustes pedidos: totais históricos à esquerda e heatmap à direita, tooltip próprio nas células, KPIs com comparação pequena ao lado do número (N% Mais, N% Menos, Igual), sem rótulo Novo, e duas linhas de dois cards até 64rem. Heatmap em 320px sem rolagem, com células fluidas aprovadas pelo usuário. Publicação com push autorizada pelo usuário.

---

author: ebrgr
created: 2026-11-17T20:48:00Z

---

Publicação: versão 0.33 (tag `v0.33`), branch `feat/issue-072-home-e-painel-kpis-tocando` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
