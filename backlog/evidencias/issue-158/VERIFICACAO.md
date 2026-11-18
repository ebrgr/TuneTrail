# ISSUE-158 — implementação e verificação

Executada diretamente pelo Codex, sem re-delegação. Plano conferido: `01a1efec-ad9e-77f3-b021-068cc05ae40a`, parecer do Advisor e aceite registrados na tarefa. A instrução posterior do usuário foi **manter o original** para os defaults: login `admin`, senha `changeme`, respeitando a configuração existente. Não foi introduzido login `user`.

## Alterações

- `005_accounts_profiles.sql` é aditiva: nenhuma reconstrução de `users`/`listening_profiles`, nenhum SQL 001–004 alterado. Goose aplica a 005 em uma transação com FKs ligadas. A ponte legada comprova o dono; contas sem perfil ganham perfil; perfis sem conta ganham usuário comum pendente com senha vazia e login próprio. Colisão não conciliável aborta a transação.
- **AC1 — equivalência funcional aprovada:** `ADD COLUMN user_id ... REFERENCES users(id) ON DELETE RESTRICT` + índice UNIQUE + triggers rejeitando dono NULL e troca de dono. A coluna não tem NOT NULL no `PRAGMA table_info`; a restrição é efetivamente aplicada pelos triggers para evitar reconstrução destrutiva dos pais. `checkDatabase` valida as duas direções da cardinalidade e o DDL canônico. `require_login` é NOT NULL, DEFAULT 0 e CHECK 0/1.
- `users.status` distingue active/pending, amarrando pending a senha vazia. Login rejeita pending antes de bcrypt; sessão autenticada exige active. Troca de senha ativa e revoga sessões pela transação existente. Sem reforma de senhas, permissões ou política de sessões.
- `SaveUser` cria a dupla em transação. `CreateListeningProfile` cria conta pendente, perfil público e chave juntos; login derivado dos primeiros 32 caracteres do slug, com sufixos -2..-9 e ErrConflict quando esgotados.
- `BootstrapAdministrator` calcula bcrypt pelos validadores/custo existentes fora da transação, usa INSERT condicionado à ausência de contas como primeira escrita, cria perfil default e chave Default e faz um único commit. O engine propaga CountUsers e falhas de bootstrap. Contas existentes sem admin não são promovidas: há orientação de recuperação local no log.
- A exclusão confirmada remove perfil e conta na mesma transação de escrita; a remoção do último admin reverte a operação inteira e retorna ErrLastAdmin/409, com tradução pt-BR/en. Triggers protegem exclusão/rebaixamento do último admin. Sessões de contas excluídas caem por CASCADE; a sessão do último admin permanece.
- Resolução de UserID/relay usa o vínculo canônico. `legacy_relay_binding`, `profile_relays`, sessões, tokens de ingestão, IDs de perfil e identidade musical são preservados; a ponte/trigger legados são aposentados.
- Bootstrap/validação aceitam até v5 e recusam v6 desconhecida. Backup de v4 usa `.pre-accounts-*`; v3 conserva `.pre-ingest-*`, legado conserva `.pre-tunetrail-*`. A ISSUE-133 recebeu comentário de coordenação para v6 ou posterior. Sem implementar seu pipeline.
- Fixtures de vários pacotes agora criam pares explicitamente; a referência visual aceita permanece idêntica. Jornadas antigas que excluíam todos os perfis foram ajustadas para conservar o último admin.

## Evidências automatizadas

- `make check` passou: Go vet/test de todos os pacotes, UI/JavaScript, comparação da referência visual aceita e **13/13 jornadas e2e**, incluindo permissões, isolamento, administração, relay/shutdown e migrações.
- `CGO_ENABLED=1 go test -race ./internal/db/sqlite` passou. Duas goroutines disputam o bootstrap (só uma cria), exclusões de dois admins (só uma exclui) e rebaixamentos (só um passa); o banco mantém o último admin e a cardinalidade.
- `accounts_test.go`: v3/v4, WAL comprometido, hashes/IDs/chaves/sessões/relay preservados, backup restaurado em outro diretório antes de migrar novamente, reinício idempotente; FK/UNIQUE/dono NULL/imutabilidade/default público; pendente não autentica nem cria sessão com hash vazio; ativação por troca de senha; colisões/sufixos, falhas de perfil/chave sem conta órfã e rollback de bootstrap.
- Falha SQL real da 005: colisão entre login de fallback e conta existente. A base permanece v4 com DDL, ponte e digests intactos. Após corrigir o slug na fixture, a mesma migração passa e o reinício não duplica dados.
- v2 sem usuários chega à v5 vazia e o bootstrap cria exatamente um admin active com os defaults originais, hash bcrypt válido, perfil e chave. v2 com usuário preserva-o e não cria conta extra. Base com contas sem admin não recebe promoção.
- Teste adicional `TestAccountsIncompleteLayout` passou: recusa conta órfã, ausência dos guards/índice e dono NULL após dano deliberado na fixture.
- Graphify atualizado via AST a partir desta worktree e consultado para bootstrap, criação/exclusão e autenticação. `git diff --check` limpo. Revisão local do diff conforme skills de Go/deslop/changelog; Reviewer externo ainda pendente.

