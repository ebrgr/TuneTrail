---
id: ISSUE-073
title: 073 — Listas, histórico e páginas de mídia compactas, com menu de ações
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-18T21:16:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-071
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
  - backlog/evidencias/ISSUE-073/verificacao.md
priority: medium
type: feature
ordinal: 73000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.4 (Listas e Mídia).

Passos:
1. Linha compacta no celular: capa, título/artista e horário relativo.
2. Ações administrativas em menu `⋯` (`<details>` ou `popover`), no lugar do link "Manage" repetido.
3. Mídia: cabeçalho com capa e cor de destaque extraída no servidor ao gerar o WebP; abas Visão geral · Faixas · Álbuns · Histórico · Gráficos.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Histórico legível em 320px com ações acessíveis por teclado
- [x] #2 Cor de destaque calculada no servidor, com contraste verificado
- [x] #3 Abas funcionam sem JavaScript (links/âncoras)
- [x] #4 Capturas e aceite visual
- [x] #5 Cabeçalho de mídia próximo ao Koito: capa grande, tipo, título grande com posição no ranking, contagens e fundo em degradê da cor da capa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:06:00Z

---

Comparação com o Koito (/track/23): o cabeçalho de mídia deve seguir o original — capa grande (equivalente a 385px no desktop), tipo, título grande com a posição no ranking (#N), contagens e 'ouvindo desde', e fundo em degradê da cor média da capa (calculada no servidor, já prevista no passo 3). A barra de ícones de gestão e a divisão da página de gestão ficam na ISSUE-108.

---

author: ebrgr
created: 2026-11-18T18:17:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-073` na branch `feat/issue-073-listas-historico-e-paginas-de` criada a partir de `main`.

---

author: codex
created: 2026-11-18T18:32:00Z

---

Verificação do executor: histórico compacto a 320px; reticências nativas com/sem scripts e confirmação existente dentro do menu; cabeçalhos com cor média persistida no processamento WebP, contraste mínimo 7:1 e cinco links de seção sem JS. make check aprovado, incluindo 13/13 jornadas E2E. Orca: 320px sem overflow, Enter/Tab/Escape e foco após exclusão; temas Yuu/Pearl; cinco âncoras com scripts bloqueados por CSP. Testes de RGB/alpha, cache legado/fallback, contraste em 4.096 cores e destinos/escopo/escaping do histórico. Evidências e limitações em backlog/evidencias/ISSUE-073/verificacao.md. Referência determinística atualizada sem substituir a histórica da ISSUE-072. Barra e páginas de gestão preservadas. Mantido Em andamento, aguardando Reviewer do agente responsável e posterior aceite visual; sem push/merge/publish/Done.

---

author: claude
created: 2026-11-18T21:14:00Z

---

Aceite explícito do usuário em 2026-11-18 após ver a interface rodando na fixture local (histórico compacto, menu de reticências, cabeçalho de mídia com degradê e abas sem JavaScript). ACs verificados pelas evidências em backlog/evidencias/ISSUE-073/verificacao.md; AC4 pelo aceite visual do usuário. Nota: o Reviewer formal não foi executado antes do aceite, por decisão do usuário.

---

author: ebrgr
created: 2026-11-18T21:16:00Z

---

Publicação: versão 0.35 (tag `v0.35`), branch `feat/issue-073-listas-historico-e-paginas-de` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
