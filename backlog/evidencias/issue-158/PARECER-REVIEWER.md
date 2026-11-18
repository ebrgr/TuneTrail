# ISSUE-158 — parecer do Reviewer (diff main...HEAD, commit a50dcd6)

Revisão estática do diff (59 arquivos, +1041 −173), sem rodar testes: o executor reporta `make check` 13/13 e `go test -race ./internal/db/sqlite` aprovados.

**Resultado: APROVADO COM RESSALVAS** (nenhum [BUG] confirmado; 2 [RISCO] a decidir).

- [RISCO] internal/db/sqlite/accounts.go:26 e db/migrations_sqlite/005_accounts_profiles.sql:13 — o slug de fallback `'user-' || id` pode colidir com um perfil existente de mesmo slug (um admin pode criar o perfil "user-5" antes de a conta 5 existir). Em `SaveUser` o INSERT do perfil falha por UNIQUE e a criação da conta é revertida sem erro tipado; na migração o mesmo caso aborta a 005 (a base fica íntegra em v4, mas o upgrade fica bloqueado). Correção sugerida: escolher o slug com a mesma lógica de sufixos de `insertPendingAccount`, ou mapear a falha para `ErrConflict`; um teste cobrindo as duas rotas.
- [RISCO] internal/db/sqlite/accounts.go:36 — `insertPendingAccount` deriva o login do slug sem passar por `validateUsername`. Se o slug admitir caracteres ou tamanho que a validação de login recusa, a ativação futura (ISSUE-162) pode esbarrar em uma conta com login inválido. Verificar a compatibilidade do padrão do slug com `validateUsername` e, se divergir, normalizar na derivação.
- [OK] AC1: `user_id` sem NOT NULL no `PRAGMA table_info`, imposto por UNIQUE INDEX + triggers (dono NULL e troca de dono) + `checkDatabase`. Equivalência registrada em VERIFICACAO.md conforme o parecer do Advisor; cabe ao usuário aceitar o desvio literal do critério no aceite.
- [OK] 005 aditiva (sem DROP/rename de users ou listening_profiles), transação única, rollback por colisão e backup `.pre-accounts-*`; `DeleteListeningProfile` adquire o lock de escrita na primeira instrução e reverte tudo no `ErrLastAdmin`; sessões e login recusam conta `pending`.
- [OK] Bootstrap atômico; engine propaga `CountUsers` e não promove contas sem admin.
- Nota: o login padrão do bootstrap no código é `admin` (`internal/cfg/cfg.go:253`), não `user`. O Codex manteve `admin`/`changeme` (configuráveis por env). Conferir se é o desejado.
- Escalonamento: toca criptografia/autenticação e migração de dados; o Advisor já revisou o plano (effort=high). Recomenda-se rodar o `make check` fora do sandbox antes do aceite.
