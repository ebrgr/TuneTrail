# ISSUE-181 — verificação

## AC1 — comportamento encontrado na base v0.56

- O resolvedor legado já lia `profile_ingest_flags.review`. Quando havia candidato e a flag estava ligada, `finalizeActiveReceipt` gravava `pending_duplicate`, decisão `review` e `candidate_event_id`. Sem a flag, usava a deduplicação legada no mesmo segundo. [Resolvedor legado](../../../internal/db/sqlite/ingest_resolve.go)
- O resolvedor por ocorrência já gravava `pending_duplicate`, decisão `review` e a lista `candidates` nas colisões com escutas. O ramo consultava os candidatos independentemente da flag. A partir desta issue, a flag controla esse ramo. [Resolvedor por ocorrência](../../../internal/db/sqlite/ingest_occurrence.go)
- A migração 009 já criava `review` com padrão `0`. A API root recebia o campo, mas `SetIngestActivation` rejeitava `review=true` e a central não oferecia um controle. Na base, somente alterações diretas no SQLite podiam ligar a flag. [Ativação](../../../internal/db/sqlite/ingest_activation.go)
- Conflitos de lote também podem gerar `pending_duplicate` sem candidato nem ocorrência resolvida. Esses recibos ficam visíveis com a razão e sem ações indisponíveis. Ligar ou desligar a revisão de possíveis duplicatas mantém o tratamento dos conflitos de lote. [Admissão](../../../internal/db/sqlite/ingest_receipts.go)

## AC2–5 — fila, decisões e interface

A fila apresenta 25 recibos por página no perfil selecionado. A projeção inclui fonte, horários, motivo, regra, faixa quando conhecida e candidatos do próprio perfil. O POST verifica sessão e CSRF. O escritor SQLite verifica o perfil, o estado pendente, o candidato registrado e as duas versões sob `BEGIN IMMEDIATE`. A decisão, a revisão humana, a escuta separada e a intenção elegível de relay são confirmadas juntas.

- `TestIngestReviewDecisions` cobre flag desligada, isolamento de recibos e candidatos, candidato fora da evidência, repetição, rollback forçado na gravação da outbox, tentativa posterior bem-sucedida e uma intenção elegível.
- `TestIngestReviewRace` disputa decisões opostas usando duas conexões. Uma decisão confirma e a outra recebe conflito. Dez execuções com o detector de corridas passaram.
- `TestOccurrenceReviewSetting` cobre o padrão desligado, múltiplos candidatos e a criação de uma escuta separada no resolvedor por ocorrência.
- `TestIngestHealthReviewActions` cobre isolamento HTTP, projeção sem segredos, GET sem mutação, CSRF ausente e de outra sessão, flag por perfil, decisões de dono e root, conflito em reenvio e a tradução em pt-BR.
- Referência ISSUE-075 regenerada por `make ui.reference`. A verificação comparou 176 arquivos. No Orca, a referência carregou em 390 e 650 px com uma coluna, 1100 px com duas e 1500 px com três, sem rolagem horizontal. Os detalhes usam uma coluna abaixo de 40 rem e duas acima. O resumo abre pelo ponteiro e pelo teclado, tem foco visível e muda de fundo no hover. `health.css` tem 68 linhas.
- `TUNETRAIL_WEB_DIR=$PWD/assets make check` passou, incluindo testes Go, referência, estilo e E2E. Grafo AST atualizado.

## AC6 — consultas com 300 mil recibos

Base SQLite descartável, com 300 mil recibos e 300 mil escutas sintéticas, dois perfis, 5 mil faixas e três fontes. O perfil medido tinha 250 mil recibos e 250 pendências. O teste executou `ANALYZE` em cada versão, descartou a primeira medição e usou a mediana das cinco seguintes. Ambiente macOS ARM64, Go 1.26.3, driver SQLite do aplicativo.

| Consulta | v15 | v16 | Plano em v16 |
| --- | ---: | ---: | --- |
| `healthSources` | 1032,28 ms | 43,67 ms | Busca em `idx_ingest_health_received` por perfil e ordem de recebimento. A janela e as ordenações temporárias trabalham sobre os 10 mil recibos mais recentes. |
| `healthDecisions` | 283,42 ms | 0,113 ms | Busca em `idx_ingest_health_decisions` por perfil. A expressão indexada entrega os 25 resultados já ordenados. |
| Contagem, primeira página e candidatos | 2,88 ms | 2,38 ms | Busca em `idx_ingest_health_review` por perfil, sem ordenação global. Candidatos consultam cada recibo e cada escuta pelas chaves `(profile_id, receipt_id)` e `(profile_id, event_id)`. |

O plano v15 de fontes percorria todos os recibos do perfil e ordenava para agrupar. O de decisões varria a tabela e usava uma B-tree temporária para ordenar. A consulta nova de pendências, quando executada no v15, usava o índice de fila por estado e uma B-tree temporária para ordenar a página. O índice parcial do v16 cobre a contagem de pendências e a ordenação da página. O custo da contagem cresce com o número de pendências abertas.

Para repetir com as consultas da aplicação e imprimir o `EXPLAIN QUERY PLAN`:

```sh
TUNETRAIL_TEST_HEALTH_SCALE=1 go test ./internal/db/sqlite \
  -run '^TestIngestHealthQueryPlans$' -count=1 -v
```

A migração 016 é aditiva. Ela mantém o índice de deduplicação automática para escutas legadas e permite separar explicitamente duas escutas legadas no mesmo segundo. O downgrade é bloqueado pelo schema. O resumo de fontes deixa de mostrar clientes cuja última submissão ficou fora dos 10 mil recibos recentes.
