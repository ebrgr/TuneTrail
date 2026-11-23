# ISSUE-172 — Inventário da fase 2

A base é o commit `0d5a98a`, que já contém a mesclagem da main e as
ISSUE-181 a 188. Produção segue de v2 diretamente ao último schema.
Os bancos intermediários de desenvolvimento são casos separados de teste.

## Escrita e admissão

| Chamador ou componente | Decisão |
| --- | --- |
| `engine/handlers/lbz_submit_listen.go`, `LbzSubmitListenHandler` | Manter admissão durável e ACK após commit para single e import. Retirar o argumento mestre, o fallback `catalog.SubmitListen` e o relay legado dessa ramificação. Preservar now-playing efêmero. |
| `internal/importer/shadow.go`, `withFileShadow`, `flushFileShadow`, `flushLegacyFileShadow` e chamada de `catalog.SubmitListen` | Manter digest, ordinal, flush e retomada da admissão por ocorrência. Retirar gravação direta, registro de sombra e seleção por mestre. |
| `internal/importer/listenbrainz.go` | Manter o adaptador por ocorrência. Retirar a ramificação `catalog.SubmitListen`. |
| `internal/importer/koito.go` | Manter associação explícita ao catálogo e captura de `CatalogTrackID` para o formato nativo anterior. Retirar a ramificação `SaveListen`. Manter o desvio para o formato de arquivo versionado. |
| `internal/importer/upload.go`, `countingStore` | Migrar contadores e hooks para admissão e resolução. Retirar `SaveListenWithResult` deste wrapper, argumentos mestre e captura de sombra. |
| `internal/admin/listens.go`, `ManualListen`; `engine/handlers/post_listens.go`; `internal/adminui/media.go` | Migrar para a ocorrência escolhida após autorização, preservando validação do horário e cliente. O tratamento de colisão entre fontes requer decisão, detalhada abaixo. |
| `internal/catalog/catalog.go`, `SubmitListen` | Retirar a fachada de escrita depois de migrar chamadores. Manter a lógica de catálogo usada de forma independente, incluindo `ResolveListenTrack`. |
| `internal/db/db.go`, `internal/db/sqlite/listen.go` | Retirar `SaveListen` e `SaveListenWithResult` quando cessarem os chamadores. Manter leitura e exclusão. A exclusão pela tripla ainda recusa ambiguidades e a rota por ID identifica o evento exato. |
| `internal/ingest/envelope.go`, `internal/db/sqlite/ingest_receipts.go` | Manter o contrato de admissão e a fila por ocorrência. Retirar o argumento mestre, `RecordShadow` e a escolha entre sombra, v9 e ocorrência para novas admissões. |
| `internal/db/sqlite/archive.go`, `RestoreArchive` | Manter a restauração transacional por ocorrência e a origem `archive_restore`. Preservar deduplicação por namespace e ID de origem e o mapeamento do catálogo. |
| `scripts/ingest_shadow_replay` e ensaios históricos | Retirar os consumidores do modo removido do produto corrente. Evidências históricas mantêm sua referência à versão ensaiada. Atualizar o ensaio 172 no corte de medição. |

## Persistência e interface

