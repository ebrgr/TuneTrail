# ISSUE-072 — verificação final dos ajustes

- Quatro KPIs continuam por período; variações exatas ou arredondadas para zero não geram texto. Base anterior zero com atividade continua “Nova atividade”; todo o tempo não compara.
- Restaurados os seis valores históricos por listening.ReadStats(all_time): faixas, álbuns, dias ativos, média diária, faixas/artista e álbuns/artista. Rótulos en/pt-BR existentes reutilizados, com data-listen-count histórico e mensagem de erro própria. Falhas dos KPIs e dos totais usam estados independentes.
- Decisão mais recente do usuário: em >=64rem totais à esquerda, heatmap à direita. Em telas menores, mantido heatmap seguido dos totais. A quantidade de colunas dos totais depende da largura do próprio bloco.
- Heatmap preserva máximo 35rem, células fluidas e 36/26 semanas sem rolagem. Usuário aprovou o tamanho; AC2 verificado em 320px.
- Células usam data-tip com a mesma data/quantidade traduzida, aparência de tooltip.css, acima da célula, sombra, alinhamento para dentro nas extremidades e sem title nativo. CSS puro, sem JavaScript adicional. Grid segue aria-hidden; tabindex=-1 permite foco programático sem criar centenas de paradas de Tab em conteúdo oculto; dados acessíveis permanecem na tabela “Valores de atividade”.
- browser-adjustments.json: larguras 320/390/768/1440, sem overflow horizontal; todos os tooltips medidos ficam dentro do cartão. Hover confirmado nas semanas inicial, central e final, opacity=1. Células em 320px: ~6,6px; em 1440px: ~11,1px. Foco programático confirmado no DOM; a webview de automação em segundo plano reporta document.hasFocus=false, portanto a aparência de :focus foi conferida na regra CSS, sem afirmar observação de foco nativo nessa webview.
- Testes SQLite/HTML incluem cinco períodos, limites, isolamento, DST, base vazia, variação zero, totais históricos independentes do período selecionado, ausência de title, preservação da tabela e tocando agora.
- make check: verde, incluindo 12/12 jornadas e2e.
- make ui.check: verde; referência regenerada em reference/, Makefile já aponta para ela.
- Capturas finais Home/demo-cheio/Yuu: depois2-390.png, depois2-768.png e depois2-1440.png. Viewport 1000px de altura, escala 1.
- Banco original preservado; utilizado novo clone temporário. Servidor encerrado ao concluir.
- Grafo da worktree atualizado por AST. AC4 permanece pendente de aceite visual.
