# Migração SQLite até o schema 22

Amostra real de Koito copiada com os componentes SQLite disponíveis para
uma pasta temporária fora da worktree. A primeira abertura SQLite ocorreu
na cópia. A aplicação fez o upgrade de schema 2 para 22 e passou por um
segundo start sem alterar as projeções comparadas.

| Dados preexistentes preservados | Linhas antes e depois |
| --- | ---: |
| Contas | 1 |
| Sessões | 1 |
| Chaves de API | 1 |
| Escutas | 72 |
| Artistas | 18 |
| Aliases de artistas | 18 |
| Álbuns | 19 |
| Aliases de álbuns | 21 |
| Associações artista–álbum | 27 |
| Faixas | 76 |
| Aliases de faixas | 76 |
| Associações artista–faixa | 85 |

O SHA-256 agregado das projeções legadas foi
`d5c9459507ec0e43fbb3e033f1d68ea2070948bf51a217da6ef0b1b46eb93c9a`
na cópia de entrada e no banco migrado. O algoritmo ordena as linhas de
cada projeção, calcula um digest por tabela e agrega o mapa ordenado desses
digests. As projeções incluem as credenciais e chaves somente na comparação
local por digest. A verificação confirmou a associação das escutas e das chaves
ao perfil correspondente, a relação conta–perfil 1:1, a presença de admin
ativo e a visibilidade pública inicial. `integrity_check` e
`foreign_key_check` passaram.

O ensaio sintético fez três migrações independentes v2 → v22. Uma delas
incluiu uma escuta comprometida no WAL. Após o upgrade, as 25 verificações de
rotas e as jornadas de administração e API passaram.

O validador de contas também migrou uma cópia de backup real v4 → v22.
Preservou 80 escutas, comparou as projeções de dados e encontrou uma conta
ativa e uma pendente. Os testes SQLite cobrem rollback de migração, colisões
de criação e invariantes de conta/perfil.

Reprodução sobre um arquivo v2 estável com
`python3 scripts/validate_migration.py --binary ./tunetrail --source-v2 CAMINHO`.
Para backups anteriores à migração de contas, use
`python3 scripts/validate_accounts_migration.py --binary ./tunetrail --source COPIA`.

As fixtures v3 e v4 para migração de contas passaram com dois perfis e
duas escutas, uma em cada perfil. Para a etapa v3 de ingestão foi usada
a fixture sintética, sem configuração de relay externo.