| Área | Decisão |
| --- | --- |
| `cfg.ActiveIngestEnabled` e `TUNETRAIL_INGEST_ACTIVE` | Retirar configuração, aliases, documentação e exemplos após a transição de fixtures. |
| `engine/handlers/ingest_activation.go`, rotas e controles de ativação | Retirar ativação e conversão por perfil. Manter Revisar por perfil e sua interface administrativa. |
| `relay_management.go`, `profile_store.go`, `relay_allowlist.go` | Retirar a condição do interruptor e a conversão dependente de flag. Manter validação do destino, allowlist e entrega durável. |
| `profile_ingest_flags` | Retirar na 021 e substituir a preferência Revisar por estrutura própria, ligada para todo perfil existente ou novo conforme o plano. |
| Gatilhos de ativação das migrações 009–012 | Retirar os guardiões que dependem de `profile_ingest_flags` e a admissão de sombra. Manter a integridade das referências de recibo, escuta, ocorrência, gravação e revisão. Preservar os arquivos históricos de migração para o salto v2 → última. |
| `idx_listens_legacy_identity` (008, 014, 016) | Retirar após extinguir o escritor. O índice único e parcial arbitra inserts legados. Nenhum leitor força seu uso. Conferir os planos de leitura após remoção. |
| `origin='legacy'` | Manter as linhas e sua origem. Manter a distinção de proveniência nas leituras e exportações, inclusive em `ingest_health.go`. Revisar a seleção preferencial do resolvedor v9 em `ingest_resolve.go` e o tratamento de colisão em `ingest_review.go` junto à remoção do v9. |
| Recibos antigos de sombra e recibos v9 abertos | Na 021, recusar com diagnóstico acionável se o banco de desenvolvimento ainda tiver trabalho incompatível. Nunca apagar escutas. O salto produtivo v2 → última precisa preservar cada escuta e comprovar restauração pelo backup. |

## Inventário mecânico dos chamadores

As ocorrências abaixo incluem fixtures e chamadas que exercitam especificamente
a compatibilidade. Fixtures de histórico corrente passam ao helper de ocorrência
e aguardam resolução. Testes de migração podem semear a representação do schema
de origem. Casos específicos de seleção do mestre, flag ou sombra permanecem
até a remoção das respectivas superfícies.

