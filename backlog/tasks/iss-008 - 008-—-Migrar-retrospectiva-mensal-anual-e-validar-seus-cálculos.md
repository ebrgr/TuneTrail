---
id: ISS-008
title: 008 — Migrar retrospectiva mensal/anual e validar seus cálculos
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
references: []
type: feature
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/008-retrospectiva-go-solved.md` — numeração local 008. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida — aceite confirmado pelo usuário em 05/11/2026**. Issue local, sem publicação.
Plano: [roadmap 3](<../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>). Terceira fatia, após
006/007; integração de produção e aceite visual na 009.

## Trabalho

- Criar normalização HTML específica para rewind usando o fuso efetivo.
  Sem query: último mês concluído; janeiro: ano anterior. Preservar ano isolado,
  ano+mês, inferência de ano e compatibilidade `month=0` anual.
- Validar ano/mês, overflow, combinações e períodos futuros; não transportar
  `NaN`, valores negativos ou strings `undefined` do cliente para o contrato Go.
- Usar URL como fonte da seleção mensal/anual, incluindo recarga e voltar/avançar.
  Seletores oferecem períodos concluídos; período atual explícito recebe rótulo
  de andamento. Preservar limites de calendário e primeira atividade do perfil.
- Chamar `summary.GenerateSummary` diretamente com ID do perfil e limites
  resolvidos uma vez. API e HTML compartilham o cálculo; não duplicar fórmulas.
- Corrigir o denominador de dias civis inclusivos para calendário, cobrindo
  meses/anos completos, bissexto e DST. Definir/testar também intervalos genéricos
  da API antes de mudar seus resultados; manter os filtros já existentes.
- Preservar top 5 por reproduções e seus tempos do período, plays, minutos,
  únicos e novos por perfil. “Mostrar tempo” só altera a apresentação dos itens.
- Implementar cartão/seletores responsivos, estado sem dados e erro explícito.
  Não ativar geração de PNG: `GenerateImage` está comentado e não é um fluxo
  existente da interface. Não adicionar exportação como parte desta migração.
- Registrar a alteração das médias no changelog na implementação.

## Aceite

Teste real de summary, sem corpo comentado, com SQLite v3 descartável: mesmo
catálogo em perfis diferentes, primeira escuta pessoal para contagens de novos,
empates, durações conhecidas e zero dados. Conferir API/HTML no mesmo intervalo.
Janeiro/2026 deve usar 31 dias, fevereiro/2026 28 e fevereiro/2024 29; cobrir DST,
virada de janeiro em fusos diferentes, anual e período atual. Validar controles
sem JavaScript e pares de capturas com minutos ligados/desligados.

## Evidência inicial

[analysis-checks.json](../evidencias/roadmap-03/analysis-checks.json) usa o helper
Go real de intervalo e a fórmula atual do summary: janeiro/2026 retorna denominador
30 e fevereiro/2026 retorna 27. Não é validação HTTP do summary nem resultado de
correção; é reprodução isolada que orienta as regressões.

## Ajuste de sequência solicitado pelo usuário

A orientação mais recente adia a conferência de layout e as screenshots de
**006 + 007 + 008 + 009** para o **final da issue 009**. A 008 continua responsável
pela implementação e testes de dados da retrospectiva; não antecipar o aceite
visual durante sua execução.

## Implementação e validação

O serviço e a página foram implementados, com testes numéricos e integração em
SQLite v3 descartável. O dispatcher de produção ainda não entrega a página Go;
a integração, a revisão de layout e as capturas ficam para a 009. A issue
permanece aberta até esse aceite. Detalhes em [ISSUE_008.md](../evidencias/roadmap-03/ISSUE_008.md).

## Encerramento

O usuário confirmou em 05/11/2026 que todas as issues desta etapa estão completas.
Este aceite encerra a issue. Os registros anteriores permanecem como histórico
das verificações executadas e das verificações manuais que não foram realizadas;
o encerramento não as apresenta como testes executados.
<!-- SECTION:DESCRIPTION:END -->
