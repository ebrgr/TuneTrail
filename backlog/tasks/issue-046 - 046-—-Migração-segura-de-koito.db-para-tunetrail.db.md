---
id: ISSUE-046
title: 046 — Migração segura de koito.db para tunetrail.db
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - banco
  - operacao
milestone: m-5
dependencies: []
type: feature
ordinal: 112000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/046-migracao-banco-tunetrail-solved.md` — numeração local 046. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Fechada e aprovada pelo usuário em 08/11/2026 · **Tipo:** compatibilidade e migração de dados · **Revisão:** Advisor

## Objetivo

Na inicialização, usar `tunetrail.db` no diretório de dados configurado. Quando houver um `koito.db` legado, gerar um backup consistente e migrar seus dados para o novo nome. Se o esquema suportado estiver em v2, atualizá-lo para a versão consolidada v3. Se não houver banco, criar `tunetrail.db` diretamente com o esquema v3.

O aplicativo usa `tunetrail.db` e migra o arquivo legado conforme o contrato abaixo. O número v3 é a versão do esquema da aplicação, não da biblioteca SQLite.

## Comportamento atual

- `internal/db/sqlite/sqlite.go` abre `<KOITO_CONFIG_DIR>/koito.db`, configura WAL e chama o Goose.
- `internal/db/sqlite/backup.go` usa `VACUUM INTO` antes da migração de perfis, quando aplicável. Ele não cria backup de bancos que já estão em v3.
- `db/migrations_sqlite/003_listening_profiles.sql` contém a migração consolidada v2 → v3. As migrações 001–003 atendem instalações novas.
- `engine.connectDB` repete falhas indefinidamente a cada cinco segundos; o fluxo de migração precisa distinguir erros permanentes para não ocultar uma falha nem inicializar uma instalação vazia. O comando `reset-password` também utiliza o armazenamento SQLite.

## Contrato de inicialização

| Estado encontrado | Resultado esperado |
| --- | --- |
| Nenhum banco | Criar `tunetrail.db` com as migrações até a v3; não criar `koito.db`. |
| Somente `koito.db` v2 suportado | Criar backup consistente; preparar uma cópia, aplicar 003 nessa cópia, validar e publicar `tunetrail.db`. |
| Somente `koito.db` v3 consolidado | Criar backup consistente e publicar cópia validada como `tunetrail.db`; não reaplicar migração 003. |
| Somente `tunetrail.db` | Validar e abrir; se for v2 suportado, fazer backup antes de migrar para v3. Não substituir seus dados. |
| Ambos presentes | Nunca sobrescrever nem mesclar automaticamente. Reabrir o destino em reinícios somente quando houver evidência durável e verificável de uma migração concluída; caso exista conflito genuíno ou dúvida sobre a origem, interromper com instrução clara. |

Definir e testar um mecanismo durável de reconhecimento da migração concluída, de modo que manter `koito.db` intacto não impeça reinícios normais e que dois bancos independentes não sejam escolhidos silenciosamente. Arquivos temporários ou incompletos não podem ser tratados como destino válido. Não escolher apenas pelo nome ou data de modificação.

## Segurança e preservação

1. Exigir que o processo/contêiner anterior esteja parado antes da migração. Um snapshot consistente inclui transações confirmadas no WAL, mas não impede que um processo antigo continue gravando no banco anterior depois do snapshot. Um lock criado apenas pela versão nova não controla processos antigos; documentar essa limitação e não prometer detecção universal.
2. Usar mecanismo de backup SQLite consistente que inclua WAL confirmado; não copiar nem renomear apenas o arquivo principal com WAL ativo. O backup deve ter nome identificável, permissões restritas (como `0600`), não sobrescrever backups existentes e ser validado e restaurável. Preservar banco de origem, WAL e backup em falhas.
3. Preparar o destino em arquivo temporário no mesmo diretório. Validar versão e formato reais, migrar a cópia quando necessário, verificar `integrity_check` e `foreign_key_check`, fechar conexões e tratar checkpoint e arquivos auxiliares antes de publicá-lo. Garantir publicação atômica sem sobrescrever um destino existente, incluindo sincronização e limpeza conforme o sistema suportado.
4. Coordenar inicializações simultâneas e testar concorrência/TOCTOU. Verificar links simbólicos, arquivos inválidos, acesso negado, espaço insuficiente, erro de leitura/escrita e volume de rede; se a garantia necessária não puder ser obtida, falhar com orientação, sem criar banco vazio ou modificar o original.
5. Nunca alterar manualmente `goose_db_version`. Recusar versões futuras ou layouts v3/v4 experimentais incompatíveis com a v3 consolidada, conforme [banco de dados](<../docs/design/doc-018 - Banco-de-dados-—-esquema-SQLite-consolidado-v3.md>); orientar recuperação por backup v2. PostgreSQL não faz parte desta migração.
6. Aplicar a mesma seleção de banco ao comando `reset-password`; não permitir que ele crie um novo banco ou altere o banco errado após falha ou conflito.

## Trabalho

- Implementar a seleção e migração antes de abrir o armazenamento para operações normais; reaproveitar a lógica de backup e migrações existentes quando seu contrato for adequado.
- Decidir e documentar o protocolo de lock, marcador de conclusão, nomes de backup, recuperação de interrupção e conflitos. Distinguir erro permanente de falha transitória no ciclo de inicialização.
- Atualizar testes Go, `scripts/validate_multiuser.py` usado no CI e outros validadores/documentos que suponham `koito.db`. Atualizar Docker/Unraid, backup, restauração e rollback; manter o caminho antigo identificado como legado onde necessário.
- Adicionar nota ao changelog e registrar evidências sem dados ou credenciais reais. Não remover o banco legado ou backups automaticamente.

## Critérios de aceite

- [x] Banco v2 legado migra para `tunetrail.db` v3; backup anterior preserva os dados e restaura corretamente. *(Evidência: `validate_migration_030.py` — backup restaurado em cópia isolada idêntico à origem; `TestBootstrapLegacy`, `TestBootstrapExistingV2`.)*
- [x] Banco v3 consolidado recebe backup e é copiado sem alteração indevida; um novo diretório cria somente o banco novo em v3. *(Evidência: `TestBootstrapLegacy` v3, `TestBootstrapFresh`, `make multiuser.check` fresh.)*
- [x] Dados confirmados ainda no WAL, contas e senhas, perfis, chaves, escutas, catálogo, relay e referências de imagens sobrevivem à migração e ao reinício. *(Evidência: `validate_migration_030.py` compara cada tabela legada antes/depois, valida token legado e roda jornada com reinício; caso `api-wal` com escuta só no WAL.)*
- [ ] O destino validado é reutilizado em reinícios; conflitos entre dois bancos independentes, versões desconhecidas e arquivos corrompidos falham sem sobrescrita. *(Parcial: reinício, conflito e corrompido testados — `TestBootstrapConflicts`, `TestBootstrapCorruptLegacy`; versão desconhecida tem guarda em `bootstrap.go:124` mas nenhum teste. Lacuna na ISSUE-064.)*
- [ ] Falhas de backup, migração e publicação, interrupção em cada fase e duas inicializações simultâneas não perdem dados nem deixam um banco parcial reconhecido como concluído. *(Parcial: concorrência e arquivos parciais testados — `TestBootstrapConcurrent`, `TestBootstrapPartialFilesNeverBecomeThePublishedDatabase`; injeção de falhas de I/O e interrupção em cada fase não realizadas. Lacuna na ISSUE-064.)*
- [ ] `reset-password` usa o banco correto; backup, restauração e rollback são exercitados com fixtures descartáveis. *(Parcial: `reset-password` no banco existente (`cmd/api/config_integration_test.go`), backup e restauração em cópia isolada; rollback para a versão anterior não exercitado. Lacuna na ISSUE-064.)*
- [x] Integridade e chaves estrangeiras são verificadas; os testes Go, o workflow e `make multiuser.check` refletem o novo caminho. Testes usam dados fictícios e não acessam o banco real. *(Evidência: `integrity_check`/`foreign_key_check` em `validate_migration_030.py`; `make multiuser.check` fresh e migrated-v2.)*
- [ ] Documentação explica a parada do contêiner antigo, o volume persistente e como recuperar uma migração interrompida. *(Parcial: `DEPLOYMENT.md` cobre parada, volume e recusa conservadora; falta procedimento passo a passo de recuperação a partir do `.bak`. Lacuna na ISSUE-064.)*

## Parecer do Advisor

**Reprovado com ressalvas para implementação sem contrato.** Os riscos principais são perda de transações no WAL, gravação posterior pelo processo antigo, confusão entre dois bancos e falha parcial que cria uma instalação vazia. O contrato e os critérios acima incorporam as recomendações; a implementação e seu aceite continuam pendentes.

> *Nota de reconciliação (ISSUE-062, 13/11/2026):* este parecer é anterior à implementação. A revisão da versão corrigida e o aceite do usuário estão em "Implementação e resultados", abaixo.

## Implementação e resultados

Adicionados bootstrap serializado, snapshot do legado com WAL, backup restrito,
migração e validação em arquivo temporário, marcador transacional, publicação sem
sobrescrita e recusa de conflitos. O marcador compara uma impressão física do
legado e do WAL: checkpoint externo pode exigir reconciliação manual mesmo sem
mudança lógica. O sistema de arquivos precisa suportar lock e hard link; os
processos antigos devem ser parados. `reset-password` não cria banco ausente.

Passaram testes de criação, v2/v3, WAL, leitura do backup, reinício, conflito,
concorrência, caminho especial, symlink e arquivo temporário incompleto;
`make multiuser.check`, `scripts/validate_migration_030.py` e `make check`
também passaram. A injeção sistemática de falhas de I/O e interrupção em cada
ponto, testes em volumes de rede e validação de restauração operacional completa
não foram realizados e permanecem como limitações registradas. O usuário aprovou e encerrou a issue em 08/11/2026.

O Advisor revisou a versão corrigida e não identificou novo bloqueador no
caminho suportado: diretório confiável, filesystem local com lock/hard link e
processos antigos parados. A verificação atual confere a presença dos triggers,
não a equivalência integral de seus corpos; Lstat seguido de abertura não protege
contra manipulação externa concorrente do diretório.

## Reconciliação dos critérios (ISSUE-062, 13/11/2026)

Critérios marcados somente com evidência reexecutada nesta data: 13 testes `TestBootstrap*` (`internal/db/sqlite`), `scripts/validate_migration_030.py` (3 migrações v2, incluindo WAL; backup restaurado idêntico; tabelas legadas preservadas; jornadas de administração e API com reinício; integridade e chaves estrangeiras) e `make multiuser.check` (fresh e migrated-v2), todos aprovados com o binário 0.8. Os critérios 4, 5, 6 e 8 permanecem parciais; as lacunas seguem na ISSUE-064.
<!-- SECTION:DESCRIPTION:END -->
