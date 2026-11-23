# ISSUE-172 — Ensaio da fase 1

O caminho por ocorrência conservou as 80 escutas preexistentes. Das 2.250
requisições da carga principal, 1.840 viraram escutas, 205 ficaram no Inbox
aguardando identificação do álbum e 205 foram rejeitadas por falta de artista.
Os 2.045 ACKs foram reconciliados por chave de idempotência e segundo. Cada
recibo aceito referencia exatamente uma escuta no perfil esperado. Os 205
recibos pendentes continuam no banco e impedem considerar cumprido o prazo
de visibilidade para todos os ACKs.

## Cópia, isolamento e execução

A instância de desenvolvimento na porta 4110 continuou com o mesmo processo.
O SQLite abriu o banco de origem em modo somente leitura e criou a cópia
com `.backup` em
`~/Documents/TuneTrail-ensaio-172/`. Antes do primeiro boot das cópias de
trabalho, o script removeu chaves, sessões, tokens de conta, destinos e
allowlist herdados, revogou os registros de integração e substituiu as
credenciais locais de login. A configuração de Subsonic ficou vazia.
MusicBrainz, Deezer e Cover Art Archive ficaram desabilitados. A migração de
18 a 20 criou backup automático na pasta privada.

O sandbox do macOS permitiu ao processo do servidor conexões de saída somente
para o mock em `localhost:4121`. O ensaio confirmou conexão ao mock e recusa
por permissão a um IP externo e à instância 4110. O mock descartou o conteúdo
após contabilizar a entrega. O bind dos servidores e do mock usou 127.0.0.1.
A prévia usou o binário normal, o mestre `TUNETRAIL_INGEST_ACTIVE=true` e
`active=1, occurrence_write=1` nos dois perfis. A flag Revisar ficou ligada
no piloto e desligada no segundo perfil. O escritor e o índice legados, o
interruptor e as flags continuam no produto.

O script oferece requisições individuais a 15 escutas/s no piloto, alternando
três chaves sintéticas e três nomes de cliente. Usa 12 threads de envio e
cronômetro monotônico. A leitura SQLite a cada 50 ms mede o ACK até a primeira
visibilidade no histórico persistido, com consultas HTTP separadas para
validar a exposição pela aplicação. A carga tem três classes de metadados
com títulos sintéticos, sem álbum e sem artista nem álbum.

## Carga principal

| Medida | Ocorrência | Limiar |
| --- | ---: | ---: |
| Ofertadas no piloto | 2.250 a 15/s por 150 s | 15/s |
| ACK p95 | 6,84 ms | < 200 ms |
| ACK até histórico p95, entre materializadas | 0,972 s | < 10 s |
| Maior ACK até histórico, entre materializadas | 1,034 s | < 10 s |
| ACK sem escuta materializada | 205 de 2.045 (10,02%), todos `release_missing` | visibilidade integral pendente |
| HTTP 503 | 0 de 2.250 (0%) | < 0,1% |

A vazão de 15/s descreve a taxa oferecida. A média de ACKs HTTP 200 foi
13,63/s e a de escutas materializadas foi 12,27/s, usando os 150 s de envio.
As duas taxas refletem a mistura de metadados especificada. Em uma segunda
rodada, com metadados completos em todas as 900 escutas, o piloto recebeu
15,02 requisições/s em um minuto. As 900 retornaram 200 e foram materializadas.
O ACK p95 foi 8,62 ms, a visibilidade p95 foi 0,967 s, o máximo foi 1,028 s
e a taxa de 503 foi 0%.

## Comparação com o escritor legado

| Medida no mesmo lote | Ocorrência | Legado |
| --- | ---: | ---: |
| HTTP 200 / 400 / 503 | 2.045 / 205 / 0 | 2.045 / 205 / 0 |
| Escutas novas no piloto | 1.840 | 2.045 |
| Total do piloto, incluindo histórico | 1.920 | 2.125 |
| Total do outro perfil | 0 | 0 |
| ACK p95 | 6,84 ms | 4,35 ms |
| Visibilidade p95 entre materializadas | 972,27 ms | 55,34 ms |

