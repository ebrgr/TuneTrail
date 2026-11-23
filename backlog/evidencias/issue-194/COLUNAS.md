# ISSUE-194 — Inventário para a ISSUE-195

Levantamento de referências Go, SQL e scripts após a limpeza. O inventário
considera acesso a dados em execução e distingue usos no DDL ou na validação
de layout. A ISSUE-195 decidirá quais campos preservar durante a consolidação.

## Campos sem leitura e escrita de dados pelo Go de produção

| Campo ou tabela | Motivo |
| --- | --- |
| `ingest_events.uncertainty_ms`, `client_name`, `client_version` | Sem acesso direto a essas colunas no Go. Os metadados do cliente usados hoje vêm de `original_snapshot`. |
| `ingest_events.normalized_snapshot`, `normalizer_version` | Sem acesso direto no Go. O par ainda faz parte das restrições históricas e do layout validado. |
| `listens.review_distinct` | O escritor v9 de decisões manuais preenchia a exceção para a chave única do legado. A 021 removeu o índice e a ISSUE-194 retirou o último acesso Go. |
| `profile_ingest_flags` | A 021 já elimina a tabela. Restam a validação de layout e os testes de migração e das guardas históricas. A configuração corrente usa `profile_ingest_review`. |

`reprocessed_at` e `reprocessed_by` continuam sendo gravados em
`requeueParkedIngest` como trilha de reprocessamento, sem leitura direta no Go.

## Campos com uso no schema 21

| Campo ou tabela | Usos |
| --- | --- |
| `ingest_events.mode` | Admissão ativa, escopo de fila e revisão, diagnósticos e exclusão de dados históricos de sombra. |
| `ingest_events.resolver_variant` | Inserção por ocorrência, guarda do worker e escopo da resolução e exclusão. Também identifica a variante de recibos preservados nas migrações. |
| `ingest_events.delivery_policy`, `relay_delivery_gate` | A política é capturada na admissão e lida no worker, revisão e relay. A guarda SQL da política segue em vigor. O novo outbox continua restrito a recibos com política `durable`. |
| `ingest_events.parked_at`, `park_reason` | O worker estaciona tentativas esgotadas. Saúde, descarte e reprocessamento consomem esses dados. |
| `ingest_events.retry_attempt_base`, `claim_attempt_base` | Controlam independentemente o orçamento de falhas e o de claims, inclusive na recuperação de leases e no reprocessamento. |
| `ingest_events.relay_admission_generation` | O gatilho grava a geração aprovada na admissão. A inserção de outbox exige a mesma geração ainda aprovada. |
| `ingest_event_details`, `outbox_details` | As rotinas de retenção ainda removem dados históricos dessas tabelas. |
