# Regressão de ingestão e relay

TuneTrail 0.72, schema 22. Dados fictícios nas jornadas de ingestão e
receptores locais para as entregas de relay.

| Fluxo | Evidência executável | Resultado |
| --- | --- | --- |
| ListenBrainz por chave | `validate_multiuser.py`, `validate_isolation.py`, `TestOccurrenceHTTPFixture`, `TestOccurrenceDistinctSameSecond`, `TestOccurrenceTrustedReplay` | Submissões persistidas por perfil, isolamento de eventos concorrentes e tratamento de reenvio. Chaves revogadas deixam de aceitar novos envios. |
| Importações por ocorrência | `TestActiveIngestImports`, testes de importação ListenBrainz ZIP, Last.fm, Spotify, Maloja e formato nativo em `engine/ingestion_e3_test.go` | Formatos exercitados, com ocorrência persistida e destino conferidos. As jornadas incluem exportação e importação para outro perfil autorizado. |
| Relay em rascunho e aprovação | `TestProfileRelayAdministration`, jornadas `admin` e `isolation` | Salvar rascunho não dispara a sondagem. Teste de destino passa após aprovação e a resposta omite o token. |
| Entrega e fila | `TestDurableRelayAdmission`, `TestSlowRelayDoesNotBlockOtherProfile` e testes SQLite de outbox | Evento elegível gera uma entrega e a fila marca um envio concluído. Recibo anterior à aprovação fica fora da entrega. Destino lento não paralisa outro perfil. |
| Histórico protegido | `TestProtectedIngestionAndRelay161` | A chave continua ingerindo no perfil protegido, a leitura pública continua bloqueada e o receptor local recebe o evento encaminhado. |

A validação da migração de ingestão usa `validate_ingest_migration.py` para
subir a fixture v3 com nomes de banco canônico e legado. Em ambas as entradas,
os digests preexistentes, as contagens de escutas por perfil, a restauração e
o reinício são conferidos.

O `make check` passou com 13 de 13 jornadas e 190 arquivos de referência
da interface comparados sem diferenças. A jornada adicional no Chrome
passou nos dois modos de JavaScript. A documentação gerou 11 páginas
com `corepack yarn build`.