~~~text
engine/adminui_media_test.go:89
engine/auth_race_issue020_test.go:37
engine/auth_race_issue020_test.go:83
engine/handlers/lbz_submit_listen.go:215
engine/handlers/lbz_submit_listen.go:217
engine/handlers/lbz_submit_listen.go:343
engine/import_cancel_issue020_test.go:24
engine/import_cancel_issue020_test.go:25
engine/import_recovery_test.go:57
engine/import_recovery_test.go:65
internal/admin/listens.go:84
internal/admin/listens.go:86
internal/admin/listens.go:88
internal/catalog/catalog.go:145
internal/catalog/issue020_test.go:32
internal/catalog/submit_listen_test.go:53
internal/catalog/submit_listen_test.go:103
internal/catalog/submit_listen_test.go:142
internal/catalog/submit_listen_test.go:174
internal/catalog/submit_listen_test.go:256
internal/catalog/submit_listen_test.go:317
internal/catalog/submit_listen_test.go:372
internal/catalog/submit_listen_test.go:421
internal/catalog/submit_listen_test.go:474
internal/catalog/submit_listen_test.go:539
internal/catalog/submit_listen_test.go:580
internal/catalog/submit_listen_test.go:627
internal/catalog/submit_listen_test.go:665
internal/catalog/submit_listen_test.go:704
internal/catalog/submit_listen_test.go:761
internal/catalog/submit_listen_test.go:842
internal/catalog/submit_listen_test.go:931
internal/catalog/submit_listen_test.go:978
internal/catalog/submit_listen_test.go:1021
internal/db/db.go:75
internal/db/sqlite/activity_issue018_test.go:21
internal/db/sqlite/activity_issue018_test.go:22
internal/db/sqlite/activity_issue018_test.go:46
internal/db/sqlite/activity_issue018_test.go:62
internal/db/sqlite/activity_issue018_test.go:64
internal/db/sqlite/activity_issue018_test.go:98
internal/db/sqlite/activity_issue018_test.go:120
internal/db/sqlite/activity_issue018_test.go:145
internal/db/sqlite/concurrency_issue020_test.go:22
internal/db/sqlite/concurrency_issue020_test.go:44
internal/db/sqlite/ingest_activation_test.go:150
internal/db/sqlite/ingest_activation_test.go:158
internal/db/sqlite/ingest_activation_test.go:170
internal/db/sqlite/ingest_activation_test.go:234
internal/db/sqlite/ingest_activation_test.go:254
internal/db/sqlite/ingest_activation_test.go:279
internal/db/sqlite/ingest_activation_test.go:345
internal/db/sqlite/ingest_activation_test.go:353
internal/db/sqlite/ingest_discard_test.go:226
internal/db/sqlite/ingest_inbox_test.go:34
internal/db/sqlite/ingest_occurrence_duplicate_test.go:32
internal/db/sqlite/ingest_occurrence_duplicate_test.go:58
internal/db/sqlite/ingest_occurrence_e4_test.go:60
internal/db/sqlite/ingest_occurrence_e4_test.go:70
internal/db/sqlite/ingest_occurrence_resolver_test.go:46
internal/db/sqlite/ingest_occurrence_resolver_test.go:114
internal/db/sqlite/ingest_occurrence_resolver_test.go:176
internal/db/sqlite/ingest_occurrence_resolver_test.go:233
internal/db/sqlite/ingest_occurrence_resolver_test.go:238
internal/db/sqlite/ingest_occurrence_resolver_test.go:240
internal/db/sqlite/ingest_occurrence_resolver_test.go:425
internal/db/sqlite/ingest_occurrence_resolver_test.go:462
internal/db/sqlite/ingest_occurrence_resolver_test.go:465
internal/db/sqlite/ingest_occurrence_resolver_test.go:504
internal/db/sqlite/ingest_occurrence_resolver_test.go:520
internal/db/sqlite/ingest_occurrence_resolver_test.go:528
internal/db/sqlite/ingest_occurrence_resolver_test.go:583
internal/db/sqlite/ingest_purge_test.go:69
internal/db/sqlite/ingest_receipts.go:28
internal/db/sqlite/ingest_receipts.go:33
internal/db/sqlite/ingest_receipts.go:99
internal/db/sqlite/ingest_receipts.go:100
internal/db/sqlite/ingest_receipts.go:105
internal/db/sqlite/ingest_retry_test.go:24
internal/db/sqlite/ingest_review_test.go:142
internal/db/sqlite/interest_test.go:96
internal/db/sqlite/isolation_issue019_test.go:25
internal/db/sqlite/listen.go:16
internal/db/sqlite/listen.go:17
internal/db/sqlite/listen.go:22
internal/db/sqlite/profiles_test.go:134
internal/db/sqlite/profiles_test.go:135
internal/db/sqlite/profiles_test.go:138
internal/db/sqlite/profiles_test.go:139
internal/db/sqlite/profiles_test.go:140
internal/db/sqlite/profiles_test.go:141
internal/db/sqlite/profiles_test.go:154
internal/db/sqlite/profiles_test.go:158
internal/db/sqlite/profiles_test.go:343
internal/db/sqlite/recording_identity_test.go:110
internal/db/sqlite/recording_identity_test.go:114
internal/db/sqlite/recording_identity_test.go:298
internal/db/sqlite/relay_outbox_test.go:705
internal/importer/koito.go:263
internal/importer/listenbrainz.go:247
internal/importer/shadow.go:42
internal/importer/shadow.go:143
internal/importer/shadow.go:196
internal/importer/upload.go:42
internal/importer/upload.go:59
internal/importer/upload.go:61
internal/importer/upload.go:67
internal/importer/upload.go:69
internal/importer/upload.go:75
internal/importer/upload.go:76
internal/ingest/envelope.go:105
internal/ingest/envelope.go:106
internal/ingest/envelope.go:111
internal/webui/handlers_test.go:58
internal/webui/home_kpis_test.go:34
internal/webui/home_kpis_test.go:122
internal/webui/i18n_051_test.go:22
internal/webui/i18n_051_test.go:110
internal/webui/media_test.go:282
internal/webui/rewind_test.go:33
internal/webui/rewind_test.go:124
internal/webui/timezone_issue018_test.go:23
~~~

## Decisão de contrato pendente