A paridade integral **falhou**. O legado gravou 205 escutas sem álbum que a
ocorrência conservou em `pending_identity / release_missing`. As 1.840
escutas compartilhadas coincidiram na projeção de perfil, segundo e cliente.
Nenhuma apareceu somente no caminho por ocorrência. Os pares do legado que
faltaram na projeção ativa têm exatamente os 205 recibos pendentes esperados.
As 80 linhas históricas conservaram perfil, ID de evento, segundo, ocorrência,
cliente e origem nas duas cópias. A cópia ativa passou em `quick_check` e
`foreign_key_check`.

| SHA-256 | Piloto |
| --- | --- |
| IDs antes, ambos os modos | `2efa49aca49f2038ab2ff12e289797bedd987138781abb531cec560792ae1c00` |
| IDs após carga, ocorrência | `2f750b093431c17daa041dadb6ef807b83c8fd5231e031a5fc82c2c846d75b3e` |
| IDs após carga, legado | `a0d8e11aae839855df670f84a775141e6fb253363d5c68d4a3501e574aa23807` |
| Projeção da carga, ocorrência | `c14b5e035a094b63ba21075860a1742bb647d1a88954a30959ea99d5890ae8c9` |
| Projeção da carga, legado | `a8bf23ca72d9c8db1b88b6eb4c4c7d5a77dd8fff0e405fa537bb9a6f26c421a7` |

Os hashes de IDs são internos a cada cópia. A comparação entre modos usa a
projeção lógica, sem depender de IDs alocados independentemente.

## Reenvio, importação, revisão e falhas

- Seis envios adicionais receberam HTTP 200. Em cada perfil, o teste
  reenviou o mesmo corpo e chave e enviou o mesmo segundo sem chave.
  O piloto produziu uma pendência `legacy_identity_collision` com Revisar
  ligado. O outro perfil registrou `duplicate / review_deferred` com Revisar
  desligado.
- A importação Last.fm enviou duas linhas iguais por perfil. As quatro
  linhas materializaram quatro escutas, duas por perfil, com origem
  `lastfm_import`. Essa observação indica que, neste formato e cenário, a
  repetição de linhas dentro do mesmo arquivo conserva as duas posições.
- O adaptador HTTP não preenche o ID externo confiável. Os testes existentes
  `TestOccurrenceTrustedReplay` (Revisar ligado e desligado),
  `TestIngestConcurrentTrustedReceipt` e `TestIngestDiscardTrustedReplay`
  passaram no armazenamento SQLite com fixtures sintéticas.
- Um trigger restrito a um recibo sintético abortou a revisão no resolvedor
  após o ACK 200. O ensaio adiantou somente o relógio de reagendamento desse
  recibo para exercitar as cinco tentativas. O worker estacionou o recibo em
  `retryable / retry_exhausted`. O trigger foi retirado para que o recibo
  possa ser reprocessado pela interface.
- O mock respondeu HTTP 429 com `Retry-After`. Foi observada pausa
  `rate_limited` e retomada com entregas confirmadas. As 1.842 linhas de
  Outbox observadas depois dos casos de borda ficaram em `sent`.
  As importações ficaram fora do relay. A rodada adicional de metadados
  completos levou o Outbox a 2.742 entregas confirmadas.

## Suítes

| Execução | Resultado |
| --- | --- |
| `TUNETRAIL_INGEST_ACTIVE=true make -j3 check`, binário e fixtures normais | Passou, inclusive Go completo e 13/13 percursos E2E |
| E2E com mestre ligado por padrão no binário de teste | 13/13 |
| `make check` com overlay de ocorrência por padrão | Falhou em testes que dependem das semânticas de fixture de outros modos |
| E2E explícito com overlay de ocorrência por padrão | 7/13 |
| `make api.race` com mestre ligado | Timeout de 120 s em `engine` e `internal/db/sqlite` |
| Go completo com `-race`, assets e idiomas configurados, prazo de 8 min | Mesmos dois pacotes atingiram o timeout. Os demais passaram |
| `-race` focado em ID confiável, descarte, ocorrência, pausa e retomada do relay | Passou em 9,48 s |

