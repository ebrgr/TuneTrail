---
id: ISSUE-006
title: 006 — Migrar páginas públicas de artistas, álbuns e faixas para Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
references: []
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/006-paginas-midia-go-solved.md` — numeração local 006. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida — aceite confirmado pelo usuário em 05/11/2026**. Issue local, sem publicação.
Issue local, sem publicação.
Plano: [roadmap 3](<../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>).

## Objetivo

Entregar as três páginas com dados reais e escopo por perfil, reaproveitando
layout, serviços e consultas existentes. Primeira fatia da etapa 3; ativação
pública condicionada aos gráficos da 007 e à integração da 009.

## Trabalho

- Resolver slug e validar ID positivo int32 antes das consultas; manter
  `requireRelated` e não habilitar `AllowUnplayed` para a página da faixa.
- Compor cabeçalho all_time: título, tipo, capa, rank, plays, duração ouvida,
  primeira escuta, artistas/álbum e link externo com tipo MusicBrainz correto.
- Preservar Various Artists e lista real de artistas; distinguir metadados
  compartilhados de estatísticas pessoais. Não inventar painel público de aliases.
- Artista/álbum: oito top tracks all_time. Histórico: onze registros filtrados,
  com links corretos para rankings/histórico Go, respeitando slug, IDs e base 1.
- Artista: até 99 álbuns associados e ouvidos pelo perfil; oferecer acesso ao
  restante pelo ranking filtrado. Não apresentar isso como discografia completa.
- Reusar tocando agora por perfil e só mostrá-lo quando corresponde à entidade.
- Propagar falhas de Scan de contagem/tempo/rank dos detalhes SQLite. Distinguir
  `db.ErrNotFound` de falha interna no HTML e avaliar o ajuste dos handlers JSON
  com regressões de compatibilidade. Não alterar migrações.
- Implementar blocos opcionais com erro explícito; capa ausente/primeira escuta
  ausente não podem virar conteúdo fictício. Escapar nomes/aliases/URLs.
- Acrescentar composição/modelos/templates organizados em `internal/webui/`,
  sem HTTP à própria API e sem duplicar SQL.

## Aceite

Testes em SQLite v3 descartável para os três tipos; valores equivalentes à API,
rank empatado, múltiplos artistas, compilação, imagem ausente, IDs inválidos,
item do outro perfil, 404 real versus 500 e falha de bloco sem zero falso.
Mesmo item deve poder ter valores diferentes em duas abas/perfis. Ações atuais
continuam acessíveis na entrega integrada da 009.

## Evidências futuras

Pares desktop/celular de cabeçalhos e regiões abaixo da dobra, registrados em
[comparisons/roadmap-03](../evidencias/roadmap-03/README.md). Nenhuma implementação
ou screenshot nova de página Go de mídia foi feita na abertura desta issue.

## Marco de implementação — 04/11/2026

Composição, modelos, templates e CSS das três páginas implementados. Testes Go
com SQLite descartável, comparação HTML/API, regressões do cliente, testes de
corrida e validações HTTP/multiusuário passaram. Corrigida propagação de falhas
nos detalhes SQLite e distinção 400/404/500 na API.

Conforme orientação posterior do usuário, a conferência de layout e screenshots
fica para o **final da issue 009**. Sem aceite visual nesta etapa; a issue permanece
aberta. Ativação das rotas e gestão React seguem na 009, como planejado.

[Resumo da implementação e evidências](../evidencias/roadmap-03/ISSUE_006.md).

## Encerramento

O usuário confirmou em 05/11/2026 que todas as issues desta etapa estão completas.
Este aceite encerra a issue. Os registros anteriores permanecem como histórico
das verificações executadas e das verificações manuais que não foram realizadas;
o encerramento não as apresenta como testes executados.
<!-- SECTION:DESCRIPTION:END -->
