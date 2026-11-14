---
id: ISSUE-032
title: 032 — Implementar pipeline Go com imaging e webp-go-pure
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - imagens
milestone: m-3
dependencies: []
type: feature
ordinal: 98000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/032-pipeline-imaging-webp-go-pure-solved.md` — numeração local 032. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada — resolvida em 06/11/2026**. Issue local, sem publicação.
Depende da [031](<issue-031 - 031-—-Contrato-corpus-e-parâmetros-da-transição-de-imagens.md>).
Plano: [transição de imagens](<../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>).

## Trabalho

- Criar componente compartilhado de inspeção/validação, orientação, corte,
  redimensionamento e encode WebP, com módulos fixados.
- Validar formato/bytes/dimensões antes de decode; aplicar a política de
  animações antes de chamar o codec e recusar os formatos fora do contrato.
- Usar imaging para transformações, preservando não ampliação, proporções e
  recorte definidos na 031; tratar EXIF/alpha e documentar limites de cor.
- Aplicar capacidade global e cancelamento da espera; não prometer interrupção
  de encode que a biblioteca não oferece. Preservar o orçamento de recursos.
- Garantir um único decoder WebP registrado: substituir o import antigo no
  `internal/summary/image.go` nesta issue. A refatoração restante fica na 035.
- Testar o componente sem CGO, com a versão Go declarada, corpus da 031 e
  concorrência; testar segurança dos readers e dimensões inválidas/truncadas.

## Aceite

Componente cumpre o contrato e os parâmetros da 031, com testes de conteúdo e
recursos, sem ativar um segundo decoder. Evidências no relatório `ISSUE_032.md`
da pasta de comparações da transição. O cache/admin ainda será integrado na 033;
não remover bimg enquanto houver consumidores.

## Entrega

[Implementação, testes, segurança e limites](../evidencias/transicao-imagens/ISSUE_032.md).
Componente isolado sem CGO, testado no Go 1.25, com admissão limitada, EXIF,
alpha, corte e encode WebP; 61 saídas passaram nos critérios de qualidade
calibrados. Decoder antigo substituído; cache e administração aguardam a 033.
<!-- SECTION:DESCRIPTION:END -->
