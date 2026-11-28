# ISSUE-218 — Parecer do Reviewer

27/11/2026, sobre `git diff main...HEAD` (commits `7b3f725` e `3e1ab90`). Revisão do coordenador sobre a implementação do Codex. Reproduções feitas em uma cópia em diretório temporário; a worktree não teve código alterado.

**REPROVADO** — cinco itens obrigatórios.

## Itens obrigatórios

1. **[BUG]** `internal/enrichment/subsonic.go`, `lookup` — falha em `getSong` ou `getAlbum`, ou nome de álbum diferente, descarta o MBID que a busca já tinha encontrado. Reproduzido: `getSong` 500 e `getAlbum` 500 devolvem `failed` sem declarações; álbum «Album (Deluxe)» devolve `ambiguous` sem declarações. Na v0.81 a escuta recebia o MBID. Corrigir: manter as declarações de identidade; a falha dos detalhes não as apaga.
2. **[BUG]** `internal/enrichment/musicbrainz.go`, `Lookup`/`lookupISRC` — ISRC desconhecido responde **404** na API pública (conferido com um código fictício) e `call` trata como falha: o resultado é `failed` e a busca por nome nunca roda. Corrigir: 404 é ausência; ausência ou falha da consulta por ISRC segue para a busca por nome.
3. **[BUG]** `internal/enrichment/musicbrainz.go`, `Lookup` — faixa com dois ISRCs devolve `ambiguous` sem consultar nada e sem busca por nome. O critério 6 trata como ambíguo o ISRC que leva a mais de uma gravação, não a faixa com mais de um ISRC. Corrigir: consultar os ISRCs e só declarar ambíguo com mais de uma gravação distinta.
4. **[BUG]** `internal/enrichment/musicbrainz.go`, `Lookup` — quando o ISRC encontra a gravação, os MBIDs de artista que faltam deixam de ser declarados (a busca por nome os trazia).
5. **[BUG]** `db/migrations_sqlite/026_descriptive_metadata.sql`, `trg_descriptive_scope_insert` e `trg_descriptive_listen_delete` — o `OR` entre colunas impede o uso de índice e o custo cresce com o número de escutas do perfil. Medido com `medir_gatilhos.py`:

   | Escutas | Gravar os metadados de 1 escuta | Apagar 1 escuta | Sem os gatilhos |
   | ---: | ---: | ---: | ---: |
   | 1.000 | 0,4 ms | 0,8 ms | 0,04 / 0,00 ms |
   | 10.000 | 4,0 ms | 7,8 ms | 0,04 / 0,01 ms |
   | 100.000 | 40 ms | 80 ms | 0,04 / 0,01 ms |

   A gravação ocorre com o bloqueio de escrita e o custo se repete mesmo quando todas as linhas são ignoradas. Por extrapolação, não medida, apagar um perfil ou expurgar 100 mil escutas com metadados levaria da ordem de uma hora. Os gatilhos de exclusão de ocorrência e de faixa têm a mesma forma e não foram medidos. Corrigir: uma condição por tipo de entidade, apoiada em índice, e repetir a medição.

Os testes existentes não cobrem nenhum dos cinco casos.

## Riscos

- [RISCO] `internal/enrichment/subsonic.go`, `Lookup` — escuta que já tem todos os IDs não gerava consulta; agora gera `search3` e, havendo correspondência, `getSong` e `getAlbum`, mesmo que a faixa já tenha metadados dessa instância. O cache dura 10 minutos. A uma chamada por segundo, isso limita a fila a poucas escutas por segundo e aumenta a carga no servidor do usuário.
- [RISCO] `internal/db/sqlite/enrichment.go`, `finishClaimedEnrichment` — passou a exigir lease não vencido: um trabalho com lease vencido fica em `working` até ser retomado, em vez de ser marcado como falho na hora.
- [RISCO] `internal/db/sqlite/enrichment.go`, `ResolveNextEnrichmentChain` — o armazenamento testa o tipo concreto `*enrichment.Subsonic` para pular a segunda chamada; uma propriedade do provedor evitaria o acoplamento.
- A exportação não leva os metadados novos (ISSUE-234).