O primeiro `go test -race` foi lançado sem os diretórios de assets e idiomas
que o Makefile exporta. Em `cmd/api`, `TestTOMLStartupAndReset` e
`TestSettingsHTTPJourney` falharam na inicialização. Em `engine` o servidor
de teste não ficou acessível. Em `internal/adminui`,
`TestExpiredCatalogReviewCannotExecute`, `TestLocalizedRateLimitKeepsHTMLAndStatus`,
`TestSettingsDurabilityWarning`, `TestSettingsPreservesInvalidDraftWhenStateUnavailable`,
`TestLogoutSubmissionRenewsExpiry`, `TestAccountTokenWeb162` e
`TestAdministrationJourney` registraram ausência dos assets.
O alvo `api.race` restaurou esses diretórios e atingiu seu limite nos testes
`TestMediaInvalidReturnTimezone140` e `TestAccountProfileSlugCollisions/false`.
O diagnóstico do timeout não atribui a falha aos testes em execução no
instante do alarme. A repetição com 8 min atingiu o limite durante
`TestLoginSetsOnlyCurrentCookie` em `engine` e
`TestIngestBetweenReviewAndConfirmKeepsBothProfilesAndRejectsStalePlan`
em SQLite, sem diagnóstico de data race. A execução inicial redundante foi
encerrada depois de registrar suas falhas de configuração.

O overlay é exclusivo dos testes em bases descartáveis. Ele liga a ocorrência
para os perfis migrados e recém-criados, preservando valores explícitos do
interruptor mestre. As fixtures que escrevem diretamente por meios legados
recebem a proteção `Legacy listen conflicts with occurrence writing`.
As que inserem flags manualmente colidem com a PK de `profile_ingest_flags`.
As que migram com recibos ou sombra ainda abertos encontram o guardião de
ativação. Houve 291 falhas de nível superior no Go, incluindo a verificação de
referência visual. O catálogo detalhado da execução Go fica em
`check-default-active.log` no diretório privado do ensaio.

| E2E que falhou com ocorrência por padrão | Primeira divergência |
| --- | --- |
| multiuser | Total de escutas consultado logo após admissão, com expectativa síncrona de 2 |
| public | Consulta inicial com expectativa síncrona de 152 escutas |
| isolation | Consulta inicial com expectativa síncrona de 4 escutas no perfil A |
| admin | Salvamento de configuração de relay retorna 409, esperado 303 |
| migration | O percurso administrativo após a migração chega ao mesmo 409 |
| shutdown | PUT de configuração de relay retorna 409, esperado 204 |

Os 409 do E2E de relay vieram da validação de destino que passa a ser
obrigatória para perfis já convertidos. Os percursos usam destinos de fixture
que o setup anterior à conversão conseguia salvar. As expectativas de
visibilidade imediata também precisam distinguir admissão de resolução.
A execução Go ainda registrou efeitos em cascata nos testes que dependiam de
catálogo populado assincronamente ou de contagens de recibos de sombra.
As causas e a primeira asserção de cada teste estão no log privado.

## Reprodução

Com o snapshot já gerado pelo SQLite e um diretório de rodada inexistente:

```sh
make api.build
ensaio="$HOME/Documents/TuneTrail-ensaio-172"
python3 scripts/rehearse_issue172.py ./tunetrail "$ensaio/original.db" "$ensaio/repeticao"
```

Os comandos adicionais a seguir usam uma segunda sessão após a carga
principal. Cada cenário de carga ou falha deve ser executado uma vez por cópia.

```sh
python3 scripts/rehearse_issue172.py --permanent-failure "$ensaio/repeticao/occurrence"
python3 scripts/rehearse_issue172.py --complete-pilot "$ensaio/repeticao/occurrence"
python3 scripts/rehearse_issue172_test_overlay.py "$ensaio/overlay-repeticao" --occurrence
GOFLAGS="-overlay=$ensaio/overlay-repeticao/overlay.json" TUNETRAIL_INGEST_ACTIVE=true make -j3 check
```

## Conferência local

A versão fica no terminal **TuneTrail ensaio 172** do Orca, em
`http://127.0.0.1:4120`. Login `ensaio-172`. A senha gerada fica em
`~/Documents/TuneTrail-ensaio-172/run-2/occurrence/login.txt`, com acesso
restrito ao usuário local. Foram verificados por HTTP o login, a Central de Saúde,
o histórico, os rankings e a abertura da exportação nos dois perfis.

1. Abra `/admin/health?profile=1` e confira o Inbox, as entregas confirmadas
   do Outbox, as pendências de identidade e a pendência de mesma escuta.
2. Em Falhas permanentes, confira a tentativa 5 e o motivo de esgotamento.
   Reprocessar usa o resolvedor sem a falha artificial.
