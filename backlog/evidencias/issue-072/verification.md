# ISSUE-072 — verificação

- Implementação: quatro KPIs do período selecionado (minutos, reproduções, artistas e maior sequência de dias civis), comparação no servidor com intervalo anterior adjacente de igual duração em segundos. Os limites são congelados no fuso solicitado e normalizados para segundos. Todo o tempo não compara; ambos vazios mostram 0%; base anterior zero com atividade mostra “Nova atividade”.
- “Tocando agora” aparece uma vez no topo via listening.ReadNowPlaying, sem contagem fictícia de reproduções.
- Preservadas as decisões da ISSUE-154: heatmap fluido de até 35rem com 36/26 semanas, sem rolagem; grade com texto de largura integral, palavras inteiras e itens de pelo menos 22rem antes de criar outra coluna; últimas reproduções em toda a grade; navegação existente.
- Seletor sticky confirmado no navegador: após rolar 600px, topo em 8px e position sticky.
- make check: passou; inclui vet, testes Go, UI e 12/12 jornadas e2e.
- make ui.check: passou, incluindo comparação com reference/ desta tarefa; Makefile repontado.
- Testes SQLite cobrem cinco períodos, vazios atual/anterior, divisão por zero, variação positiva/negativa, limites adjacentes, isolamento de perfis e DST America/New_York. Teste HTML cobre posição e isolamento de tocando agora. Testes de tradução ajustados para o novo resumo de período.
- browser-matrix.json: 105 casos, três perfis × cinco períodos × sete larguras (320, 390, 640, 768, 1024, 1440, 1920), sem overflow horizontal nem erros de conteúdo; tema Yuu.
- Capturas Home/demo-cheio/Yuu: before-390.png, before-768.png, before-1440.png e after-390.png, after-768.png, after-1440.png. Altura de viewport 1000px, escala 1.
- Banco original preservado; utilizado somente clone temporário. Servidor encerrado ao concluir.
- Grafo atualizado por AST da worktree; dependências de loadHome/readHomeKPIs/homeValues verificadas.

## Limitação e pergunta para aceite

Em 320px as células visíveis têm aproximadamente 6,6px: permitem reconhecer densidade, mas a identificação de dias individuais é limitada (especialmente na faixa de 3–7px). O detalhamento textual “Valores de atividade” já existe. AC2 permanece pendente de avaliação de legibilidade; AC4 permanece desmarcado.

1. A leitura do heatmap atual é suficiente ou deseja planejar uma visão compacta opcional de 13 semanas, com células maiores e mantendo a visão atual disponível, sem rolagem? Essa alternativa não foi implementada.
