---
id: ISS-036
title: 036 — Admissão e cancelamento de imagens
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - imagens
milestone: m-3
dependencies: []
references:
  - docs-dev/issues/036-admissao-cancelamento-imagens-solved.md
type: feature
ordinal: 36000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/036-admissao-cancelamento-imagens-solved.md](../../docs-dev/issues/036-admissao-cancelamento-imagens-solved.md) — numeração local 036.

Status: **fechada — resolvida em 06/11/2026**. Issue local, sem publicação.
Origem: [parecer Advisor das 032/033](../../docs-dev/comparisons/transicao-imagens/ADVISOR_032_033.md).
Prioridade: corrigir antes da [034](../../docs-dev/issues/034-build-distribuicao-sem-cgo-solved.md).

## Trabalho e aceite

- Limitar e permitir cancelamento da admissão de uploads administrativos antes
  de interpretar corpo, ler arquivos ou aguardar travas; preservar associação e
  coordenação com coleta de órfãos. A capacidade global de imagens decodificadas
  permanece separada do limite das requisições administrativas.
- Separar o contexto dos participantes do trabalho compartilhado em derivações e
  redownload: espera cancelável por cliente, limite de trabalhos distintos e prazo
  próprio; encerrar trabalho quando não houver participantes, sem perder controle
  de codecs que não aceitam interrupção.
- Padronizar erro de arquivo grande e de formato recusado em upload e download.
- Testar concorrência, cancelamento, sobrecarga, erros HTTP, imagens válidas,
  compatibilidade de cache, proteção de associação e ausência de arquivos parciais.
  Registrar resultados e limitações. O risco histórico de coleta em produtores
  automáticos do catálogo fica em demanda separada, conforme o plano acordado.

## Entrega

[Implementação, testes e limitações](../../docs-dev/comparisons/transicao-imagens/ISSUE_036.md).
Admissão antes da leitura HTTP, cancelamento independente por participante e
padronização de erros foram implementados; suíte Go, race e Go 1.25 sem CGO
passaram no escopo documentado. Risco histórico segue na issue 037.
<!-- SECTION:DESCRIPTION:END -->
