---
id: ISSUE-002
title: 002 — Atualizar caminhos da documentação no verificador de versão
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - docs
  - processo
milestone: m-0
dependencies: []
type: docs
ordinal: 68000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/002-versionamento-caminhos-docs-solved.md` — numeração local 002. Original removido em 12/11/2026; consulte o histórico do Git.

**Status: fechada e resolvida em 4 de novembro de 2026.** Sem publicação.

## Problema

Durante a validação do roadmap 1, `python3 scripts/version.py` falhou porque ainda
procurava `MULTIUSER_DESIGN.md` e sua versão inglesa na raiz. Os documentos já
haviam sido movidos para `docs-dev/` antes desta implementação.

## Correção

Atualizar somente os dois caminhos na lista `FILES` de `scripts/version.py`.
Preservar a versão RC2: não houve commit e não cabe incremento nesta etapa.

## Validação

Executar `python3 scripts/version.py` e confirmar a sincronização dos arquivos
atuais, sem usar `--bump` e sem alterar o histórico de releases.

Resultado registrado: `Koito 0.3.2-SUM-RC2: version files synchronized`.
<!-- SECTION:DESCRIPTION:END -->
