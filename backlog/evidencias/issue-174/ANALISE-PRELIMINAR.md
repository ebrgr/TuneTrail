# ISSUE-174 — observações para implementação

Análise preliminar da worktree em 24/11/2026, registrada antes da implementação.

- A configuração Subsonic atual (`cfg.SubsonicUrl` e `cfg.SubsonicParams`) é global. O catálogo de gravações e artistas é compartilhado; portanto, uma declaração obtida de uma biblioteca não deve ser atribuída globalmente sem uma política que preserve conflitos e escopo por perfil e instância.
- `ingest.validMBIDs` remove valores inválidos das listas. Uma lista incompleta não é prova de correspondência posicional com os créditos: usar nomes e vínculos inequívocos, e preservar identificadores já recebidos.
- `finishOccurrence` verifica lease e versão sob o lock de escrita e atualizações de merge podem alterar a gravação e incrementar `resolution_version`. O resultado externo precisa revalidar recibo, perfil, ocorrência, gravação e créditos antes de qualquer aplicação; rede deve ficar fora desse lock e da fila de resolução da ingestão.
- `recordings.mbid_source` aceita apenas `legacy_declared` ou `validated`. Proveniência externa precisa de armazenamento aditivo com data e escopo, sem marcar a resposta como validada.
- A busca `search3` deve recusar resultados potencialmente truncados e correspondências ambíguas ou incompletas; o ID da música na API Subsonic não é o MBID de gravação.
- O código do [Navidrome](https://github.com/navidrome/navidrome/blob/master/server/subsonic/helpers.go) associa `musicBrainzId` da música a `MbzRecordingID`; o contrato [OpenSubsonic Child](https://opensubsonic.github.io/docs/responses/child/) exige `mediaType` para identificar a entidade. A implementação exige `mediaType=song` antes de tratar esse campo como gravação.

Esta análise registrou riscos para orientar a implementação posterior; não é um relatório de validação.
