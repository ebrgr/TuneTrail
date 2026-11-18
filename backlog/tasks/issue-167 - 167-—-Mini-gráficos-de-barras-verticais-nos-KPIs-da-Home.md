---
id: ISSUE-167
title: 167 — Mini-gráficos de barras verticais nos KPIs da Home
status: Aberta
assignee: []
created_date: '2026-11-18T10:33:00Z'
updated_date: '2026-11-18T10:40:00Z'
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
- [ ] #3 Consulta agrupada testada (as contagens por balde batem com os KPIs, fuso e horário de verão) e medida em histórico grande (24 mil escutas ou mais)
- [ ] #4 Card Maior sequência com barra segmentada proporcional (dias da maior sequência sobre os dias do período, ex.: 3 de 7), em SVG, sem JavaScript, válida em dia, semana, mês, ano e tudo; se a faixa de dias for adotada, GetLongestListenStreak passa a devolver também o início e o fim da sequência
- [ ] #5 A comparação dos KPIs mostra o ícone de chama junto de 'N% Mais' e um ícone de gelo junto de 'N% Menos' (ícone escolhido pelo usuário entre as opções); 'Igual' e a ausência de comparação sem ícone; ícones decorativos (aria-hidden) com o texto acessível; catálogo local de ícones e manifest atualizados
- [ ] #6 Layout responsivo em 320 a 1920px sem rolagem; capturas e aceite visual do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-18T10:40:00Z

---

Decisões do usuário (2026-11-18): (1) Maior sequência: gostou da opção 2, a BARRA SEGMENTADA PROPORCIONAL (parte preenchida = dias da maior sequência sobre os dias do período, ex.: '3 de 7 dias'), dependendo de como ficar a implementação (aprovação final no aceite visual); funciona em qualquer período, em especial ano e tudo; a faixa de dias com a sequência destacada fica como alternativa para semana e mês, se agradar. (2) Ícones na comparação dos KPIs: uma CHAMA junto do 'N% Mais' e um CUBO DE GELO junto do 'N% Menos'. Observação técnica: o conjunto Lucide do projeto não tem cubo de gelo; opções: Snowflake (floco), Cuboid ou Box (cubo genérico) ou um SVG próprio; Flame existe. A escolha do ícone do 'Menos' fica para a implementação, com opções mostradas ao usuário (como foi feito com a chave na ISSUE-154). Sem ícone no 'Igual'.

---
<!-- COMMENTS:END -->
