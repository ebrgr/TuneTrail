---
id: ISS-018
title: 018 — Corrigir e validar fusos, intervalos e agregados civis
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
references:
  - docs-dev/issues/018-fusos-intervalos-agregados-solved.md
type: task
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/018-fusos-intervalos-agregados-solved.md](../../docs-dev/issues/018-fusos-intervalos-agregados-solved.md) — numeração local 018.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 5](../../docs-dev/ROADMAP_05_VALIDACAO.md). Dependências: 016; matriz de permissões da 017 disponível para integração final.

## Trabalho

- Transformar as cinco reproduções do Advisor em regressões que falham antes da correção; ver evidências do roadmap 5.
- Corrigir atividade/streaks que arredondam para hora UTC antes da data local, dias ativos agrupados por UTC, exclusão do último segundo mensal e seleção de mês/semana sem ano.
- Aplicar a mesma regra temporal em serviços compartilhados/SQLite e verificar Go/JSON com oráculo independente; incluir filtros artista/álbum/faixa e perfil.
- Fixar relógio nos testes com o mínimo de adaptação necessária. Testar UTC, São Paulo histórico, New York, Kathmandu e Lord Howe, com timezone do processo diferente.
- Cobrir ForceTZ > query > cookie > processo, aliases e inválidos; fronteiras bissextas, semanas ISO1/53, viradas locais, from/to, futuro e all-time.
- Rever sinais inclusivo/exclusivo nos limites SQL e documentar precisão em segundos antes de mudar; preservar contagens sem duplicar fronteiras.
- Escuta manual: horário inexistente/ambíguo, offset explícito, instante salvo e retorno local. Definir escopo temporal dos campos days_active/streak/média e contrato AddDate dos períodos móveis antes de mudança funcional.

## Aceite

Escuta2024-05-01T00:05+05:45 pertence a01/05; último segundo31/05 entra no mês;04/30 23:55 e05/01 00:05 contam dois dias locais; janeiro sem ano em2024-12-31T23:30Z/Kathmandu resolve2025; semana53 em2021-01-01 resolve início2020-12-28. Todas as regressões passam, sem alterar timestamps persistidos nem esquema v3. Demais casos da matriz temporal têm esperado/observado. Mudanças visíveis constam no CHANGELOG.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../../docs-dev/comparisons/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Entrega — 05/11/2026

Correções, regressões, jornadas Go/React, revisão Advisor e contratos preservados
estão no [relatório da 018](../../docs-dev/comparisons/roadmap-05/ISSUE_018.md). Aceite explícito do usuário recebido em 05/11/2026.

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário.
<!-- SECTION:DESCRIPTION:END -->