A escuta manual usa uma ocorrência escolhida pela pessoa, responde HTTP 201
na API e informa na interface se inseriu ou se a escuta já existia.
No caminho legado, a unicidade só compara linhas `origin='legacy'` e
`review_distinct=0` pela tripla perfil, ocorrência e segundo. Em um perfil
convertido, o guardião atual bloqueia a gravação manual. Assim, o código
atual não fornece um comportamento manual por ocorrência para conservar.
O doc-039 e a decisão 008 definem colisões de API e arquivo, mas não definem
o que a ação manual deve fazer quando uma escuta de outra fonte ocupa a
mesma tripla. A escolha muda a contagem, o estado do Inbox e as garantias da
resposta administrativa. O corte 1 deve aguardar a definição dessa regra
para que suas fixtures não fixem por antecipação um comportamento de produto.

## Verificação do corte 0

`make check` passou com o alvo Go executado com `-timeout 30m`, por meio
de um override de Makefile fora do repositório. Os 13 percursos E2E passaram
nas bases descartáveis das fixtures. O corte não altera runtime nem schema.

## Corte 1 — Regra manual definida e ponto de parada

A decisão sobre a escuta manual foi registrada em `081563f`. Quando a
tripla já tem uma escuta aceita de qualquer fonte, a ação deve informar
que já existe, mantendo contagem e Inbox.

A fixture `testfixtures.OccurrenceListen` usa admissão e resolução reais,
verificando que cada recibo aceito aponta para a ocorrência e o segundo
esperados. As fixtures de histórico web e de leituras SQLite convertidas
passaram. O E2E de múltiplos perfis exercita ocorrência com espera pela
materialização. O `make check` passou com mestre ligado no Makefile, prazo
Go de 30 minutos e 13/13 E2E. A referência visual ainda usa uma fixture
histórica explícita de sombra. Os cenários legados restantes conservam suas
expectativas até a conversão.

Antes de converter as asserções da importação nativa, foi executada
uma comparação local em duas bases descartáveis em
`~/Documents/TuneTrail-ensaio-172/phase2/`, com saída de rede bloqueada.
O mesmo JSON Koito v1 continha uma escuta e dois aliases explícitos
(um primário e um secundário) para cada entidade.

| Modo | Escutas | Aliases de artista | Aliases de álbum | Aliases de faixa |
| --- | ---: | ---: | ---: | ---: |
| Legado | 1 | 2 | 2 | 2 |
| Ocorrência | 1 | 1 | 1 | 1 |

O caminho por ocorrência em `internal/importer/koito.go` copia para o
snapshot os nomes primários. O caminho legado passa os aliases fornecidos
ao catálogo. O doc-039 define identidade e proveniência, porém não define
a política de importação de aliases secundários do Koito v1 quando o caminho
legado deixa de existir. A fase foi pausada antes de mudar a asserção que
exige preservar os aliases do arquivo ou estender o escritor de ocorrência
para importá-los. A comparação preservou a quantidade de escutas. O risco
observado é de metadados do arquivo aceito.

### Retomada e verificação

Com a decisão registrada no plano, o teste de ocorrência confirmou uma escuta,
três aliases secundários preservados e três conflitos ignorados, sem pendência
por alias. A contagem durável aparece no resumo ao repetir o mesmo arquivo,
sem nova escuta. O alias mantém a proveniência do perfil e fica fora da
resolução global de identidade. `make check` ficou verde com o mestre ligado e
13/13 E2E.

Antes da remoção, a comparação em duas cópias isoladas enviou 450 requisições a
15/s por modo. Ambos materializaram os 409 ACKs e preservaram as 80 escutas
históricas. A projeção perfil/segundo/cliente coincidiu. Por ocorrência, p95 de
ACK 3,64 ms, visibilidade 0,971 s e HTTP 503 zero. Esta é uma medição da base,
anterior ao corte de schema.

