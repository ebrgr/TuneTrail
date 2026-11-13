---
id: ISSUE-027
title: 027 — Usar Go por padrão e desligar o fallback React
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-2
dependencies: []
references: []
type: feature
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/027-go-padrao-react-desligado-solved.md` — numeração local 027. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida em 05/11/2026**. Issue local; não publicar.

## Solicitação

Ativar a interface Go por padrão e impedir que o modo Go execute a SPA React
implicitamente. A seleção explícita de React é mantida para compatibilidade e
para comparações futuras; ela só inicia com `KOITO_WEB_UI=react`.

## Trabalho e aceite

- Configuração ausente seleciona Go; `go` e `react` explícitos permanecem válidos.
- O modo Go não entrega o HTML nem assets da SPA como fallback de rotas ausentes;
  recursos próprios de Go, favicon, `/public`, API e ListenBrainz permanecem disponíveis.
- Testar default, Go, React explícito, rotas válidas/inexistentes e assets; atualizar
  documentação, changelog e roadmap. Preservar SQLite v3 e não remover o cliente.

## Resultado

[Configuração F5, verificação e limites](../evidencias/roadmap-06/ISSUE_027.md).
<!-- SECTION:DESCRIPTION:END -->
