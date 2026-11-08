# Banco de dados — esquema SQLite consolidado v3

Decisão confirmada em 4 de novembro de 2026.

**O Koito-SUM utiliza SQLite, com esquema consolidado na versão 3 (v3).**
Neste documento, “v3” identifica a versão do esquema da aplicação, não a versão
do software SQLite nem a versão de release do Koito-SUM.

As alterações de perfis de escuta, chaves e relay foram reunidas na migração
única `db/migrations_sqlite/003_listening_profiles.sql`:

- Bancos com esquema v2 são atualizados para v3 automaticamente ao iniciar a API.
- Instalações novas aplicam as migrações 001, 002 e 003, terminando em v3.
- A antiga divisão em migrações 003 e 004 foi substituída pela consolidação.
  Referências a esquema v4 ou atualização v2 → v4 nos documentos são registros
  históricos, anteriores à consolidação, e não descrevem o esquema vigente.

Não alterar manualmente `goose_db_version` para converter um banco experimental
antigo em v3. Conforme o procedimento documentado, bancos da antiga organização
experimental v3/v4 exigem restauração de um backup v2 antes de aplicar a migração
consolidada. Migrações futuras devem partir do esquema consolidado v3 e preservar
as migrações já aplicadas.

Para detalhes da implementação e validação, consulte a seção “Consolidação
autorizada — migração única v2 → v3” de [MULTIUSER_DESIGN.md](MULTIUSER_DESIGN.md).

## Nome e migração do arquivo

O TuneTrail usa `tunetrail.db` no diretório configurado. Ao encontrar apenas
`koito.db` v2 ou v3 consolidado, cria `koito.db.pre-tunetrail-*.bak` por
snapshot SQLite, prepara o novo banco e mantém o legado. Pare todos os processos
antigos antes da migração: um snapshot não impede escritas posteriores no legado.

Quando os dois arquivos existem, a aplicação verifica um marcador no banco novo
e uma impressão digital dos arquivos legados. Se o legado mudar, inclusive por
um checkpoint externo do WAL, a inicialização é recusada conservadoramente;
pare todas as instâncias e reconcilie os arquivos a partir de cópias e backups.
A publicação usa hard link sem sobrescrita e requer um diretório confiável e
sistema de arquivos local com suporte correto a lock e hard link; volumes de rede
não foram validados. Falhas não devem ser resolvidas apagando arquivos sem antes fazer
backup. O comando `reset-password` não cria banco vazio.
