# ISSUE-064 — Recuperação de migração, importação e restauração

**Data:** 13/11/2026 · **Versões:** TuneTrail 0.8 (atual) e 0.5 (`6b13514`, última que lê `koito.db`) · **Dados:** somente fixtures fictícias em diretórios descartáveis.

## Migração `koito.db` → `tunetrail.db`

`internal/db/sqlite/bootstrap.go` ganhou o ponto de teste `bootstrapPhase`, nulo em produção, em cada fronteira da migração: `locked`, `backup-written`, `backup-synced`, `temp-copied`, `temp-migrated`, `marker-written`, `temp-synced`, `published` e, na migração local de um `tunetrail.db` v2, `inplace-backup-synced` e `inplace-migrated`.

A fixture (`legacyFixture`) é um `koito.db` v2 com artista, álbum, faixa, conta de administrador, chave de API e 200 escutas, **todas pendentes no WAL** durante a migração. Depois de cada recuperação, `requireRecovered` compara o hash de cada coluna legada de cada tabela com o original, confere esquema v3, integridade, chaves estrangeiras, o backup indicado no marcador e que `koito.db` não mudou.

| Teste | Modelo de falha | Resultado |
| --- | --- | --- |
| `TestBootstrapRecoversFromInjectedFailureAtEveryPhase` | erro `ENOSPC` injetado em cada uma das 8 fases (disco cheio); `defer` executa | falha sem publicar `tunetrail.db` (exceto após `published`, quando o banco já está completo); temporário removido; nova partida conclui |
| `TestBootstrapRecoversFromCrashAtEveryPhase` | processo filho termina com `os.Exit` em cada fase (queda, `kill -9`); nenhum `defer` | nenhum arquivo parcial publicado; nova partida conclui com todos os dados |
| `TestBootstrapInPlaceV2RecoversAtEveryPhase` | queda antes e depois do `goose.Up` local | nova partida conclui em v3 com dados preservados |
| `TestBootstrapRejectsUnknownSchemaVersions` | `koito.db` com versão 4; `tunetrail.db` com versão 99 | recusa sem criar backup nem temporário e sem alterar o banco |
| `TestBootstrapRestoreFromBackup` | restauração: `.bak` copiado como `koito.db` em diretório vazio | migração refeita com todos os dados |
| `TestBootstrapReconcileAfterLegacyChange` | `koito.db` alterado após a migração (versão antiga rodou) | recusa; as duas formas de reconciliar do `DEPLOYMENT.md` funcionam |

Arquivos deixados por uma queda (nunca adotados; a nova partida cria outro backup):

| Queda em | Sobra |
| --- | --- |
| `locked` | nada |
| `backup-written`, `backup-synced` | um `.bak` sem marcador |
| `temp-copied` … `temp-synced` | um `.bak` sem marcador e um `.tunetrail-migrate-*.db` |
| `published` | o `.tunetrail-migrate-*.db` é um hard link do próprio `tunetrail.db` publicado; remover o link é seguro |

Teste de mutação: publicar o temporário antes de migrá-lo e remover a limpeza do temporário fazem os testes acima falharem.

## Importação

O contrato continua não transacional: cada escuta é gravada isoladamente, e reimportar o mesmo arquivo conta duplicatas em vez de repetir escutas.

| Teste | Resultado |
| --- | --- |
| `TestImportDiskFailureKeepsSavedListensAndResumes` | falha `ENOSPC` após 7 de 20 escutas: as 7 ficam; o arquivo temporário é removido; reimportar grava 13 e conta 7 duplicatas; uma terceira importação não grava nada |
| `TestImportShutdownMidwayResumesWithoutDuplicates` | encerramento após 12 de 30: as 12 ficam; reimportar grava 18 e conta 12 duplicatas |
| `TestImportTruncatedFileStoresNothing` | arquivo cortado em 4 pontos: recusado como inválido antes de gravar qualquer escuta; o arquivo completo importa depois |
| `TestImportCancellationAfterFirstSaveReportsPartial` (existente, ISSUE-020) | cancelamento após a primeira escuta informa importação parcial |

## Ensaio de restauração e rollback

`scripts/validate_rollback.py --binary ./tunetrail --previous-binary <0.5>` com a 0.5 compilada por `git archive 6b13514`:

1. A 0.8 migra o `koito.db` legado; uma escuta é gravada depois da migração.
2. A 0.5 no mesmo diretório lê o `koito.db` intacto: 1 escuta (a posterior à atualização não aparece) e a chave legada continua válida.
3. A 0.8, de volta ao mesmo diretório, **recusa** iniciar: "legacy database changed after migration", porque a 0.5 atualiza o `koito.db` no lugar. Reconciliação manual documentada.
4. A 0.5 a partir do `.bak` copiado como `koito.db` em diretório novo: 1 escuta, chave válida.
5. A 0.8 a partir do mesmo `.bak`: migra de novo, integridade ok.

`DEPLOYMENT.md` ganhou a seção "Recover an interrupted or refused migration" (passo a passo) e a seção "Rollback" foi reescrita com esses resultados.

## Contratos mantidos e lacunas restantes

Mantidos por decisão (não são falhas):

- Importação parcial: o que foi gravado antes de uma falha permanece; a recuperação é reimportar o mesmo arquivo.
- Exportação dinâmica: reflete o banco no momento da leitura, sem snapshot.
- Migração sem downgrade SQL; rollback por cópia de segurança.

Lacunas registradas, para aceite do usuário:

| Lacuna | Situação |
| --- | --- |
| Falta de espaço real no sistema de arquivos | Simulada por injeção de `ENOSPC` em cada fase; não houve teste com disco realmente cheio (exigiria volume dedicado). Limitação aceita. |
| Volumes de rede (NFS/SMB) | Não testados; a migração exige lock e hard link e recusa o que não suporta. Limitação documentada desde a ISSUE-046. |
| Interrupção no meio de um comando SQL do `goose.Up` | Coberta pela atomicidade das transações do SQLite; os testes interrompem antes e depois do `goose.Up`, não dentro dele. |
| Arquivos que sobram de uma queda | Não são removidos automaticamente; o `DEPLOYMENT.md` explica como identificá-los e removê-los. Pode virar tarefa pequena de limpeza automática sob o lock. |
| Rollback fora do CI | `validate_rollback.py` exige compilar a 0.5; fica como ferramenta manual. |
| Permissões de volume e container sem root | Fora do escopo; ISSUE-102. |
| Backup e restauração pela interface | ISSUE-103. |
