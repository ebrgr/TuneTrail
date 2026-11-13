# Issue 007 — atividade compartilhada e interesse em SVG

**Situação atual: issue encerrada por aceite do usuário em 05/11/2026.**
O restante preserva o histórico de implementação e verificação.

Registro de 4 de novembro de 2026. [Issue](<../../tasks/issue-007 - 007-—-Migrar-atividade-e-interesse-das-páginas-de-mídia-para-HTML-SVG.md>) ·
[Plano](<../../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>).

**Estado: núcleo implementado e testado; issue aberta para aceite visual ao final
da 009.** A orientação mais recente do usuário substitui a revisão ao final da
008. Não foi feita inspeção de layout, abertura de navegador ou screenshot nesta
etapa. A integração das rotas públicas de mídia continua reservada à 009.

## Implementação

`internal/webui/activity.go` extrai a grade antes embutida na home e a reutiliza
em artista/álbum/faixa. Cada chamada recebe o contexto do perfil e somente o
filtro da entidade selecionada na rota. Home não ganha filtro de mídia.

A grade mantém 252 células, começo da semana do cookie de locale, datas civis no
fuso efetivo e dias futuros separados. Sem fuso explícito, o nome `Local` mantém
o fallback UTC usado na API de atividade; um fuso configurado conserva seu nome.
Os limiares de saturação são 100/20/10/5 para home/artista/álbum/faixa. A sequência
streak vem da mesma consulta filtrada; uma falha não pode virar streak zero.
O CSS existente continua escondendo dez semanas antigas em até 640 px; a tabela
acessível conserva os valores de todos os dias passados da grade.

`internal/webui/interest.go` transforma os intervalos reais em coordenadas
numéricas de SVG, usando interpolação cúbica natural, curva, área, gradiente e
variáveis do tema. Não há dependência nova de gráficos ou JS. Há tratamento de
série vazia, um ponto, valores iguais e contagens grandes, com verificação de
valores não negativos, overflow da soma e intervalos consistentes. O SVG usa
clipping da área de desenho e descrição do total. A paridade visual da curva
com Recharts ainda não foi conferida.

`templates/graphs.html` compartilha a apresentação de atividade e oferece tabelas
de datas/contagens em `<details>` nativos. Os valores podem ser acessados sem
JavaScript, cor ou hover; o SVG tem título/descrição e os controles têm foco CSS.
São verificações estruturais por testes; foco/teclado no navegador e aparência
em temas/breakpoints ficam para a revisão consolidada da 009.

Erros de interesse, atividade ou streak aparecem como indisponibilidade do bloco,
sem fabricar uma série vazia. Os demais dados da mídia permanecem disponíveis.
A home passa a usar o helper compartilhado; páginas de mídia continuam preparadas
no handler e exercitadas diretamente por testes, sem alteração do dispatcher.

## Contrato do interesse e correções da API

- `db.DefaultInterestBuckets = 16`; páginas Go usam esse valor fixo.
- `db.MaxInterestBuckets = 512`; store e endpoints validam 1–512 antes de consultar
  ou alocar. A API conserva a exigência de `buckets` explícito; ausência, formato
  inválido, zero, negativo ou acima do teto retorna 400. IDs não positivos também.
- Intervalos são contíguos, com início incluso/fim exclusivo; **o último inclui o
  instante final**. O eixo vai da primeira escuta válida da entidade/perfil até
  o horário capturado uma vez antes da consulta. Datas do JSON continuam em UTC;
  a tabela HTML converte para o fuso efetivo.
- Timestamps posteriores a esse horário são excluídos da consulta de interesse.
  Não se altera o registro nem a ingestão dessas escutas; elas passam a participar
  desse gráfico quando seu horário chegar.
- Cada timestamp consultado pertence a uma única divisão. Fronteiras repetidas,
  em uma história mais curta que a quantidade de divisões, representam intervalos
  vazios; no caso de um único instante, as escutas ficam na última divisão fechada.
  Sem escutas, os intervalos têm início/fim em agora e contagem zero.
- As fronteiras e a contagem usam a mesma partição. No caso inicial 100→110 com
  três divisões, o fim final agora é 110, e o instante 110 aparece na divisão cujo
  limite o inclui. Não há rótulo 109 contendo escuta 110.
- O cálculo evita multiplicar todo o intervalo temporal pelo número de divisões,
  preserva timestamps antigos e verifica cancelamento durante a distribuição.
- Os três handlers da API compartilham a validação; erro interno retorna 500
  genérico, sem expor SQL. Campos JSON existentes foram preservados.

A alteração está no changelog. O SQLite permanece no esquema consolidado **v3**,
sem migration. A consulta ainda carrega os timestamps elegíveis em memória; não
foi feita otimização SQL ou medição de desempenho nesta issue.

## Validação executada

- `make check ui.check`: vet, suíte Go, typecheck, oito testes do cliente,
  três testes de tema e verificação de sintaxe dos scripts passaram.
- `go test -race ./internal/webui ./internal/db/sqlite ./internal/webquery -timeout 90s`:
  passou, sem corrida detectada nos pacotes exercitados.
- Binário Go compilado. Nenhum código/dependência React mudou; validações HTTP
  usaram o build React local existente.
- `validate_webui.py`, ampliado nesta issue: além das regressões anteriores,
  verifica os endpoints reais de interesse de artista/álbum/faixa, 16 intervalos
  contíguos, totais iguais aos detalhes da mídia, isolamento de Luna e 400 para
  quantidades/IDs inválidos. Mara retornou 31/31/3 para artista/álbum/faixa de ID 1.
- `validate_multiuser.py`: PASS em banco novo e migrado de v2, com ingestão
  concorrente, permissões, importação/exportação, revogação, reinício e integridade.

Casos em `internal/db/sqlite/interest_test.go` cobrem associação de cada timestamp
à divisão anunciada, soma preservada, último segundo, duplicados, instante único,
intervalos mais curtos que a quantidade de divisões, vazio, valores inválidos,
cancelamento, datas antigas, futuro e isolamento em SQLite real descartável.

`internal/webui/graphs_test.go` cobre datas no fuso, todos os começos da semana,
DST de 23/25 horas, limiares, corte das semanas, erro de streak, opções filtradas,
coordenadas finitas, SVG bem formado, tabela sem JS, falhas parciais e equivalência
HTML/API por perfil. O teste de fallback `Local` usa uma zona com esse nome para
não depender do fuso configurado pela `.env` no processo de testes.

[validation-007.json](validation-007.json) registra os resultados sem credenciais.
Bancos usados são descartáveis em `.local/issue-007-*`; o banco da instalação
não foi usado. Os processos de validação encerraram ao concluir. O serviço
iniciado anteriormente pelo usuário não foi reiniciado.

## Próximos aceites

A 008 implementa a retrospectiva e seus cálculos. Ao final da 009, conferir o
layout e produzir os pares de screenshots **006 + 007 + 008 + 009**, incluindo
home após a extração, curvas, tabelas, foco/teclado, temas escuro/claro e celular.
Não considerar esses itens visuais aprovados pelos testes de dados desta issue.

Sem publicação de issues, commit ou push.
