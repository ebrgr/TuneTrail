# Verificação da ISSUE-131

Bug presente na versão publicada v0.26: a consulta usava somente tracks_with_title e QueryRow, sem considerar aliases secundários ou detectar múltiplas candidatas.

Implementação sem migração: busca exata em todos os aliases do lançamento, inclusão dos artistas solicitados sem depender da ordem ou repetição, erros distintos de ambiguidade e conflito de identidade. Os dois caminhos de título propagam erros de banco. A ingestão preenche um MBID ausente com validação e atualização na mesma transação; edições administrativas continuam permitindo substituir o ID.

- Os quatro testes TestAssociateTrackAlias, TestAssociateTrackLookupFailures, TestTrackAliasIdentity e TestFillTrackIdentityConcurrent falharam com track.go e associate_track.go anteriores à correção (log anexo).
- Testes dos pacotes internal/catalog e internal/db/sqlite passaram com a implementação.
- go test -race ./internal/catalog ./internal/db/sqlite -count=1 -timeout 120s passou, incluindo rollback por falha e tentativas concorrentes de preencher o MBID.
- make check passou: Go vet, testes Go, verificações da interface e 11 jornadas E2E.
- git diff --check passou; grafo atualizado por extração AST local.

Limitação deliberada do recorte: conflitos recusam a associação e a escuta não é salva; importação Last.fm é interrompida e a API ListenBrainz mantém seu tratamento atual (HTTP 500). Não foi criada fila durável de pendências (ISSUE-132+). Título principal, distinção de maiúsculas e modelo de gravação/lançamento preservados.
