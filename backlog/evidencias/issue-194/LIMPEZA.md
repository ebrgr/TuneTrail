# ISSUE-194 — Limpeza de compatibilidade anterior ao schema 21

## Alterações

| Arquivo | Limpeza |
| --- | --- |
| `internal/db/sqlite/historical_resolution_test.go` | Removidos `finishIngest`, `finalizeActiveReceipt` e `errIngestTrackMissing`. As fixtures inserem recibos por SQL e chamam o resolvedor de ocorrência. |
| `internal/db/sqlite/ingest_review.go` | `ingestReviewEnabled`, `healthReviews` e `SetIngestReview` usam `profile_ingest_review`, com revisão ligada quando não há override. `resolveIngestReview` perdeu o escritor manual do v9 e o uso de `review_distinct`. |
| `internal/db/sqlite/ingest_receipts.go` | Removidos `ingestSchemaVersion` e a interface `ingestModeReader`. A interface pequena compartilhada pela leitura de revisão ficou com seu consumidor. |
| `internal/db/sqlite/relay_outbox.go` | `insertRelayIntent` exige a geração aprovada capturada no recibo. Retirados o caminho sem esse vínculo e a escrita de marcadores `pre_082_best_effort` para recibos v9 abertos. |
| `internal/db/sqlite/relay_panel.go` | `RelayPolicyProfiles` lê diretamente `awaiting_approval`. |
| `internal/db/sqlite/ingest_delete.go` | `deleteSelectedActiveReceipts` perdeu a busca por aliases e a criação de revisões para recibos v9 abertos. A 021 impede que esses recibos cheguem ao schema suportado. A exclusão por ocorrência resolvida continua. |
| `internal/db/sqlite/{relay_schema,root_session,sqlite}.go` | Removidos `migrateDatabase`, `shadowPurgeScope` e `tzOffset`, sem chamadas ou leituras. |
| `engine/handlers/{image_handler,stats}.go` e `internal/db/period_test.go` | Removidos `downloadMissingImage`, `safeRatio`, `bod`, `eod` e o teste vazio cujas chamadas estavam comentadas. |

As migrações publicadas permanecem intactas. O inventário de colunas e
tabelas está em [COLUNAS.md](COLUNAS.md). A mudança é interna ao schema 21 e
às fixtures, sem entrada de changelog.

## Cobertura portada

- `TestResolveIngestLegacyCollision` ainda usa uma escuta pré-upgrade. Os
  estados `duplicate` e `pending_duplicate`, motivos, contagem de escutas e
  revisão única são verificados com o resolvedor atual. O segundo uso do
  mesmo claim agora verifica um retorno sem efeito, conforme o contrato de
  `finishOccurrence`.
- Os testes de outbox, revisão, paginação, descarte e corridas semeiam
  ocorrências por SQL. A fixture conserva relays já configurados via
  migração real. O teste de troca de geração modela aprovação por SQL para
  testar o fencing sem DNS. Em `TestRelayConfigurationTransitions`, as
  expectativas pré-021 de que a troca do token mantinha a fila aberta e
  preservava a geração foram substituídas pelas guardas de rascunho e da
  nova geração. A equivalência de URLs e a expiração de trabalho antigo
  continuam cobertas.
- Foi retirada a expectativa de `ErrConflict` ao habilitar revisão em
  perfil que não participou da ativação. Isso dependia de `active=1` no
  schema antigo. A disponibilidade, os overrides e o isolamento entre
  perfis seguem cobertos no schema atual.
- `TestProfileDeleteClosesV9Ingest` foi retirado. Sua seleção por alias e
  ID de catálogo, o fechamento com revisão de invalidação e o replay do
  recibo invalidado cobriam exclusivamente a política de exclusão do
  resolvedor v9. Permanecem `TestOccurrenceHistoryDeleteWithUnresolvedLease`,
  `TestProfileDeleteRemovesResolvedOccurrencePending` e
  `TestRelayHistoryDeletion` para os contratos atuais de exclusão, vínculo
  de escutas e isolamento. Testes de guarda do schema antigo, backups,
  migrações v2 → 21 e v20 → 21 permanecem.
