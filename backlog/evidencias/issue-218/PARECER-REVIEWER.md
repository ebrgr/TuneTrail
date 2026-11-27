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
