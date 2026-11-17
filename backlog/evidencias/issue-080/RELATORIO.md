# ISSUE-080 — infraestrutura SQLite v4

Implementação direta, sem delegação. Contratos aceitos: doc-040/decision-009 e doc-039/decision-008. Plano ai-memory: `01a1eabc-2ce1-7b03-a43a-859056afb029`, `plans/issue-080-advisor-v4.md`.

## Escopo técnico

- `004_ingest_infrastructure.sql`: integrations com identidade opaca, referência de segredo, revogação, retenção 90 (7–3650) e diagnóstico off; ingest_events como ledger único, envelope/snapshots/evidência vigente, índices únicos parciais para chaves externas confiáveis, cliente/operação/ordinal e arquivo/formato/versão/ordinal; detalhes, variantes imutáveis e revisões versionadas separadas; outbox e detalhes operacionais.
- FKs compostas com perfil impedem cruzamento de identidade. Remover perfil remove recibos/variantes/detalhes. Remover integração com proveniência viva é impedido no commit; revogar mantém a identidade. Constraints diferidas permitem cascata completa do perfil.
- Guardas impedem accepted/duplicate, IDs futuros e versão de resolução >0; triggers impedem revisões e outbox. Não existe FK para evento inexistente. A 133 deverá adicionar vínculos reais, retirar guardas e ampliar os estados terminais da purga antes de ativar escritores; nenhuma versão além da v4 está reservada. Relay atual preservado, nenhuma carga na outbox.
- Purga explícita, sem scheduler: lotes de no máximo 500, transação curta com revalidação de fechamento/estado/lease/proteção e prazo por integração. Núcleo, snapshot/evidência vigentes e pendências não expiram. Na v4 o único encerramento de recibo permitido é invalid; sent/discarded da outbox só são exercitados em fixture que retira a guarda, sem ativação em produção. Variantes abertas e detalhes protegidos permanecem. Diagnóstico não possui coletor nesta entrega; a flag inicia desativada, sem armazenamento de corpo bruto.
- Bootstrap valida DDL canônico de todas as tabelas/views/índices/triggers v3/v4, além de integridade/FKs. Versão desconhecida e schema incompleto são recusados. Backup VACUUM INTO, incluindo WAL comprometido, validado e sincronizado antes de migrar tunetrail.db; fluxo legado preserva koito.db e publica cópia consolidada. SQL 001–003 intactos.
- Sem tags/datas da 055, tombstones ou chave auxiliar de reimportação. Migração interna, sem entrada no CHANGELOG conforme skill.

## Migrações em cópias privadas

O script `scripts/validate_ingest_migration.py` copia o banco e seus componentes WAL/SHM/journal para diretório temporário **antes de qualquer conexão SQLite**. As origens autorizadas foram data154 (tunetrail.db e koito.db) e demo (tunetrail.db). Nenhum banco do checkout principal foi aberto. O script verifica também hashes físicos das origens antes/depois sem gravar valores nas evidências. Dados e bancos temporários são eliminados ao terminar. `migracoes-agregadas.json` contém somente contagens e resultados.

| Caso | Entrada | Escutas por perfil antes | Depois |
| --- | --- | --- | --- |
| 1 | tunetrail.db v3 | P1: 80 | P1: 80 |
| 2 | koito.db v3 | P1: 79; P2: 1 | P1: 79; P2: 1 |
| 3 | tunetrail.db v3 | P1: 24296; P2: 298 | P1: 24296; P2: 298 |

Nos três casos: v3→v4; digests de todas as colunas/tabelas existentes iguais; contagens por perfil iguais; integrity_check=ok, foreign_key_check vazio; estatísticas públicas reconciliadas. Backup automático v3 (0600) igual à origem copiada; restauração em outro diretório, nova migração e reinício idempotente comprovados. Os casos são independentes; não somar cópias como histórico distinto.

## Verificações

- Testes Go de migração: falha SQL após criar ledger/detalhes reverte DDL e versão da 004, preserva digests e permite retry. Down falha transacionalmente; restauração é explícita.
- Backup de tunetrail.db v3 inclui escuta comprometida somente no WAL; hook confirma schema ainda v3 depois do fsync do backup. Restore retorna v3 íntegra antes de migrar novamente.
- Bootstrap recusa v3 sem trigger, v4 sem tabela/índice/trigger ou com DDL alterado e versões desconhecidas. Recuperação/crash/reinício do legado e v2 continuam cobertos.
- Constraints: perfil cruzado, JSON/envelope/chaves incompletas, imutabilidade de snapshots/variantes, guardas da v4, IDs futuros nulos, exclusão de perfil e variantes; repetição sem chave não colapsa por segundo. Duas conexões SQLite disputam a mesma chave confiável: uma grava, outra recebe conflito UNIQUE (busy_timeout como no runtime).
- Retenção: 7/90/3650, default/off, valores inválidos, cutoff exato, reabertura, lease ativo, evidência protegida, limite global, variantes abertas/encerradas, outbox terminal versus falha aberta; núcleo preservado.
- Binário v3 compilado da main local em diretório temporário: recusa schema 4 com `unsupported schema version 4`, sem mudar digests de dados da fixture.
- `make check`: Go vet/test, verificações UI e 13/13 jornadas e2e, incluindo migration, ingest-migration e shutdown/relay. O novo e2e exercita v3 multi-perfil tanto pelo nome canônico quanto pelo legado, backup/restore, estatísticas e reinício.
- `CGO_ENABLED=1 go test -race ./internal/db/sqlite`: passou.
- Graphify AST atualizado a partir desta worktree (argumento explícito ao script externo); sem extração semântica. `git diff --check` limpo.

## Limites operacionais

Parar escritores externos antes de manutenção/restauração; o lock de bootstrap coordena os binários que o respeitam. Backup exige espaço extra e não possui política de remoção automática. Restore após novas escritas pode perdê-las e exige reconciliação antes de workers futuros. Não foram testadas integrações remotas reais nem o pipeline futuro das 081/082/133/134. Nenhum push, merge, publish ou Done.
