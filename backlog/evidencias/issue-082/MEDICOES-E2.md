# ISSUE-082 — Medições da E2

Ensaio local em macOS 15.7.3 arm64, 10 CPUs lógicas, Go 1.26.3. Banco v11 descartável com 300 mil recibos sintéticos em sombra, snapshot pequeno, um perfil e fila de entrega vazia. O WAL foi truncado antes da medição. Os tempos abaixo são de uma execução e separam o backup da aplicação do schema. DNS, validação integral do boot e recuperação não fazem parte da cronometragem da 012.

| Operação | Tempo | Tamanho observado |
| --- | ---: | ---: |
| Banco v11 após checkpoint | — | 99.348.480 B |
| Backup consistente com `VACUUM INTO` | 247,8 ms | 95.576.064 B |
| Transação da 012 | 1.427,9 ms | WAL de 74.192 B |
| `ADD COLUMN delivery_policy` com CHECK, isolado em cópia v11 | 1.420,8 ms | WAL de 16.512 B |

O custo observado é compatível com a varredura para validar o CHECK. A mudança não regrava os 300 mil recibos. O teste separado `TestRelayBootstrapWAL` conserva um recibo recém-confirmado no WAL no backup v11, valida o upgrade e confirma que a segunda abertura não cria outro backup. A injeção antes da DDL, após a DDL, após a semeadura e antes do commit confirma rollback de schema, dados e versão.

## Consultas quentes

- Candidatos de destino. `SCAN d` e ordenação temporária de destinos pelo último claim. A subconsulta de vencimento busca pelo índice parcial `idx_outbox_relay_due` (perfil, destino, próximo instante). A subconsulta de capacidade busca por `idx_outbox_relay_capacity` (perfil, destino). A checagem de configuração usa a chave primária de `profile_relays`.
- Seleção de itens do lote. Busca em `idx_outbox_relay_due` por perfil, destino e instante. Sem ordenação temporária da fila no plano observado. A tabela de destinos mantém o contador; não há COUNT da fila no caminho de escrita.
- O teste de capacidade é separado. Preenche 100 mil itens efetivos, confirma que a intenção excedente fica em `failed/queue_capacity`, conclui um item e verifica a readmissão e a igualdade entre o contador e a contagem real.
- Ainda falta medir em E4 a latência com destinos concorrentes, a expiração de uma fila antiga e o custo do handshake por POST. Os tempos daqui não representam carga sustentada nem um banco com snapshots maiores.

```sh
TUNETRAIL_RELAY_MEASURE=1 go test ./internal/db/sqlite -run '^TestRelay' -count=1 -v
```
