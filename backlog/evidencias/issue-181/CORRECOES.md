# ISSUE-181 — correções e verificação

Com a revisão desligada, o resolvedor por ocorrência agora associa uma
reentrega no mesmo segundo à escuta existente do mesmo perfil e ocorrência,
registrando `duplicate/same` com `review_deferred`. Candidatos de até dois
segundos sem igualdade exata resultam em `accepted` com `review_deferred`.
Com a revisão ligada, ambas as classes ficam pendentes. Uma reentrega exata
não cria escuta nem intenção de relay.

As ações em massa processam até 25 dos recibos elegíveis mais antigos do
perfil em um POST com CSRF. Aceitar exige um único candidato registrado
que ainda exista no perfil. Manter exige uma ocorrência existente e um
horário válido. O formulário identifica os recibos e suas versões por um
digest calculado no servidor. O writer recalcula a seleção antes de
aplicar o CAS de cada recibo. Repetições ou mudanças no lote devolvem
conflito sem avançar para o próximo lote. Cada revisão nova registra o
ID da conta em `evidence.decided_by`, tanto na ação individual quanto
na ação em massa. O contador reutiliza a consulta da fila e aparece
apenas no topo da central.

## Resumo de fontes e plano de execução

A migração 017 cria e preenche `ingest_source_summary`. A chave agrupa
perfil, instância da fonte, formato e cliente. A ISSUE-184 pode acrescentar
a dimensão do ID interno da chave sem acoplar a maneira de projetar a
entrega a credenciais. A admissão executa um UPSERT indexado em uma linha,
sem agrupar ou percorrer o histórico. Excluir o recibo mais recente
restaura o anterior por busca indexada ou remove o grupo vazio. O resumo
aponta para o recibo mais recente e usa o estado atual da outbox ao
exibir a última entrega. Todas as fontes ficam acessíveis em páginas
de até 100 grupos, independentemente da idade do último recibo.

A medição usou a fixture descartável do `TestIngestHealthQueryPlans`, com
300 mil recibos, 300 mil escutas, 5 mil faixas e dois perfis. O perfil
medido tinha 250 mil recibos e 250 pendências. O teste aplicou `ANALYZE`
em cada versão e registrou a mediana de cinco consultas aquecidas no
macOS ARM64 com o driver SQLite da aplicação.

| Consulta | v15 | v17 |
| --- | ---: | ---: |
| Fontes | 1065,50 ms | 0,102 ms |
| Decisões | 283,77 ms | 0,112 ms |
| Contagem, primeira página, candidatos e seleção dos lotes | 3,51 ms | 2,59 ms |

Na medição anterior, a janela de 10 mil recibos do v16 levou 43,67 ms para
fontes. No v17, o `EXPLAIN QUERY PLAN` mostrou busca ordenada no índice
`idx_ingest_source_summary_recent` por perfil e buscas exatas nas chaves
de recibo, integração e outbox. A consulta de fontes não usa ordenação
temporária. A migração de v15 a v17 levou 2,06 s nessa fixture. O backfill
e o índice de recuperação percorrem o histórico uma vez e têm custo
proporcional ao volume. A seleção de lotes grava no máximo 25 decisões,
mas pode percorrer mais pendências quando poucos recibos são elegíveis.

```sh
TUNETRAIL_TEST_HEALTH_SCALE=1 go test ./internal/db/sqlite \
  -run '^TestIngestHealthQueryPlans$' -count=1 -v
```

## Verificações

- Os testes de ocorrência cobrem reentrega exata com a flag ligada e
  desligada, candidatos da janela de dois segundos, isolamento entre
  perfis e ausência de uma segunda intenção de relay.
- Os testes de revisão cobrem autorização HTTP, CSRF, autor individual
  e em massa, limite, elegibilidade, repetição, corrida entre decisões
  opostas, rollback na segunda intenção e uma intenção por escuta aceita.
- O teste de fontes cobre backfill de fonte antiga, ordem de admissão e
  desempate, queda para o recibo anterior ao excluir o último, paginação
  e isolamento entre perfis.
- A referência ISSUE-075 foi regenerada. No Orca, os dois botões e o
  contador ficaram visíveis em 390, 1100 e 1500 px, com uma, duas e três
  colunas respectivamente e sem rolagem horizontal.
