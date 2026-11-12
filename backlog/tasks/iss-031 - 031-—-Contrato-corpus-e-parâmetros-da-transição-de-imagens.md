---
id: ISS-031
title: 031 — Contrato, corpus e parâmetros da transição de imagens
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - imagens
milestone: m-3
dependencies: []
references: []
type: feature
ordinal: 31000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/031-contrato-corpus-imagens-go-solved.md` — numeração local 031. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada — resolvida em 06/11/2026**. Issue local, sem publicação.
Plano: [transição de imagens](<../docs/estudos/doc-014 - Plano-de-transição-—-imaging-webp-go-pure.md>). Fase 1; sem dependências novas.

## Trabalho

- Inventariar o comportamento atual da bimg e fixar versões candidatas de
  imaging/webp-go-pure, licenças e requisitos de Go.
- Criar corpus fictício JPEG, PNG, WebP legado e casos de transparência,
  EXIF 1–8, imagem pequena/retangular, origem diferente de zero, ICC/CMYK,
  animação, metadados, truncamento e dimensões inválidas.
- Definir formatos permitidos por conteúdo, política para animações e formatos
  adicionais hoje aceitos pela libvips; registrar efeitos nas mensagens de erro.
- Comparar imaging + codec com o pipeline atual, incluindo leitura de saídas
  antigas/novas em ambos, qualidade visual, latência, memória máxima, bytes e
  carga paralela. Repetições, máquina, versão, filtro/esforço e corpus precisam
  constar do relatório; números do protótipo com draw não validam imaging.
- Definir antes da integração os limites por operação e a capacidade global de
  trabalho: preservar limites atuais ou justificar a alteração, com critérios
  mensuráveis de qualidade, tempo, memória e tamanho de saída.

## Aceite

Contrato e parâmetros reproduzíveis registrados, diferenças classificadas e
critérios verificáveis para a 032. Corpus sanitizado e relatório em
`backlog/evidencias/transicao-imagens/ISSUE_031.md`; nenhuma leitura do banco
operacional, limpeza do cache do usuário ou migração de esquema.

## Entrega e validação

[Relatório e parecer do Advisor](../evidencias/transicao-imagens/ISSUE_031.md):
contrato observado, corpus fictício de 38 fixtures, 184 casos, 61 pares de
transformação e 138 saídas com qualidade medida. Módulo experimental isolado,
compilado sem CGO para o candidato; testes de bounds/alpha/EXIF1–8, race e vet
passaram. Contrato, hashes, medições, prévias e critérios calibrados preservados.

Advisor aprovou o encerramento como estudo, com limitações documentadas para a
032. Os alertas originais de qualidade não foram apagados. Produção continua
usando bimg/libvips; nenhum banco ou cache operacional foi alterado.
<!-- SECTION:DESCRIPTION:END -->
