# ISSUE-171 — verificação do modo sombra após a revisão

O [parecer do Reviewer](PARECER-REVIEWER.md) avaliou o commit inicial `95c0875`. A correção trata o item obrigatório de exclusão e as ressalvas 1, 2, 4, 5 e 8. As ressalvas 3, 6 e 7 ficaram registradas na ISSUE-081 como entradas obrigatórias da ativação.

## Comportamento verificado

- A exclusão individual remove recibos no mesmo perfil e segundo UTC quando artista e título correspondem aos nomes ou aliases do catálogo. As observações e os detalhes caem em cascata. `ApplyProfileDelete`, quando apaga escutas, limpa todos os recibos e apenas as integrações de sombra do perfil afetado. A configuração de fontes externas é preservada. A exclusão da conta usa a cascata do perfil. A limpeza global remove todos os recibos e as integrações de sombra `listenbrainz:` e `file:`. Cada operação usa a mesma transação das exclusões legadas.
- `BEGIN IMMEDIATE` reserva a escrita antes da consulta às chaves e aos candidatos. Reenvio equivalente reutiliza a chave e devolve `same` ao chamador, preservando decisão, evidência e versão gravadas. Uma variante incompatível preserva o snapshot original, grava observação e sinaliza Revisar. Decisões históricas já gravadas pela versão anterior não são reconstruídas.
- `shadow-v2` normaliza caixa Unicode, NFC e espaços. MBIDs de gravação declarados e mapeados participam da comparação como evidência de candidatura. A janela de ±2 s seleciona uma ambiguidade e pode produzir Revisar. Igual continua exigindo chave confiável e conteúdo equivalente.
- Cada importador acumula até 200 recibos por transação e descarrega a última página antes de arquivar o arquivo. O sombreamento mantém memória limitada ao lote, sem alterar as estruturas já usadas pelos parsers legados. Os ordinais incluem itens filtrados. O cursor lê checkpoints persistidos sem staging. O reenvio preenche páginas ausentes e mantém os IDs das existentes. Erros da gravação em sombra seguem em fail-open. Se o contexto for cancelado, a cauda ainda em memória pode ser recuperada reenviando o arquivo.
- A API responde com 200 após o commit do recibo e o caminho legado. Falha de disco, armazenamento ou commit do recibo dá 503. Falha estritamente interna de sombra antes do commit continua em fail-open. `playing_now` segue efêmero e o relay mantém o fluxo existente. Para itens sem `listened_at`, o recibo guarda a hora de chegada como `estimated` e `listens` calcula por item. Se os dois instantes caírem em segundos diferentes, a correspondência de uma exclusão individual fica limitada ao segundo de `listens`; a limpeza do perfil remove esses recibos também.

## Critérios de aceite

| AC | Evidência |
| --- | --- |
| 1 | `TestShadowReceiptACK` confirma dois recibos, uma decisão de Revisar e uma escuta legada. `TestShadowDeletePaths` cobre exclusão, rollback por falha, cascatas, preservação de outros perfis e reimportação após apagar. `TestShadowFileBatching` e `TestShadowFileResume` cobrem página parcial, reenvio, cursor e contadores legados. |
| 2 | `TestShadowReceiptACK` injeta falha no commit por FK adiada e comprova 503, reenvio, fail-open e `playing_now`. `TestShadowDiskFull` provoca SQLITE_FULL com `max_page_count`, comprova rollback integral e repete o lote com sucesso. |
| 3 | `TestShadowWALRetry` disputa os índices de ID externo, chave de cliente mais ordinal e arquivo mais ordinal por dois pools WAL, com uma linha por chave e preservação da decisão original. `TestShadowRepetitions` cobre fuso, horário de verão e mesmo segundo. `TestShadowUnicodeAndMappedRecording` cobre Unicode decomposto, caixa, espaços e MBID mapeado. |
| 4 | Replay histórico em cópia privada e replay sintético reproduzíveis, descritos abaixo. O histórico mede a classificação de linhas que sobreviveram ao caminho legado e permite verificar a invariância dos dados. O cenário sintético comprova a resposta da regra, sem estimar a incidência dessas situações no tráfego. |
| 5 | `TestShadowV4AcceptsSame` aplica somente até a migração 004. A regra compara com um candidato de ID confiável e a fixture insere uma nova linha com o resultado `same`, evidência apontando ao candidato e `event_id IS NULL`. A chave única continua no candidato, cuja decisão inicial é preservada. |

