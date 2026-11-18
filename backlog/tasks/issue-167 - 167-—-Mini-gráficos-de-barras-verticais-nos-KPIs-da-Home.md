---
id: ISSUE-167
title: 167 — Mini-gráficos de barras verticais nos KPIs da Home
status: Aberta
assignee: []
created_date: '2026-11-18T10:33:00Z'
updated_date: '2026-11-18T10:33:00Z'
labels:
  - ui
milestone: m-10
dependencies: []
priority: medium
type: feature
ordinal: 180000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: conversa com o usuário em 2026-11-18, após a ISSUE-072. Os KPIs da Home (Minutos, Reproduções, Artistas, Maior sequência) mostram só o número e a comparação com o período anterior. Ideia: um mini-gráfico de BARRAS VERTICAIS dentro do card, para ver a evolução no período. Barras (e não linha) para diferenciar do gráfico de linha 'Interesse ao longo do tempo' das páginas de artista, álbum e faixa. Já descartada: imagem mista com 4 capas 64x64 (redundante com os destaques da Home e exigiria compor e cachear imagens no servidor). Como os KPIs são calculados hoje: consultas ao vivo sobre a tabela listens (CountTimeListened soma tracks.duration; CountListens; CountArtists distintos; GetLongestListenStreak), sem tabela de histórico; a comparação repete a consulta na janela anterior de mesma duração. O histórico necessário já existe em listens.listened_at, indexada por perfil. Proposta de desenho: SVG gerado no SERVIDOR, sem JavaScript, na altura do número, cor do tema, aria-hidden com os valores acessíveis (tabela ou details como nos outros gráficos), e o tooltip próprio (data-tip, o mesmo do heatmap da ISSUE-072) em cada barra. Baldes por período: dia = 24 horas; semana = 7 dias; mês = dias do mês; ano = 12 meses; tudo = por ano ou até N baldes (a decidir). Cards: Reproduções (COUNT por balde) e Minutos (SUM da duração por balde) numa única consulta agrupada; Artistas (COUNT DISTINCT por balde, a mais cara, medir antes); Maior sequência SEM gráfico (é um número único). Considerar uma barra do período anterior esmaecida só se não poluir. Perguntas ao usuário na implementação: Artistas com barras ou não, baldes do período Tudo, posição (ao lado do número ou no fundo do card) e se mostra o período anterior. Coordenar com a ISSUE-092 (semântica das estatísticas e desempenho em históricos grandes) e a ISSUE-093 (comparação de períodos e setas).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Reproduções e Minutos mostram barras verticais por balde do período (dia, semana, mês, ano e tudo), geradas no servidor em SVG, sem JavaScript
- [ ] #2 Cada barra tem o tooltip próprio com o valor; os valores ficam acessíveis fora do SVG; períodos vazios e perfis sem escutas tratados
- [ ] #3 Consulta agrupada testada (contagens por balde batem com os KPIs, fuso e horário de verão) e medida em histórico grande (24 mil escutas ou mais)
- [ ] #4 Layout responsivo em 320 a 1920px sem rolagem; o card Maior sequência não tem gráfico; capturas e aceite visual do usuário
<!-- AC:END -->
