---
id: ISSUE-113
title: 113 — Colunas da Home lado a lado em larguras médias
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T22:40:00Z'
updated_date: '2026-11-14T22:41:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
type: bug
ordinal: 122000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Relatado pelo usuário: na Home, as colunas Artistas, Álbuns e Faixas mais ouvidos só ficavam lado a lado a partir de 1536px de largura, porque .tracks ocupava a linha inteira (width:100%) até o breakpoint de 96rem. Correção: .tracks recebe flex: 1 1 400px, então a coluna de faixas fica ao lado dos dois cards de 350px quando sobram 400px na linha (a partir de ~1318px de janela, já descontada a barra lateral) e cresce até 750px; em larguras menores continua quebrando para a linha de baixo em largura total, como antes. Acima de 96rem mantém 450px.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Três colunas lado a lado a partir de ~1318px, respeitando a barra lateral
- [x] #2 Abaixo disso as faixas quebram para a linha de baixo como antes
- [x] #3 Layout inalterado em 375, 1024 e 1920px; make check aprovado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T22:40:00Z

---

Início: branch `fix/issue-113-colunas-da-home-lado-a` criada a partir de `main`.

---

author: ebrgr
created: 2026-11-14T22:41:00Z

---

Verificação: make check aprovado (12/12 jornadas, referência sem diferenças); capturas Chrome antes/depois: 375, 1024 e 1920px idênticas, 1300px empilha, 1320/1366/1440px com três colunas.

---

author: ebrgr
created: 2026-11-14T22:41:00Z

---

Aceite: aprovada pelo usuário em 14/11/2026. Publicação local autorizada na mensagem: "1 e após commite e publique". Push não solicitado.

---

author: ebrgr
created: 2026-11-14T22:41:00Z

---

Publicação: versão 0.13 (tag `v0.13`), branch `fix/issue-113-colunas-da-home-lado-a` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
