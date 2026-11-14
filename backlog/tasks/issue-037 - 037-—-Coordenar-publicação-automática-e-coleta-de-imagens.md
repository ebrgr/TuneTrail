---
id: ISSUE-037
title: 037 — Coordenar publicação automática e coleta de imagens
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - imagens
milestone: m-3
dependencies: []
type: feature
ordinal: 103000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/037-publicacao-automatica-coleta-imagens-solved.md` — numeração local 037. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada — resolvida em 06/11/2026**. Issue local, sem publicação.
Origem: [revisão Advisor](../evidencias/transicao-imagens/ADVISOR_032_033.md).

## Problema

Produtores automáticos de artista/álbum podem publicar uma source antes da
associação no catálogo, sem participar da trava usada pela coleta de órfãos.
Uma coleta concorrente pode apagar o arquivo nessa janela. Esse risco antecede
a migração do codec; o redownload posterior depende da fonte remota.

## Trabalho e aceite

- Proteger publicação e associação nos caminhos automáticos sem segurar uma
  trava global durante downloads ou criar inversão de ordem de locks.
- Preservar cancelamento, `SkipCacheImage`, recuperação remota e tratamento de
  erro de banco; não remover arquivos possivelmente associados.
- Reproduzir a corrida em testes coordenados com dados fictícios, incluindo
  falha/cancelamento e concorrência entre produtores e coletor.
- Registrar o resultado em `backlog/evidencias/transicao-imagens/` e changelog
  se houver mudança perceptível. Pode ser tratada após a 034.

## Entrega

[Implementação e testes](../evidencias/transicao-imagens/ISSUE_037.md): reservas
por UUID protegem as cinco publicações automáticas sem manter a trava durante
rede ou banco. Testes coordenados, SQLite, cancelamento, falhas e redownload
passaram; a coordenação é limitada ao processo atual.
<!-- SECTION:DESCRIPTION:END -->