3. Compare a Central de Saúde do segundo perfil, onde Revisar está desligado.
4. Volte ao perfil pelo seletor e consulte Escutas e Rankings.
5. Na administração, exporte cada perfil. A exportação contém o histórico
   da cópia e deve permanecer local.

O principal risco para o corte é a mudança de comportamento para escutas sem
álbum. As expectativas síncronas e de configuração de relay das fixtures
também precisam ser revistas para a fase 2. Seguem abertas as decisões sobre
o padrão de Revisar e a política de reagendamento do timeout do MusicBrainz.
O ensaio usou MusicBrainz desligado.

## Inventário das falhas Go com ocorrência por padrão

As entradas são os testes de nível superior. O log privado conserva
as subasserções e os testes filhos quando há mais de uma divergência.

<details>
<summary>Fixture gravou pelo escritor legado em perfil convertido (212)</summary>

- `TestAccountTokenTransaction162`
- `TestActiveKoitoValidatesBeforeCatalog`
- `TestAdminMergeAPIContracts`
- `TestAdminSessionRejectsOrdinaryAccount159`
- `TestAlbumCreditsSurviveTrackEdits`
- `TestAliasDeleteRejectsPrimaryAndMissing`
- `TestArtistAlbumOnlyRelationIncludesAffectedListeners`
- `TestArtistAlbumsFollowTrackEdits`
- `TestArtworkDropContract142`
- `TestBoundedExportUnderConcurrentIngestion`
- `TestCatalogMergeBothDirectionsPerProfile`
- `TestCatalogPlanStaleAndScopedMerge`
- `TestCatalogReviewStaleAndOneUse`
- `TestDeleteListenRequiresExactProfileAndRecord`
- `TestDeleteTrackPreservesUnrelatedUnplayedMedia`
- `TestDocumentAssetMetadata`
- `TestExpiredIngestLeaseCanBeReclaimed`
- `TestFillTrackIdentityConcurrent`
- `TestFillTrackIdentityRollback`
- `TestGraphFailuresPreserveHeadersWithoutFakeZeros`
- `TestHTMLManagementMatrix160`
- `TestHistoryHTMLAndJSONShareCivilMonthBounds`
- `TestHomeAndMediaGraphsUseOnlyTheirOwnFiltersAndCounts`
- `TestHomeHistoricalTotalsAndActivityTooltips`
- `TestHomeKPIComparisonTranslations`
- `TestHomeKPIsPeriodsAndEmptyBaseline`
- `TestHomeNowPlayingAppearsBeforePeriodSummary`
- `TestIngestAdmissionModeReplay`
- `TestIngestAttemptDeadline`
- `TestIngestAttemptShutdown`
- `TestIngestBetweenDeleteReviewAndConfirmRejectsStalePlan`
- `TestIngestBetweenReviewAndConfirmKeepsBothProfilesAndRejectsStalePlan`
- `TestIngestDigestUpgrade`
- `TestIngestDiscardBatch`
- `TestIngestDiscardProtectsListens`
- `TestIngestDiscardRetention`
- `TestIngestDiscardScale`
- `TestIngestDiscardSetDrift`
- `TestIngestDiscardStates`
- `TestIngestDiscardTrustedReplay`
- `TestIngestDiscardWriterRace`
- `TestIngestLeaseClaimCap`
- `TestIngestReclaimsShadowBatch`
- `TestIngestReprocessAllLargeQueue`
- `TestIngestReprocessAllWithResolver`
- `TestIngestReprocessHistory`
- `TestIngestRetryBudgetAndRecovery`
- `TestIngestRetryRejectsInvalidInput`
- `TestIngestReviewBatch`
- `TestIngestReviewBatchRace`
- `TestIngestReviewDecisions`
- `TestIngestReviewRace`
- `TestIngestedCreditsAndAlbumMergeFollowTracks`
- `TestIngestionAuthorizationBoundaryAfterRevocation`
- `TestInstantSearchFragment`
- `TestInterestAPIValidationAndHTMLParity`
- `TestJSONManagementMatrix160`
- `TestListenDialog141`
- `TestListenDisclosure073`
- `TestListenHistoryControls141`
- `TestListenMediaReturn141`
- `TestListenSecurity141`
- `TestListeningVisibility161`
- `TestLiveErrorsDoNotBecomeFakeZerosOr404`
- `TestLivePagesUseScopedSQLiteAndAPIValues`
- `TestLivePagingFiltersAndSameSecond`
- `TestLoginReturn163`
- `TestManagementCatalogDestinations139`
- `TestManagementHistoryMediaReturn139`
- `TestManagementLegacyListenDeleteReturn139`
- `TestManagementListenErrorReturn139`
- `TestManagementTaskFragments139`
- `TestManagementTaskLogin139`
- `TestManagementTaskMutationSecurity139`
- `TestManagementTaskPages139`
- `TestManagementUnplayedSurvivor139`
- `TestMassAssignmentAndCurrentRole160`
- `TestMediaActionVisibility140`
- `TestMediaDialogSecurity140`
- `TestMediaEssentialAndOptionalFailures`
- `TestMediaFiltersAndIDsCannotReplaceRouteIdentity`
- `TestMediaHeadersMatchAPIAndKeepProfileScope`
- `TestMediaInvalidReturnTimezone140`
- `TestMediaLimitsAndCompleteAlbumLink`
- `TestMediaLiveSearch140`
- `TestMediaNoFirstListenAndTiedRanks`
- `TestMediaNowPlayingMatchesEntityAndProfile`
- `TestMediaOptionalTimezone140`
- `TestMediaRelationsImagesAndExternalLinks`
- `TestMediaSectionLinks073`
- `TestMediaTaskDialog140`
- `TestMergeIdentitiesAndRollback`
- `TestMergeTracksPreservesProfilesAtSameTimestamp`
- `TestMergesPreserveUnrelatedAndUnplayedSurvivors`
- `TestMoreGroupsReplaceSecondSidebar`
- `TestNavigationByPermission163`
- `TestNavigationLabelsBelongToThemePanel`
- `TestNoScriptNavigationExposesProfileAndScopedSearchLinks`
- `TestNowPlayingIsScopedAndMissingTrackClearsOnlyItsProfile`
- `TestOccurrenceAdmissionSQLiteFull`
- `TestOccurrenceAliasCredits`
- `TestOccurrenceArchiveLocalNamespaceScope`
- `TestOccurrenceArchiveRoundTrip`
- `TestOccurrenceCutRequiresClearedShadow`
- `TestOccurrenceDiscardPending`
- `TestOccurrenceDistinctSameSecond`
- `TestOccurrenceExportPlan`
- `TestOccurrenceExternalRecordingClaim`
- `TestOccurrenceHTTPCompatibility183`
- `TestOccurrenceHistoryDeleteWithUnresolvedLease`
- `TestOccurrenceImportMasterOffReadsUnderContention`
- `TestOccurrenceLegacyUnpositionedCharacterization`
- `TestOccurrenceLinkedReleaseGuard`
- `TestOccurrenceMergeRequiresRevision`
- `TestOccurrenceMergeSync`
- `TestOccurrenceOriginGuard`
- `TestOccurrenceProofRedirects`
- `TestOccurrenceProvenTrackDisambiguatesRelease`
- `TestOccurrenceProvisionalPromotion`
- `TestOccurrenceReconciliationRevisesActiveReceipts`
- `TestOccurrenceReimportAfterDeletion`
- `TestOccurrenceResolverAB`
- `TestOccurrenceResolverCut`
- `TestOccurrenceResolverIntro`
- `TestOccurrenceResolverOverlap`
- `TestOccurrenceResolverRecovery`
- `TestOccurrenceResolverStandalone`
- `TestOccurrenceReviewSetting`
- `TestOccurrenceTrustedReplay`
- `TestOccurrenceWriteConflicts`
- `TestOutboxDetailRetention`
- `TestParticipationDoesNotDeleteLastAlbumOrHistory`
- `TestParticipationRollback`
- `TestPlanAccountsForArtistAlbumTrigger`
- `TestPlanRollbackAndIntegrity`
- `TestPreparedUploadsCountAndScope`
- `TestPrimaryArtistRequiresRelation`
- `TestProfileDeleteClosesV9Ingest`
- `TestProfileDeleteDerivedAlbumCredits143`
- `TestProfileDeleteExactText143`
- `TestProfileDeleteNames143`
- `TestProfileDeleteOtherProfileChange143`
- `TestProfileDeleteRemovesResolvedOccurrencePending`
- `TestProfileDeleteSecurity143`
- `TestProfileDeleteSpentTicket143`
- `TestProfileExportPaginationDoesNotSkipSameTimestamp`
- `TestPublicCatalogFallbackAndEscaping`
- `TestPublicTranslationsKeepMediaAndProtocolStable`
- `TestRecordingMergeLargeFixture`
- `TestRecordingUndoRetention`
- `TestRelayBatchWritesUseIDs`
- `TestRelayCapacity`
- `TestRelayConfigurationTransitions`
- `TestRelayCounterCrash`
- `TestRelayFullCompletionKeepsClaim`
- `TestRelayGenerationFencesClaim`
- `TestRelayGrantRootFence`
- `TestRelayHistoryDeletion`
- `TestRelayInvalidSourceRetiresItem`
- `TestRelayInvalidStoredPermissionsPausesDestination`
- `TestRelayLeaseQueryPlans`
- `TestRelayOccurrenceIntent`
- `TestRelayOutboxClaims`
- `TestRelayPauseAndCancellation`
- `TestRelayQueueStatusAbsent`
- `TestRelaySourceRevisionAndMerge`
- `TestRelaySuccessWithoutRateLimitDeadline`
- `TestRelayTransientPauses`
- `TestRemoveArtistDoesNotPurgeUnplayedCatalog`
- `TestRenderedGraphTablesMatchIndependentProfileCounts`
- `TestResolveIngestLegacyCollision`
- `TestResponsiveNavigationDestinations`
- `TestReviewedLegacyRepairGuards`
- `TestRewindFirstListenError`
- `TestRewindLimitsRankingsToFive`
- `TestRewindSQLiteAndSummaryAPI`
- `TestRoutingSearchAndLegacy`
- `TestSavedTrackDoesNotCreateIndependentAlbumCredit`
- `TestShadowDeletePaths`
- `TestSignOutConfirmation157`
- `TestSignOutLinksKeepReadOrigin157`
- `TestSignOutSubmission157`
- `TestSpeculation075PublicDocumentsOnly`
- `TestSpeculation075SessionRace`
- `TestSubmitListen_AssociateAllMbzIDs`
- `TestSubmitListen_AssociateAllMbzIDsWithMbzUnreachable`
- `TestSubmitListen_AssociateReleaseAliases`
- `TestSubmitListen_CreateAllMbzIDs`
- `TestSubmitListen_CreateAllMbzIDsNoReleaseGroupID`
- `TestSubmitListen_CreateAllNoMbzIDs`
- `TestSubmitListen_CreateAllNoMbzIDsNoArtistNamesNoReleaseTitle`
- `TestSubmitListen_MatchAllMbzIDs`
- `TestSubmitListen_MatchAllMbzIDsNoReleaseGroupIDNoTrackID`
- `TestSubmitListen_MatchFromTrackTitleNoMbzIDs`
- `TestSubmitListen_MatchNoMbzIDs`
- `TestSubmitListen_MatchOneArtistMbzIDOneArtistName`
- `TestSubmitListen_MatchTrackFromMbzTitle`
- `TestSubmitListen_MusicBrainzUnreachable`
- `TestSubmitListen_MusicBrainzUnreachableMBIDMappings`
- `TestSubmitListen_UpdateTrackDuration`
- `TestSubmitListen_UpdateTrackDurationWithMbz`
- `TestSubmitListen_VariousArtistsRelease`
- `TestTrackAlbumParticipation`
- `TestTrackAliasIdentity`
- `TestTrackArtistUpdateUsesOneTransaction`
- `TestTrackChartGrouping`
- `TestTrackRankingsByIdentity`
- `TestTranslatedRewindAndPartialFailures`
- `TestUIReference`
- `TestUserHTMLLoginAndLogout159`
- `TestUserManagementRoutesStayForbidden159`
- `TestWALSnapshotRaceMapsToConflict`

