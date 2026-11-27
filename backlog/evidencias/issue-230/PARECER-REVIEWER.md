# ISSUE-230 — Parecer do Reviewer

Revisão em 27/11/2026 do diff da branch (`internal/architecture/`, `DEVELOPMENT.md`), feita pelo mesmo agente que implementou, sem delegação. Não é uma revisão independente.

**APROVADO** — nenhum item obrigatório. Limites conhecidos:

- [RISCO] `internal/architecture/architecture_test.go`, `exceptions` — cada exceção vale para o pacote inteiro: `webui`, `summary` e `relayoutbox` podem ganhar outro uso de `database/sql` sem falha. Comandos SQL literais nesses pacotes continuam reprovados.
- [RISCO] `sqlText` — só reconhece palavras-chave em maiúsculas, em uma mesma string. SQL em minúsculas ou montado em pedaços passa; a regra de import do driver continua valendo.
- [RISCO] `sqlDrivers` — lista fechada de drivers; um driver novo precisa ser acrescentado.
- [RISCO] `sqliteBase` — a regra 6 olha imports diretos. `internal/utils`, que está na lista, já importa `internal/mbz`.
- [RISCO] `internal/testfixtures` — a exceção de SQL pressupõe uso só em testes (34 arquivos de teste o importam; nenhum de produção hoje), mas nenhuma regra impede um import de produção.

Nenhum código de produção foi alterado. Sem entrada no `CHANGELOG.md` (mudança interna).
