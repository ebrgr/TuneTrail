---
id: ISSUE-069
title: 069 — Tokens de design, três pontos de quebra e container queries
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-16T17:20:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-068
references:
  - backlog/evidencias/issue-069/README.md
  - backlog/evidencias/issue-069/responsive.json
  - backlog/evidencias/issue-069/revisao.md
  - backlog/evidencias/issue-069/antes/manifest.json
  - backlog/evidencias/issue-069/depois/manifest.json
priority: high
type: enhancement
ordinal: 69000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.1 e doc-031 fase 4. Muda o layout de propósito.

Passos:
1. Tokens: `--space-1..8`, escala tipográfica fluida `--step--1..5` com `clamp()`, raios e sombras.
2. Pontos de quebra únicos: 40rem, 64rem e 90rem (mobile-first, `min-width`); container queries em cards, heatmap e listas.
3. Eliminar larguras fixas em px de blocos (`.overview`, `.activity-card`, `.period-selector` e demais): `min()`, `clamp()`, grades `auto-fill`.
4. `check_web_style.py` rejeita `@media` fora da escala.
5. Capturas antes/depois e aceite visual do usuário.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Sem rolagem horizontal de página em 320, 360, 390, 768, 1024 e 1440px
- [x] #2 Somente os três pontos de quebra aprovados no CSS (verificado no ui.check)
- [x] #3 Nenhuma largura fixa em px para blocos de layout
- [ ] #4 Capturas antes/depois em Yuu e Pearl com aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-16T16:30:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-069` na branch `feat/issue-069-tokens-de-design-tres-pontos` criada a partir de `main`.

---

author: advisor
created: 2026-11-16T16:32:00Z

---

Parecer do Advisor (Modo B): REPROVADO COM RESSALVAS para o escopo original; APROVADO o plano da variante conservadora, condicionado ao refinamento dos ACs antes de implementar. [PLANO ISSUE-069] salvo no ai-memory: page_id 01a1e534-2654-7932-96de-7cb6ed44a064, plans/issue-069-advisor-preservacao-layout.md. Prioridade: preservar layout atual. Normalização integral para40/64/90rem altera com alta probabilidade (>95%, estimativa estática) os degraus existentes: tokens.css títulos em60rem; home.css gaps em48rem e limites em80/96rem; media-list.css .tracks em96rem; rewind.css em900px. Variante: tokens com valores exatos, escala futura sem adoção visual, min()/clamp() apenas quando geometricamente equivalentes; manter quebras legadas inventariadas e checker que impeça novas exceções, sem alegar AC#2 atendido. Fluidez aplicada, escala única estrita, auto-fill e container queries que mudem sizing devem ser adiados/separados e conciliados com071–074/078. Prova: preservar referência065, gerar antes/depois do HEAD atual em Yuu/Pearl,132 capturas por lado nas6 larguras mais limites de quebra, mesmo browser/OS/fontes/dados; gate0 pixels diferentes na dupla controlada, geometria e overflow estrito. Plano completo registra seletores/linhas, probabilidades por passo, contratos, arquivos e sequência. Nenhum código modificado, teste/captura nova executado ou status/AC alterado nesta consultoria.

---

author: codex
created: 2026-11-16T17:12:00Z

---

Escopo original autorizado: usuário revogou a preservação visual. Capturas BASE da referência ISSUE-065 concluídas antes das alterações em backlog/evidencias/issue-069/antes/ (132 imagens). Implementação direta sem delegação; aceite visual permanece pendente.

---

author: codex
created: 2026-11-16T17:20:00Z

---

Verificação: escopo original implementado após revogação da preservação visual. Tokens de espaço/tipografia fluida/raios/sombras, somente min-width 40/64/90rem nas media queries de layout, container queries em cards/heatmap/listas e blocos com limites intrínsecos. Checker integrado ao ui.check com 9 testes. make check aprovado (12/12 e2e), make ui.check aprovado, CSS parse e git diff --check aprovados; grafo AST atualizado (tokens.css ignorado pela heurística do extrator, revisado diretamente). BASE da ISSUE-065 capturada antes de editar: 132 PNGs em antes/. DEPOIS: 132 PNGs nos mesmos temas/larguras; 0 overflow horizontal. Auditoria adicional: 598 vistas/23 páginas/13 larguras/Yuu e Pearl, 0 falhas. Revisão direta/deslop registrada em revisao.md, sem delegação. ACs 1–3 verificados; AC4 permanece desmarcado, pois exige aceite visual do usuário. Sem push, merge, publish ou Done.

---
<!-- COMMENTS:END -->
