# Backup automático e restauração

| Entrada | Backup automático | Restauração em outro diretório |
| --- | --- | --- |
| Cópia real v2, 72 escutas | Snapshot SQLite em schema 2, modo 0600 e digest igual ao da entrada. | Restaurado como banco Koito. O binário iniciou, migrou a schema 22 e manteve as 72 escutas. |
| Três fixtures v2 | Um backup consistente por instância, incluindo a escuta gravada no WAL da terceira fixture. | Cada backup subiu em sua própria instância e preservou as projeções legadas após migrar. |
| Backup real v4, 80 escutas | Backup pré-migração de contas em schema 4, com inventário igual ao da entrada. | Restaurado como banco canônico. Subiu em schema 22, manteve as escutas e passou nos testes de reinício. |
| Fixture v3 com dois perfis | Backups de upgrade na entrada pelo nome canônico e pelo nome legado. | Ambas as cópias restauradas subiram em schema 22 com as contagens por perfil preservadas. |

A cópia real restaurada de v2 apresentou o digest SHA-256
`d5c9459507ec0e43fbb3e033f1d68ea2070948bf51a217da6ef0b1b46eb93c9a`
para as mesmas projeções do ensaio de migração. As verificações SQLite
passaram e o endpoint de saúde respondeu após cada start de restauração.
Os servidores foram encerrados e as cópias descartadas. O ensaio restaura
a base física e inicia o binário alvo em loopback.