## Corte 2 — Escuta manual e ponto de parada na admissão HTTP

A escuta manual agora verifica a tripla perfil, ocorrência e segundo sob
a reserva de escrita do SQLite. Uma escuta aceita de API, arquivo, arquivo
restaurado ou origem manual existente mantém a contagem e o resultado
administrativo de duplicata. A ação manual guarda a origem `manual` e
continua síncrona, sem criar relay ou pendência. O teste da rota cobre
colisão com escuta aceita da API e repetição de inclusão manual. O percurso
público E2E passou com todos os perfis dessa fixture convertidos.

O processamento de `playing_now` foi separado da escrita histórica. As
duas mudanças passaram em `make check`, com 13/13 E2E.

Antes de trocar a rota histórica para ocorrência obrigatória, o teste
`TestMappedArtistMBIDResolution170` acusou perda de atribuição. Uma confirmação
isolada em bancos descartáveis, com sandbox sem saída de rede, comparou o
binário estável ao candidato à retirada do fallback. Ambos receberam o
mesmo item válido: nome com grafia diferente e MBID em
`mbid_mapping.artist_mbids`, já existente no catálogo.

| Medida após materialização | Binário estável | Candidato |
| --- | ---: | ---: |
| Escutas no histórico do perfil | 1 | 1 |
| Escutas no ranking do artista do MBID | 1 | 0 |
| Artistas no catálogo | 1 | 2 |

A escuta existe no candidato, mas é atribuída a um novo artista. Isso viola
a compatibilidade da ISSUE-170 e muda o histórico por artista e o ranking
ao retirar o fallback. A precedência entre um MBID mapeado recebido e os
créditos de ocorrência exige conciliar o contrato de compatibilidade com
a prova de identidade do doc-039. O catálogo de artistas pode conter MBIDs
legados sem marca de verificação independente.

A troca experimental da rota histórica não foi commitada. A worktree
permanece no último ponto verde. O restante do corte 2 e os cortes
3 a 5 ficam pendentes. A medição de 450 requisições por modo é da base
anterior à retirada do escritor.

### Retomada da rota após a decisão sobre MBID

O candidato passou a vincular o artista já identificado. Na comparação
isolada com o mesmo payload de MBID, o histórico do perfil, o ranking
do artista original e o total de artistas ficaram em 1, tal como no legado.
O teste cobre os dois campos de MBID, a ausência de correspondência, o
crédito sem álbum e a ambiguidade causada por duas grafias do mesmo UUID.

A comparação isolada da rota histórica renovada enviou 450 requisições
por modo a 15/s. Ambos materializaram todos os 409 ACKs e preservaram as 80
escutas históricas. A projeção perfil/segundo/cliente e o digest da projeção
coincidiram. No candidato, p95 de ACK 9,49 ms, p95 de visibilidade 0,971 s,
máxima 1,029 s e zero HTTP 503. Esta medição antecede a migração 021.

### Ponto de parada na importação Koito v1

A rota HTTP histórica foi commitada após `make check` verde e 13/13 E2E.
A conversão do importador foi avaliada separadamente em bancos
descartáveis fora do repositório, com todas as conexões de saída do servidor
bloqueadas. Para cada caso, os dois binários receberam os mesmos bytes
num perfil sem histórico ou flags. Nenhum arquivo continha dados do usuário.

| Fixture Koito v1 | Legado | Candidato por ocorrência |
| --- | --- | --- |
| 1 escuta, 1 alias secundário, 387 bytes | HTTP 204, 1 escuta e 1 alias | HTTP 204, 1 escuta, 1 alias e 1 recibo ativo |
| 1 escuta, 220 aliases secundários, 18.345 bytes | HTTP 204, 1 escuta e 220 aliases | HTTP 400, 0 escutas, 0 aliases e 0 recibos |

