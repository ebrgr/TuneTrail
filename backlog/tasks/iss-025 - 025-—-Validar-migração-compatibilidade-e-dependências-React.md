---
id: ISS-025
title: 025 — Validar migração, compatibilidade e dependências React
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
  - validacao
milestone: m-2
dependencies: []
references: []
type: task
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/025-migracao-compatibilidade-react-solved.md` — numeração local 025. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida em 05/11/2026, com limitações registradas**. Issue local, sem publicação.
Plano: [roadmap 6](<../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>). Usa a fixture da 022.

## Trabalho

- Executar jornadas web e administrativas em cópias SQLite v2 migradas para v3,
  nos modos React/API e Go; registrar processo de migração, backup, contagens e
  integridade. Não alterar migrações existentes ou dados operacionais.
- Inventariar métodos e destinos de rotas legadas/canônicas, ações, API, assets,
  fallback para SPA, `/theme-helper`, `/favicon.svg`, `/public` e links diretos;
  conferir status, conteúdo, autenticação e comportamento de URL ausente.
- Mapear dependências no Dockerfile, Makefile, CI, docs, configuração, preview,
  scripts, fontes e empacotamento. Distinguir assets Go embutidos de arquivos
  servidos externamente; incluir estratégia futura para instalação e rollback.
- Registrar destino das lacunas 018–021 de fusos, gráficos e I/O que afetam a
  decisão, com justificativa; não classificá-las como testadas sem reprodução.

## Aceite

Jornadas migradas e evidências de integridade documentadas; dependências e
compatibilidade têm destino explícito. Nenhuma remoção ou troca de padrão é
executada nesta issue.

## Resultado

[Migração, rotas, inventário e limites](../evidencias/roadmap-06/ISSUE_025.md).
A falha reproduzida no helper React permanece aberta na issue 028.
<!-- SECTION:DESCRIPTION:END -->
