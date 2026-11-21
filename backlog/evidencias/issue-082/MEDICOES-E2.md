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

## Revisão do Advisor

Medições na mesma máquina arm64, com SSD local. Cada medição de configuração usa uma fixture independente com 100 mil intenções abertas, recibos aceitos e escutas correspondentes. Os tempos são de parede e incluem a transação e seu commit. O preparo das fixtures ficou fora do cronômetro.

| Operação | Trigger anterior | 012 emendada |
| --- | ---: | ---: |
| Trocar URL pelo store | 532,0 ms | 174,3 µs |
| Remover relay pelo store | 514,4 ms | 125,0 µs |
| Excluir perfil e seus dados | 2,130 s | 2,132 s |
| Expirar lote sem itens vencidos, fila de 100 mil | — | 14,5 ms |

Troca e remoção ultrapassaram o limiar de 500 ms, então a decisão d foi aplicada. O trigger agora avança a geração e zera o contador. A contagem de trabalho elegível considera somente linhas abertas da geração atual. O teste confirma o contador zero logo após a mudança, mesmo com as 100 mil linhas antigas ainda materializadas como abertas. O store materializa a falha em grupos limitados. Excluir o perfil ainda percorre o histórico e suas cascatas. O tempo de expiração corresponde ao pior caminho de procurar sem encontrar vencidos nessa fixture, com lease e data ainda elegíveis no mesmo índice da fila.

### EXPLAIN antes e depois

- Renovação e liberação anteriores: `SEARCH outbox USING INDEX idx_outbox_event (profile_id=?)`. A busca examina a faixa do perfil e filtra a posse depois.
- Renovação e liberação emendadas: `SEARCH outbox USING INDEX sqlite_autoindex_outbox_1 (id=?)`, alimentado por `json_each` com os IDs do lote. Perfil, destino, geração, posse e estado são filtros adicionais.
- Expiração ficou em duas instruções. A busca de itens abertos usa `idx_outbox_relay_due` por perfil, destino e geração. A busca de overflow usa `idx_outbox_relay_capacity` e também limita `created_at`. Cada instrução modifica no máximo 100 linhas.
- A seleção de POST continua em `idx_outbox_relay_due`, com geração e prazo na busca.

### 300 mil marcadores pré-082

Um segundo banco v11 descartável contém 300 mil escutas, recibos aceitos e marcadores `discarded/pre_082_best_effort` correspondentes. Após truncar o WAL, medimos o backup e toda a 012. Em seguida medimos, de forma independente na cópia descartável v11, o CHECK de `outbox.format_version` e a criação dos dois índices parciais de fila após instalar as colunas necessárias. Nenhum marcador foi promovido.

| Operação | Tempo | Tamanho observado |
| --- | ---: | ---: |
| Backup consistente | 755,7 ms | 240.709.632 B |
| Transação da 012 completa | 2.302,8 ms | WAL de 90.672 B |
| `ADD COLUMN format_version` com CHECK, isolado | 354,0 ms | WAL de 8.272 B |
| Dois índices parciais de fila, isolados | 37,3 ms | WAL de 32.992 B |

Na repetição com outbox vazia, a 012 emendada levou 1,391 s e deixou 90.672 B de WAL. Os índices parciais não incluem os marcadores encerrados, embora sua criação leia a tabela. Esta foi uma execução por cenário em SSD local, sem carga concorrente ou DNS nessa fixture. NAS e cartão SD podem ter custo perto de 10× maior. A ordem de grandeza serve para planejar a janela, sem extrapolar como garantia.

```sh
TUNETRAIL_RELAY_MEASURE=1 go test ./internal/db/sqlite -run 'TestRelay(MarkerMeasurements|MutationMeasurements|MigrationMeasurements|LeaseQueryPlans)' -v -count=1
```
