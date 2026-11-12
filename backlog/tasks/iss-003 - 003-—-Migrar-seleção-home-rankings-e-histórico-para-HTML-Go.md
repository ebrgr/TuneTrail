---
id: ISS-003
title: 003 — Migrar seleção, home, rankings e histórico para HTML Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
references:
  - docs-dev/issues/003-paginas-go-dados-reais-solved.md
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/003-paginas-go-dados-reais-solved.md](../../docs-dev/issues/003-paginas-go-dados-reais-solved.md) — numeração local 003.

Status: **fechada e resolvida em 04/11/2026**. Issue local, sem publicação.

Plano: [roadmap 2](../../docs-dev/ROADMAP_02_PAGINAS.md).

## Trabalho

- Separar renderização compartilhada de fixtures e referência do passo 1.
- Criar handlers reais para as seis páginas, recebendo serviços de leitura.
- Resolver slug para ID antes de cada consulta pessoal; nunca usar escopo global.
- Reutilizar catálogo, imagens e consultas, sem HTTP à própria API.
- Preservar diferenças entre estatísticas gerais, atividade e rankings por período.
- Entregar estados vazio/erro, paginação e links reais, sem controles fictícios.
- Manter SQLite no esquema consolidado v3 e a prévia isolada disponível.

## Aceite

Valores HTML equivalentes à API em bancos de teste, isolamento entre perfis,
testes de renderização e capturas comparativas documentadas. A ativação depende
dos contratos da [004](../../docs-dev/issues/004-contratos-paginas-solved.md) e da convivência da
[005](../../docs-dev/issues/005-convivencia-go-react-solved.md). Não substituir as rotas da aplicação nesta
etapa de análise (restrição da proposta original; integração concluída na 005).

## Marco da implementação — núcleo da 003

Handlers HTML e composição compartilhada com a API implementados. Testes em
SQLite descartável passaram para as seis páginas, isolamento, equivalência das
estatísticas, perfil vazio e erros. Ativação e evidências visuais dependem da
004 e da 005, que serão executadas na sequência.

## Encerramento

Páginas reais entregues em `internal/webui/`, com serviços compartilhados e
escopo por ID do perfil, estados vazio/erro, imagens e links reais. Estatísticas
HTML/API e isolamento validados em SQLite v3 descartável. A 004 e a 005 também
foram concluídas, permitindo ativação opcional com `KOITO_WEB_UI=go`.

Validação e capturas: [relatório do roadmap 2](../../docs-dev/comparisons/roadmap-02/README.md).
`make check ui.check`, build React/Go, testes de corrida focados e validações
HTTP/multiusuário passaram. Sem commit ou publicação.
<!-- SECTION:DESCRIPTION:END -->
