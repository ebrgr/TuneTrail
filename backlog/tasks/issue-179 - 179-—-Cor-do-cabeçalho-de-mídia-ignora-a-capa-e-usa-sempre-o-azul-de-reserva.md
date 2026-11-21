---
id: ISSUE-179
title: 179 — Cor do cabeçalho de mídia ignora a capa e usa sempre o azul de reserva
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-21T14:02:00Z'
updated_date: '2026-11-21T14:07:00Z'
labels:
  - imagens
dependencies: []
type: bug
ordinal: 192000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Causa: imagecache.HeaderAccent (imagecache/color.go) só aceita URLs com 4 partes (/image/<uuid>/<arquivo>). Desde a ISSUE-161 as capas têm URL /image/profile/<id>/<uuid>/<arquivo> (6 partes), então a função devolve sempre o fallback #253044 sem ler a cor média da capa. Sintoma: o cabeçalho das páginas de artista, álbum e faixa fica sempre azul-acinzentado (--media-accent:#253044), qualquer que seja a capa. O teste em imagecache/color_test.go usa o formato antigo de URL e por isso não pegou a regressão.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 HeaderAccent aceita a URL /image/profile/<id>/<uuid>/<arquivo> e devolve a cor derivada da capa
- [x] #2 O formato antigo /image/<uuid>/<arquivo> continua aceito
- [x] #3 O teste usa a URL real do produto e URLs inválidas continuam no fallback
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Correção: imagecache/color.go ganhou artworkID, que aceita /image/<uuid>/<arquivo> e /image/profile/<id>/<uuid>/<arquivo>. Teste TestHeaderAccentProfileURL cobre os dois formatos e URLs inválidas. Verificado: go test ./imagecache e ./internal/webui verdes. Não verificado: tela no navegador (o servidor em 127.0.0.1:4110 roda o binário antigo) e make check completo.
<!-- SECTION:NOTES:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-21T14:03:00Z

---

Início: branch `fix/issue-179-cor-do-cabecalho-de-midia` criada a partir de `main`.

---

author: claude
created: 2026-11-21T14:07:00Z

---

Aceite: aprovada pelo usuário em 21/11/2026, após ver a correção num build temporário com a ISSUE-180.

---
<!-- COMMENTS:END -->
