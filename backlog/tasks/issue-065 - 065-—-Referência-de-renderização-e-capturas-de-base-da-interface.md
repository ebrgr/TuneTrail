---
id: ISSUE-065
title: 065 — Referência de renderização e capturas de base da interface
status: Done
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T14:56:00Z'
labels:
  - ui
  - validacao
milestone: m-10
dependencies:
  - ISSUE-061
references:
  - backlog/evidencias/issue-061/fontes-retrospectiva.md
  - backlog/evidencias/ui-responsiva/base/README.md
  - backlog/evidencias/ui-responsiva/base/reference-manifest.json
  - backlog/evidencias/ui-responsiva/base/capturas/manifest.json
  - backlog/evidencias/ui-responsiva/base/verification.json
priority: high
type: chore
ordinal: 65000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 §Sequência de execução e validação. Referência da interface atual, antes da reorganização dos arquivos. Gerar HTML/CSS/JS/assets de prévia, páginas públicas e administração em fixtures fictícias isoladas; comparar bytes e indentação de HTML; capturar seis larguras (320,360,390,768,1024,1440) nos temas Yuu e Pearl; registrar tipografia computada e fontes efetivamente usadas na retrospectiva em três larguras. Evidências em backlog/evidencias/ui-responsiva/base/. A coleta não decide novo alvo visual nem antecipa a mudança funcional da 066; a 061 conserva seu próprio aceite.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 HTML e CSS de referência gerados por comando repetível
- [x] #2 Comparação byte a byte e normalizada disponível para as próximas tarefas
- [x] #3 Capturas de base nas seis larguras e dois temas registradas
- [x] #4 Referência tipográfica da retrospectiva registrada em dados fictícios, mesmo tema e viewport, em pelo menos três larguras; alvo visual posterior depende de aceite.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Atualização do estudo 061: capturar a estrutura atual antes de mover todos os arquivos para assets/ (incluindo assets/templates/). Acrescentar referência tipográfica da retrospectiva para a 070; comparar no mesmo tema/viewport/dados fictícios, registrando família, peso, tamanho, caixa, line-height e fonte efetivamente renderizada. Fontes locais e correção visual serão mudanças deliberadas da 070, não exigem igualdade byte a byte com o CSS antigo.
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-14T14:22:00Z

---

Refinamento do plano para refletir assets/ como diretório único e a comparação de fontes solicitada pelo usuário na ISSUE-061.

---

author: Codex
created: 2026-11-14T14:37:00Z

---

Início autorizado pelo usuário: preparar referências da interface atual em fixtures descartáveis, sem antecipar implementação da 066. Dependência 061 continua aguardando aceite; nenhuma decisão de layout visual é inferida desta tarefa.

---

author: Codex
created: 2026-11-14T14:52:00Z

---

Verificação: referência de 23 HTMLs e 113 arquivos em fixtures fictícias, relógio/UTC fixos, tokens de formulário mascarados. make ui.reference gerado sob TZ externos Honolulu e Tóquio; make ui.compare resultou 0, byte a byte idêntico. Comparador testa conteúdo, indentação, arquivos ausentes/extra e fontes, integrado em ui.check. 132 capturas (11 páginas representativas × seis larguras × Yuu/Pearl), 0 overflow horizontal, 36 registros de tipografia com fontes reais via CDP em três larguras; dimensões PNG e amostras visuais conferidas. make check aprovado (11/11 e2e); Reviewer Opus aprovou condicionado a UTC, aplicado e verificado. Grafo atualizado. Limites documentados: bucket de interesse agregado na fixture, fontes externas ainda sujeitas à rede com falhas registradas, capturas representativas de 11 das 23 páginas. Sem mudança de runtime, dados reais ou publicação. Aguarda aceite da entrega; 061 permanece com seu aceite próprio.

---

author: Codex
created: 2026-11-14T14:56:00Z

---

Aceite: ISSUE-065 aprovada explicitamente pelo usuário em 14/11/2026, com solicitação de commit. Entrega de testes, ferramentas e evidências, sem alteração de runtime e sem incremento de versão; nenhum push autorizado.

---
<!-- COMMENTS:END -->