## Não verificado nesta revisão

`make check` e o detector de corridas não foram repetidos (o Codex relata `make check` aprovado); serão rodados depois das correções. Mesclagens foram lidas, não reproduzidas.

**Advisor:** a mudança altera o banco e a fila de enriquecimento; recomenda-se acioná-lo depois das correções.

---

## Segunda revisão — commit `7aef45c`

27/11/2026, sobre `git diff 37ae3aa 7aef45c`.

**REPROVADO** — dois itens obrigatórios. Os cinco itens do primeiro parecer foram resolvidos.

### Itens obrigatórios

1. **[BUG]** `internal/enrichment/subsonic.go`, `lookup` (`partial`) — divergência de dados é tratada como falha passageira. Álbum com nome diferente no `getAlbum` marca o resultado como `Retryable`: não entra no cache e grava `metadata_retries`, que nunca sai. Reproduzido: cinco escutas da mesma faixa, com IDs completos e metadado já gravado, geram **15 chamadas** (3 por escuta), para sempre. Contraria a regra de consultar só quando faltar algo. O mesmo ramo vale para a divergência no `getSong`. Corrigir: só erro de chamada é repetível; divergência é resposta definitiva, entra no cache e não grava marcador.
2. **[BUG]** `internal/enrichment/subsonic.go`, `Lookup` — os campos pendentes agora dependem de `req.Prior`, que fica fora da chave do cache. Reproduzido: uma consulta com o ID da gravação em `Prior`, seguida de outra igual sem ele, devolve o resultado guardado, sem o MBID da gravação, por 10 minutos. Alcance estreito (exige a mesma escuta com `Prior` diferente). Corrigir: incluir na chave os campos pendentes calculados, com teste.

### Ajustes pequenos, a fazer junto

- `internal/db/sqlite/bootstrap.go` — a guarda de prévia da versão 26 só exige `descriptive_metadata`. Um banco criado pelo commit anterior desta branch passa pela guarda e não tem `metadata_retries`. Exigir também essa tabela (verificado pela leitura).
- `026_descriptive_metadata.sql` — `metadata_retries` não tem índice por `occurrence_id`: com chaves estrangeiras ligadas, excluir uma ocorrência custa 0,08 ms sem marcadores e 2,5 ms com 100 mil. Com o item 1 corrigido a tabela fica pequena; avaliar o índice e o efeito em bytes.

### Riscos aceitos ou a decidir

- Faixa com mais de quatro ISRCs sem resultado por nome devolve `failed`, sem cache. Caso raro.
- Metadado já gravado não é atualizado depois (limitação registrada no `HANDOFF.md`).
- Os gatilhos ficaram mais longos, com consultas curtas de atalho; o resultado é o mesmo dos anteriores (abaixo).

### Verificado nesta revisão

- `make check` aprovado, com 13/13 jornadas.
- `go test -race` em `internal/enrichment` e nos testes de enriquecimento, mesclagem e exclusão de `internal/db/sqlite`: aprovado. A suíte completa de `sqlite` com detector de corridas não foi rodada.
- Gatilhos novos contra os de `37ae3aa`: 4.000 bases aleatórias, 100 mil exclusões com chaves estrangeiras ligadas, **nenhuma diferença** nas linhas mantidas.
- Desempenho (`medir_gatilhos.py` repetido e `medir_gatilhos_fk.py`, biblioteca com um artista em 20% das faixas, 100 mil escutas): gravação 0,06 ms (antes 40 ms); exclusão de uma escuta 0,05 ms (antes 80 ms); apagar todas as escutas 3,1 s; apagar o perfil 0,9 s.
- API pública do MusicBrainz aceita `inc=artist-credits` na consulta por ISRC (código de uma gravação conhecida, agente genérico).
- As mutações relatadas pelo Codex não foram repetidas.

Reproduções dos dois itens em `reproducoes-parecer-2.go.txt`. **Advisor:** recomendado depois das correções.