</details>

<details>
<summary>Fixture de isolamento ou rollback recebeu conflito de alteração de dados (2)</summary>

- `TestConsolidatedProfileMigrationRollsBackLateFailure`
- `TestIngestHealthIsolation`

</details>

<details>
<summary>Fixture inseriu uma flag que o overlay já criou (18)</summary>

- `TestActiveIngestACK`
- `TestActiveIngestBusy`
- `TestActiveIngestImports`
- `TestDurableRelayAdmission`
- `TestIngestHealthDiscard`
- `TestIngestHealthReprocess`
- `TestIngestHealthReviewActions`
- `TestListenAdmissionErrorClasses`
- `TestMappedArtistMBIDResolution170`
- `TestOccurrenceHTTPFixture`
- `TestOccurrenceHTTPZeroTimestamp`
- `TestOccurrenceLastFM`
- `TestOccurrenceListenBrainzZIP`
- `TestOccurrenceMaloja`
- `TestOccurrenceNative`
- `TestOccurrenceSpotify`
- `TestOccurrenceZIPZeroTimestamp`
- `TestSlowRelayDoesNotBlockOtherProfile`

</details>

<details>
<summary>HTML de estado da importação divergiu do esperado pela fixture síncrona (1)</summary>

- `TestAdminUIImportExport`