## Cópias de dados reais

`scripts/validate_accounts_migration.py` copia os componentes DB/WAL/SHM/journal **antes de abrir qualquer conexão SQLite**. As origens reais nunca são abertas pelo SQLite nem alteradas. Os hashes físicos antes/depois são comparados somente em memória. Todos os bancos de teste são privados e descartados; a evidência JSON contém somente resultados e contagens.

| Cópia | Entrada | Antes por perfil | Depois por perfil |
| --- | --- | --- | --- |
| Real 1 | v3 | P1: 80 | P1: 80 |
| Real 2 | v3 | P1: 79; P2: 1 | P1: 79; P2: 1 |
| Real 1 com a 004 publicada aplicada só à cópia | v4 | P1: 80 | P1: 80 |

Em cada caso: origens fisicamente inalteradas; todas as linhas e colunas originais preservadas (a ponte é aposentada); IDs/contagens de escutas e chaves intactos; um admin active e uma conta pending; integridade/FKs e cardinalidade válidas; backup restaurado antes de remigrar; dois reinícios estáveis. Não somar as cópias como históricos distintos. Resultados em `migracoes-agregadas.json`.

## Correções dos dois riscos do Reviewer

Parecer original preservado em `PARECER-REVIEWER.md` (sobre a50dcd6). As correções foram solicitadas pelo usuário e executadas diretamente, sem re-delegação.

- **Slugs ocupados:** o helper compartilhado por SaveUser/BootstrapAdministrator tenta o slug base e sufixos determinísticos -2..-9. Se esgotados, retorna `db.ErrConflict` e a conta é revertida junto ao perfil. A instalação vazia continua usando `default`; credenciais `admin/changeme` e sua configuração não mudam.
- **Migração sem bloqueio por slug:** a 005 escolhe o primeiro candidato livre entre N+1 candidatos para N perfis existentes. Como as bases `user-<id>` de contas distintas não se sobrepõem, há sempre um candidato livre, inclusive quando a faixa até -9 está ocupada (teste escolhe `user-1-10`). Não é necessário abortar por esgotamento de slug. Não se trata da colisão de login de fallback que continua coberta pelo teste de rollback transacional. A 005 foi ajustada antes da publicação; as aplicações anteriores ocorreram somente em cópias descartáveis.
- **Logins válidos:** o schema de perfil só aceita ASCII minúsculo (`a-z`, `0-9` e hífen), e `validateUsername` admite esse conjunto, com limite de 32 caracteres. Cortar no meio de um grupo de hífens também continua válido para login. A derivação agora explicita lowercase, reserva espaço para o sufixo antes de truncar e passa cada candidato por `validateUsername` antes de inserir. O plano SQL usa lowercase/substr e CHECK equivalente para os logins derivados e seus fallbacks, sem alterar os validadores ou a política de senha.
- **Testes novos:** SaveUser com slug preexistente, esgotamento tipado sem conta órfã; migração com base ocupada e nove candidatos ocupados, preservando IDs e conteúdo dos perfis originais; slugs de 64 caracteres, hífen na posição 32, colisões após corte com sufixo -2, logins migrados conferidos pelo validador Go e rejeição de entrada interna inválida.
- **Revalidação:** `make check` passou novamente, com 13/13 e2e e referência visual idêntica. `CGO_ENABLED=1 go test -race ./internal/db/sqlite` também passou (90,457 s). Cópias reais atualmente v4 (80 escutas) e v3 (79+1 escutas), além de outra execução v4, passaram em preservação, backup/restore e reinício, com origens fisicamente inalteradas. Os resultados adicionais estão em `reviewer_fixes_revalidation` no JSON agregado. Graphify AST foi atualizado e o diff incremental foi revisado conforme deslop.

## Limites

Ativação por token, autenticação comum, autorização, gate e telas permanecem nas 159–163. As contas pendentes não têm senha temporária; sua futura ativação será emitida sob demanda pelo admin. A restauração operacional exige parar escritores e reconciliar relay/workers conforme decision-009; não foi executada restauração no banco real. A tarefa permanece Em andamento para o Reviewer; sem push, merge, publish ou Done.
