# ISSUE-172 — Limpeza do código legado

## Alterações

- Removidas as operações sem rota `PreviewIngestActivation`,
  `SetIngestActivation` e `ConvertOccurrence`. A verificação compartilhada de
  sessão root ficou em `internal/db/sqlite/root_session.go`. Os tipos de
  preview foram removidos de `internal/db/ingest_activation.go`.
- Removidos `catalog.SubmitListen`, `catalog.ResolveListenTrack`,
  `Sqlite.SaveListen` e `Sqlite.SaveListenWithResult` da aplicação.
  `SaveManualListen` é o escritor da escuta manual.
- O caminho de produção de `AdmitOccurrence` grava um lote de ocorrência
  sob a mesma reserva de escrita. O worker não despacha recibos para o
  resolvedor v9. As migrações existentes e as colunas históricas foram
  preservadas.
- A configuração de relay usa o rascunho e a aprovação de rede do schema
  21. O CLI de replay de sombra, seu validador e os ensaios que dependiam
  dos modos anteriores foram retirados da branch. Os scripts históricos
  estão disponíveis no commit `d2ea6ba`. O ensaio 172 e o benchmark 092
  usam os auxiliares preservados em `scripts/rehearsal_support.py`.
- A busca usou referências Go, graphify e `go vet`. O executável
  `staticcheck` não estava instalado. As tentativas de compilar `deadcode`
  com os módulos locais encontraram dependências ausentes do cache.

## Testes

As asserções de associação de `TestSubmitListen_*` foram portadas para
`PublishNowPlaying`. A fixture usa `SaveManualListen` para as asserções de
histórico. A falha de associação continua sendo verificada pelo caminho
ao vivo.

As fixtures de atividade, isolamento e resolução por ocorrência passaram
para o schema atual. A admissão nessas fixtures passa por `AdmitOccurrence`.
O auxiliar que reproduzia a admissão antiga e o interruptor foi removido. Os
testes de atualização de retry e reprocessamento e as fixtures históricas
da Inbox passaram a inserir recibos por SQL. Os testes de replay de digest
v1 e v9 inserem o recibo histórico por SQL e exercitam a admissão atual
após a migração. As asserções de preservação observam os mesmos digests,
contagens e guardas. Os testes v2 → v21 e v20 → v21 continuam na suíte,
juntamente com a verificação de backups.

Foram retirados os testes exclusivos das ações de ativação e conversão
removidas, do bloqueio de importação com o interruptor desligado,
da substituição de recibos de sombra sem escuta e do apagamento
de histórico nesse modo. A migração 021 verifica a recusa de recibos incompatíveis.
`TestOccurrenceWriteConflicts` continua exercitando diretamente as guardas
do schema 20. O teste de recuperação de lease usa o resolvedor atual e
verifica que uma reivindicação obsoleta não materializa uma escuta. A remoção de histórico com lease em aberto continua
coberta por `TestOccurrenceHistoryDeleteWithUnresolvedLease`.

A suíte Go completa e `go vet ./...` passaram. Os testes foram executados
com os diretórios de assets e idiomas da worktree. O `make check` passou
com 13/13 jornadas E2E. A execução completa com `-race` foi iniciada
no mesmo código e segue sem resultado registrado neste ponto.

## Escuta ao vivo

Passaram `TestNowPlaying`, `TestStaleNowPlayingLookupDoesNotEraseNewUpdate`,
`TestNowPlayingIsScopedAndMissingTrackClearsOnlyItsProfile`,
`TestHomeNowPlayingAppearsBeforePeriodSummary` e
`TestMediaNowPlayingMatchesEntityAndProfile`.

No binário compilado após a limpeza do caminho de admissão, uma instância
com banco novo em `/tmp` na porta 4127 devolveu 200 para `playing_now`.
A faixa apareceu em `now-playing` e na home, com zero escutas no histórico.
Uma faixa diferente enviada como `single` recebeu 200 e apareceu como a
única escuta no histórico. O processo foi encerrado e o diretório
temporário removido.

## Carga em cópia privada

A instância e o mock usaram portas temporárias de loopback. O processo do
TuneTrail teve a saída de rede restrita ao mock. O lote misto no binário
final fez 120 requisições a 15,11/s, com ACK p95 de **7,52 ms** (limite de 200 ms),
visibilidade máxima de **1,031 s** e zero 503. As 109 escutas aceitas foram
materializadas. As 11 requisições sem artista receberam 400. O legado produziu as mesmas
109 escutas, com digest igual na projeção por perfil, segundo e cliente.

O piloto sustentou 15,01 escutas/s. As 900 escutas receberam 200, com ACK
p95 de **45,80 ms**, visibilidade máxima de **1,287 s**, zero 503 e todas as escutas materializadas.

## Ainda em compatibilidade

As fixtures históricas restantes ainda usam `finishIngest` e
`finalizeActiveReceipt` em `historical_resolution_test.go`. `ingestReviewEnabled`, `healthReviews`, `SetIngestReview`,
`insertRelayIntent` e `RelayPolicyProfiles` ainda têm caminhos para schemas
antigos. O teste de referência da interface ainda gera a página de saúde
com uma fixture do schema 20.
A retirada desses caminhos ainda exige portar as fixtures correspondentes.