- `TestUIReference` usa o schema atual. Só `admin/ingest-health.html` e
  `admin/listen.html` mudaram na comparação de 188 arquivos. A amostra
  histórica de sombra foi substituída por um recibo ativo com diagnóstico,
  mudando o estado exibido e os digests de descarte da fixture. O relay
  passa de “Not tracked” para “Not queued” no recibo ativo, e o detalhe da
  escuta de API mostra `listenbrainz_api` como origem.

## Análise de referências

`staticcheck` v0.8.2 e `deadcode` (x/tools v0.53.0) foram instalados em
`/tmp` e executados com `./...` (`deadcode -test ./...`). As remoções
adicionais foram confirmadas por referências e graphify. `instanceLock`,
apontado pelo staticcheck, mantém o descritor do lock durante a vida do
servidor. Os demais diagnósticos de estilo e os candidatos exportados
permaneceram fora da limpeza. A busca repetida confirmou a retirada dos
três símbolos centrais e das comparações operacionais de versão 21. A
menção remanescente a `profile_ingest_flags` no Go de produção pertence à
validação de layout. O grafo AST foi atualizado.

## Escuta ao vivo

No binário final, em banco novo em `/tmp` e porta 4127, `playing_now`
retornou 200. A faixa apareceu no endpoint de escuta ao vivo e na home.
O banco permaneceu com zero escutas. Um `single` diferente retornou 200 e
apareceu como a única escuta no histórico web e no SQLite. O processo foi
encerrado ao fim do roteiro.

## Carga e paridade

`scripts/rehearse_issue172.py --total 120` rodou contra o binário da
base da branch e o final, em cópias do mesmo banco sintético, com o mesmo
intervalo de timestamps e saída de rede limitada ao mock de loopback.

| Medida | Base | Final |
| --- | --- | --- |
| Requisições HTTP | 109 × 200, 11 × 400 | 109 × 200, 11 × 400 |
| ACK p95 | 100,10 ms | 5,83 ms |
| Visibilidade máxima | 1.124,52 ms | 998,65 ms |
| Ritmo | 15,12 req/s | 15,12 req/s |
| Escutas aceitas e materializadas | 109 | 109 |

As 11 respostas 400 correspondem aos itens sem artista. Houve zero 503 e
zero escutas aceitas ausentes, com preservação da escuta pré-existente e
mesmo digest da projeção por perfil, segundo e cliente
(`c0b7d5595780d6b59b4e8bc02bebb8fadfcca8cafa57612b4c91bf8b18e1243b`).
O piloto adicional do binário final confirmou 900/900 materializações a
15,01 req/s, ACK p95 de 10,96 ms e máximo de 1.017,31 ms até a
visibilidade. A falha sintética recebeu 200 e estacionou após cinco
tentativas, com `retry_exhausted`. Diferenças de latência são observações
de execuções locais sob cargas concorrentes.

## Verificação automatizada

- `go vet ./...` e `go test -count=1 -timeout 30m ./...` passaram.
- `make check` passou no código commitado, com 13/13 jornadas E2E e
  igualdade dos 188 arquivos da referência após a atualização.
- Passaram em execução direcionada `TestBootstrapKoitoV2WithoutUpstreamIndexes`,
  `TestSchema021DevelopmentUpgrade` e `TestProfileMigrationPreservesLegacyDataAndBackup`.
- Passaram as fixtures de escala `TestRelayE4Store` (300 mil recibos,
  100 mil itens de outbox), `TestRelayMutationMeasurements` e
  `TestIngestDiscardScale`.
- A execução com `-race -count=1 -timeout 30m` e filtro
  `Ingest|Relay|Occurrence|Import|NowPlaying|Live` passou em SQLite
  (743,4 s), ingest, catalog, relay e engine (452,2 s). O filtro não
  selecionou testes nos pacotes importer e admin.
  As suítes completas de importer e admin passaram em uma execução
  complementar com as mesmas opções de race, sem filtro.