</details>

<details>
<summary>Formulário de catálogo retornou 409 com a fixture de ingestão convertida (2)</summary>

- `TestMediaSectionsWithoutReload`
- `TestPreparedMediaFormsAndProfileContext`

</details>

<details>
<summary>Importação ou leitura síncrona esperou uma entidade, proveniência ou escuta ainda ausente (18)</summary>

- `TestImportKoito`
- `TestImportLastFM`
- `TestImportLastFM_MbzDisabled`
- `TestImportListenBrainz`
- `TestImportListenBrainz_MBIDMapping`
- `TestImportListenBrainz_MbzDisabled`
- `TestImportMaloja`
- `TestImportSpotify`
- `TestImportTruncatedFileStoresNothing`
- `TestListenBrainzImportAcceptsDecimalTimestamp`
- `TestOwnerHTTPImportProvenance160`
- `TestOwnerImportMetadata160`
- `TestProfileBackendAdminAndSlugChanges`
- `TestProfileBackendIsolationAndIngestion`
- `TestProfileImportAndRelay`
- `TestProfileImportUpload`
- `TestProfileManagementLifecycle`
- `TestUploadArchiveFailureReportsPartial`

</details>

<details>
<summary>Fixture de falha parcial ou erro síncrono esperava o hook do escritor anterior (4)</summary>

