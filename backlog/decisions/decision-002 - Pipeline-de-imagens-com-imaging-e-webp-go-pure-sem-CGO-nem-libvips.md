---
id: decision-002
title: Pipeline de imagens com imaging e webp-go-pure, sem CGO nem libvips
date: '2026-11-12T23:40:00Z'
status: accepted
---
## Context
libvips/bimg exigiam CGO e bibliotecas nativas na imagem Docker (doc-012, doc-013).


## Decision
Usar `imaging` + `webp-go-pure`: JPEG, PNG e WebP estático de entrada; WebP de saída; `x/image` para texto (ISS-031–ISS-037, doc-014).


## Consequences
Build sem CGO para amd64/arm64; imagens animadas e outros formatos são rejeitados; o detector de corrida opcional ainda requer compilador C.