O arquivo maior respeita os limites públicos de tamanho de upload e JSON.
O adaptador por ocorrência carrega os aliases explícitos no snapshot, que
ultrapassa o limite de 10 KiB por item em `captureFileListen`. A falha
acontece antes da admissão, portanto não há recibo para retomar ou revisar.
Os aliases precisam ser preservados pela decisão já aprovada, então
truncá-los também violaria o contrato. O ensaio confirmou a perda da escuta
na comparação e acionou a condição de parada.

A troca experimental do importador foi retirada. A validação da rota HTTP
ocorreu antes da conversão obrigatória dos importadores e da migração 021.
No schema 20, o guardião ainda recusa a conversão de perfis com sombra
sem vínculo ou recibo v9 aberto. A migração, a documentação final e a
medição completa continuam pendentes.

### Orçamento de aliases e verificação da conversão do importador

Foi aplicada a decisão sobre aliases secundários. O adaptador mede o JSON
do recibo, reserva espaço para os dados da escuta e guarda o maior prefixo
de aliases que cabe nos 10 KiB. O corte do prefixo respeita a ordem dos
campos e das listas do arquivo. O resumo apresenta os excedentes por
tamanho separadamente dos conflitos de propriedade.

O candidato de importação obrigatória por ocorrência reproduziu as
contagens do legado para as cinco fixtures do repositório (Koito 13,
Last.fm 10, ListenBrainz ZIP 19, Spotify 1 e Maloja 38). Num teste
HTTP isolado, o arquivo com 1 alias teve 1 escuta e 1 alias nos dois
modos. O arquivo com 220 aliases teve 1 escuta nos dois modos. No reensaio
HTTP com exatamente 18.345 bytes, o candidato
preservou 143 aliases e registrou os 77 restantes no recibo.

O teste de fronteira também encontrou perda independente dos aliases.
Um arquivo Koito v1 válido de 10.807 bytes, sem aliases secundários, com
10.500 bytes no campo `client`, retornou HTTP 204 e materializou 1 escuta
no legado. O candidato retornou HTTP 400, sem escuta nem recibo. O snapshot
já ultrapassava 10 KiB antes de qualquer alias. A comparação usou bases
descartáveis locais com conexões de saída bloqueadas.

A correção de orçamento foi mantida no caminho por ocorrência já
existente. A troca experimental dos importadores foi retirada após a
confirmação da perda. O restante do corte 2 e os cortes 3 a 5 seguem
pendentes. O ensaio atual não é prova de migração 021 nem de restauração.

### Normalização de metadados opcionais e fronteira dos campos obrigatórios

A preparação de recibos de arquivo limita cliente a 255 bytes e demais
textos opcionais a 1 KiB, respeitando fronteiras UTF-8. Quando o conjunto
de opcionais ainda estoura o envelope, descarta primeiro listas de
créditos/identificadores opcionais e depois textos de cliente e álbum,
com contagem no resumo. Aliases usam o orçamento remanescente. Título e
artista acima de 4 KiB recebem erro explícito. Os testes cobrem os cinco
adaptadores.

A comparação HTTP isolada do Koito v1 com cliente de 10.500 bytes gravou
uma escuta nos dois modos. No candidato, o cliente ficou em 255 bytes e
o recibo registrou um campo truncado.

A mesma matriz confirmou uma incompatibilidade entre os limites para
campos obrigatórios. Um arquivo válido com artista de 3.121 bytes e título
de 3.141 bytes, contendo aspas e barras invertidas, foi aceito no legado
(HTTP 204, uma escuta). O candidato retornou 400, sem escuta ou recibo.
O escape JSON ocupa mais de 10 KiB só para os dois campos obrigatórios.
Reduzir opcionais ou aliases não resolve essa entrada. Aceitá-la exigiria
alterar a representação do recibo, rever o limite serializado ou permitir
truncar campos obrigatórios.

A normalização foi mantida no caminho por ocorrência existente. A troca
experimental dos importadores foi retirada. A 021, a documentação final
e o ensaio de backup/restauração continuam pendentes.