- `TestImportCancellationAfterFirstSaveReportsPartial`
- `TestImportDiskFailureKeepsSavedListensAndResumes`
- `TestImportShutdownMidwayResumesWithoutDuplicates`
- `TestListenBrainzResponsesDeclareJSON`

</details>

<details>
<summary>Fixture esperava um recibo de sombra e recebeu zero (2)</summary>

- `TestShadowFileResume`
- `TestShadowReceiptACK`

</details>

<details>
<summary>Configuração de relay em perfil convertido retornou 409 (3)</summary>

- `TestAdministrationJourney`
- `TestProfileRelayAdministration`
- `TestProtectedIngestionAndRelay161`

</details>

<details>
<summary>Entidade ou agregado esperado pelo teste ainda não estava no catálogo (11)</summary>

- `TestAlbumReplaceImage`
- `TestAliasesAndSearch`
- `TestArtistReplaceImage`
- `TestGetters`
- `TestLoginGate`
- `TestManualListen`
- `TestMerge`
- `TestMerge_ConflictingListens`
- `TestSetPrimaryArtist`
- `TestStats`
- `TestUpdateArtists_Track`

</details>

<details>
<summary>Guardião de conversão encontrou recibos, sombra ou relay incompatíveis no setup (14)</summary>

- `TestAccountsMigrationPreservesData`
- `TestArchiveBoundaryUpgrade`
- `TestExpandedV9SnapshotReplay`
- `TestIngestActivationUpgrade`
- `TestIngestDiscardMigration`
- `TestIngestReprocessUpgradeFromV17`
- `TestIngestRetryUpgradeFromV12`
- `TestOccurrenceUpgrade`
- `TestRecordingIdentityAllocator`
- `TestRecordingIdentityBackupRestore`
- `TestRecordingIdentityBridge`
- `TestRecordingIdentityCursor`
- `TestRecordingIdentityUpgrade`
- `TestRelayBootstrapWAL`

</details>

<details>
<summary>Fixture de ativação ou sombra encontrou perfil convertido com admissão pausada (3)</summary>

- `TestIngestActivationConfirmation`
- `TestShadowClientKey`
- `TestShadowRepetitions`

</details>

<details>
<summary>Fixture esperava vinculação automática de relay legado (1)</summary>

- `TestProfileKeysCannotMoveAndRelayDoesNotReassign`

</details>