## Replay reproduzível

```sh
python3 scripts/validate_ingest_shadow.py
python3 scripts/validate_ingest_shadow.py --source /caminho/para/banco-quiescente.db
```

O script copia DB, WAL, SHM e journal, quando presentes, para uma pasta temporária antes de usar SQLite. Compara em memória os hashes físicos da origem antes e depois. O replay usa o `ReceiveBatch` de produção com `shadow-v2`. No caso histórico, a cópia passou da versão 5 para a 6 e as contagens, o digest de `listens`, `integrity_check` e `foreign_key_check` foram conferidos. A origem teve hashes idênticos antes e depois.

| Replay | Diferente | Igual | Revisar |
| --- | ---: | ---: | ---: |
| Histórico real, 80 escutas em um perfil com histórico | 80 | 0 | 0 |
| Fontes cruzadas, diferença de 0 s | 1 | 0 | 1 |
| Fontes cruzadas, diferença de 1 s | 1 | 0 | 1 |
| Fontes cruzadas, diferença de 2 s | 1 | 0 | 1 |
| Duas entradas da mesma faixa no mesmo segundo | 1 | 0 | 1 |
| Controle de fontes cruzadas, diferença de 3 s | 2 | 0 | 0 |

As duplas sintéticas têm grafias diferentes de álbum, caixa, espaços e acentos. Ao todo, foram **4 de 10** itens encaminhados para Revisar nessas fixtures. O **0 de 80** histórico vem de linhas que já passaram pela chave legada por perfil, faixa e segundo, sem proveniência de múltiplas fontes no replay. A amostra não recupera reenvios ou repetições já descartados e não estima a taxa em produção.

## Medição antes da ativação

Medir algumas semanas de modo sombra em produção antes de ligar a ISSUE-081. Fixar início e fim em Unix UTC, manter a consulta junto do relatório e repetir por versão da regra. A unidade é o recibo persistido; reenvios equivalentes reutilizam o recibo. Falhas de sombra que não geram recibos ficam fora do denominador.

```sql
SELECT rule_version, source_type, duplicate_decision, reason_codes, COUNT(*) AS receipts
FROM ingest_events
WHERE received_at >= :start_utc AND received_at < :end_utc
  AND rule_version = 'shadow-v2' AND source_type IN ('listenbrainz', 'file')
GROUP BY rule_version, source_type, duplicate_decision, reason_codes
ORDER BY source_type, duplicate_decision, reason_codes;
```

## Verificações locais

- `TUNETRAIL_WEB_DIR="$PWD/assets" make check` passou, incluindo Go vet/test, UI e 13 de 13 jornadas e2e.
- `CGO_ENABLED=1 go test -race -count=1` passou em `engine`, `engine/handlers`, `internal/db/sqlite`, `internal/importer` e `internal/dedup`. `internal/ingest` e `scripts/ingest_shadow_replay` foram compilados com o detector. O teste de concorrência usou duas conexões WAL independentes.
- Graphify AST foi atualizado a partir da worktree. `git diff --check` passou.

## Separação entre histórico e configuração de fontes

A criação dos recibos e a limpeza usam as mesmas origens reservadas. A exclusão de histórico reconhece as identidades exatas `listenbrainz:<perfil>` e `file:<perfil>`, preservando outros registros de configuração. `TestShadowDeletePaths` inclui uma configuração do próprio perfil com prefixo `file:` que sobrevive à exclusão individual, à exclusão de histórico por catálogo e à limpeza global. A exclusão por catálogo remove todos os recibos e as integrações de sombra daquele perfil na mesma transação. O teste também confirma a preservação da configuração, da integração de sombra e dos recibos do outro perfil nas exclusões de escopo individual.

As entradas obrigatórias 3, 6 e 7 foram confirmadas na ISSUE-081. O comentário também registra a correspondência de exclusão quando um recibo `estimated` e a escuta legada caem em segundos diferentes.

A verificação desta correção passou em `make check` com 13 de 13 jornadas e em `go test -race` para os pacotes tocados. O Graphify AST foi atualizado.
