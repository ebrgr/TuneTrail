# Verificação da ISSUE-184

## Inbox e migração

- A autenticação pelo ListenBrainz registra somente o ID interno da chave no recibo. A migração mantém os IDs existentes e faz a alocação de IDs monotônica para que uma chave nova jamais assuma os recibos de uma chave excluída. A guarda do downgrade falha dentro da transação. O bootstrap usa o backup anterior à migração antes de modificar uma base existente.
- O teste de migração parte do schema v18 e verifica o backfill de um recibo sem chave, importação de arquivo, chave conhecida, chave excluída e substituição sem reuso de ID. Exercita as três janelas, o limite inclusivo de 24 h, a atualização de estado e a remoção de um recibo dos contadores. Uma projeção de metadados legada com objeto em vez de título mantém o título vazio, sem renderizar o JSON.
- Os testes HTTP enviam recibos de duas chaves do mesmo perfil e de uma chave de outro perfil. Confirmam atribuição, filtro, remoção de chave e ausência das credenciais e dos dados de outro perfil. Os testes existentes exercitam a autorização de dono e root e continuam cobrindo os POSTs de revisão e reprocessamento.
- Os totais descrevem os recibos persistidos e o estado atual de cada um. Reenvios que reaproveitam o mesmo recibo continuam aparecendo uma vez. Solicitações rejeitadas antes da admissão ficam fora dos totais. Recibos no modo sombra podem permanecer pendentes, pois o desfecho do escritor legado não é registrado neles.

## Medição em SQLite

Teste local em macOS ARM64 com o driver da aplicação, WAL e base em disco descartável. A base tem 300 mil recibos e 300 mil escutas sintéticas. O perfil medido tem 250 mil recibos, 12 combinações possíveis de cliente e três fontes no conjunto de dados. As janelas usam o horário de recebimento. Valores abaixo são medianas de cinco execuções com cache aquecido, incluindo projeções de exemplo no método do repositório. A medição cobre a consulta e a leitura dos resultados, sem a renderização HTTP.

| Janela | Índice sobre recibos | Contadores diários e bordas |
| --- | ---: | ---: |
| 24 h | 5,99 ms | 8,16 ms |
| 7 dias | 46,96 ms | 8,26 ms |
| 30 dias | 98,55 ms | 6,33 ms |

O índice sobre recibos percorreu todos os registros do perfil para a janela de 30 dias. A versão adotada mantém contadores diários por perfil, chave, cliente, estado e classificação de metadados. Os dias inteiros usam os contadores. Os intervalos parciais consultam recibos para preservar o horário exato dos limites. Cada exemplo consulta o índice parcial de metadados com perfil, chave, cliente, classificação e intervalo de recebimento.

Resumo do `EXPLAIN QUERY PLAN` final, igual nas três janelas:

```text
CO-ROUTINE receipts
COMPOUND QUERY
SEARCH ingest_inbox_daily USING PRIMARY KEY (profile_id=? AND day>? AND day<?)
UNION ALL
SEARCH ingest_events USING INDEX idx_ingest_inbox_edges (profile_id=? AND received_at>? AND received_at<?)
UNION ALL
SEARCH ingest_events USING INDEX idx_ingest_inbox_edges (profile_id=? AND received_at>? AND received_at<?)
SCAN receipts
USE TEMP B-TREE FOR GROUP BY
```

A consulta existente de última entrega por fonte reutiliza `ingest_source_summary` e levou 0,108 ms. A de decisões levou 0,115 ms. A contagem e a primeira página de revisão com candidatos levaram 2,72 ms.

A passagem de v17 a v19 levou 7,00 s após a preparação do volume. O backfill constrói os índices e os contadores no startup sem reescrever os recibos existentes em massa. SQLite mantém o lock de escrita durante a migração. Esse tempo não inclui o backup e pode crescer com o volume ou com armazenamento mais lento. Concentração de muitos recibos nos dias parciais e alta cardinalidade de clientes também elevam o custo de leitura.

```sh
TUNETRAIL_TEST_HEALTH_SCALE=1 go test ./internal/db/sqlite \
  -run '^TestIngestHealthQueryPlans$' -count=1 -v
```

## Interface

Referência ISSUE-075 regenerada com `make ui.reference`. A fixture tem duas chaves, uma amostra de cada faixa de metadados, revisão pendente, falha permanente e falha de relay. O Orca verificou a referência com os detalhes abertos.

| Viewport | Colunas | Largura rolável da página | Largura visível/rolável da tabela de relay |
| --- | --- | ---: | ---: |
| 390 px | 1 × 358 px, Inbox primeiro | 390 px | 308 / 308 px |
| 1100 px | 2 × 469 px, Inbox à esquerda | 1100 px | 419 / 419 px |
| 1500 px | 2 × 538 px, Inbox à esquerda | 1500 px | 488 / 488 px |

O fundo do resumo muda no hover com o token já usado pela central. O CSS tem 67 linhas e usa os breakpoints de 40 e 64 rem. A comparação da referência verificou os 176 arquivos.

## Comandos de verificação

```sh
TUNETRAIL_WEB_DIR=$PWD/assets make check
TUNETRAIL_WEB_DIR=$PWD/assets go test -race ./internal/db/sqlite ./engine \
  -run 'Test(IngestInbox|IngestHealthInbox|IngestHealthReprocess|IngestHealthReviewActions)' -count=1
```

Teste com detector de corridas aprovado nos dois pacotes. Grafo AST atualizado para esta worktree.
