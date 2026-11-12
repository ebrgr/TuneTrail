---
id: ISS-007
title: 007 — Migrar atividade e interesse das páginas de mídia para HTML/SVG
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
references:
  - docs-dev/issues/007-atividade-interesse-go-solved.md
type: feature
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/007-atividade-interesse-go-solved.md](../../docs-dev/issues/007-atividade-interesse-go-solved.md) — numeração local 007.

Status: **fechada e resolvida — aceite confirmado pelo usuário em 05/11/2026**. Issue local, sem publicação.
Plano: [roadmap 3](../../docs-dev/ROADMAP_03_MIDIA_RETROSPECTIVA.md). Após o núcleo da 006;
necessária antes da ativação na 009.

## Trabalho

- Extrair a composição da grade embutida na home para uso compartilhado com
  artista/álbum/faixa: 252 dias, fuso efetivo, começo da semana, dias futuros,
  streak, legenda e corte das dez semanas antigas no celular.
- Preservar limiares de intensidade: home 100, artista 20, álbum 10, faixa 5.
  Cada bloco recebe apenas o filtro da entidade resolvida na rota.
- Substituir Recharts por SVG responsivo gerado no Go: curva, preenchimento,
  gradiente e cores de tema. Não adicionar dependência de gráficos ao navegador.
- Usar 16 divisões por padrão, da primeira escuta da entidade no perfil até agora.
  Validar número positivo com teto razoável no endpoint existente; não alocar
  segundo valores negativos/enormes recebidos do cliente.
- Reproduzir/corrigir divergência entre fronteiras publicadas e distribuição das
  escutas em `GetInterest`; definir último limite e inclusão dos extremos.
- Tratar série vazia, instante único, todos os valores iguais e timestamps futuros;
  nenhum panic, divisão por zero, índice negativo ou coordenada não finita.
- Oferecer descrição/tabela acessível e valores por intervalo. Não depender só
  de cor/hover. Erro é indisponibilidade, não um gráfico de zeros.
- Registrar custo atual de leitura dos timestamps em memória; medir antes de
  otimizar SQL. Preservar cancelamento e isolamento, sem cache global entre perfis.

## Aceite

Regressões para home e mídia, limiares e fusos; soma das divisões igual à quantidade
prevista, limites coerentes e casos degenerados. Testes de parâmetros inválidos
sem alocações perigosas. Comparação visual da curva e da grade em desktop/celular,
tema escuro/claro e teclado/acessibilidade.

## Evidência inicial

[analysis-checks.json](../../docs-dev/comparisons/roadmap-03/analysis-checks.json) reproduz as
fórmulas: com início 100, fim 110 e três divisões, o instante 110 é contado na
última divisão, mas seu fim publicado é 109. É análise isolada, sem banco/HTTP;
correção funcional e reprodução com store real ainda pendentes.

## Marco de implementação — 04/11/2026

Grade compartilhada de 252 dias e gráfico de interesse SVG implementados, com
tabelas acessíveis sem JavaScript. Corrigidas partição/contagem dos intervalos,
tratamento de escutas futuras e validação de 1–512 divisões na API/store; a página
usa 16. Testes de dados, API/HTML, SQLite, concorrência e regressões reais passaram.

Por orientação mais recente do usuário, **layout e screenshots ficam para o final
da issue 009**, substituindo o agendamento anterior na 008. A issue permanece
aberta para esse aceite. Não houve ativação pública das páginas de mídia.

[Implementação, contratos e evidências](../../docs-dev/comparisons/roadmap-03/ISSUE_007.md).

## Encerramento

O usuário confirmou em 05/11/2026 que todas as issues desta etapa estão completas.
Este aceite encerra a issue. Os registros anteriores permanecem como histórico
das verificações executadas e das verificações manuais que não foram realizadas;
o encerramento não as apresenta como testes executados.
<!-- SECTION:DESCRIPTION:END -->
