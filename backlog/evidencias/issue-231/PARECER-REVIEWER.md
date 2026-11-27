# ISSUE-231 — Parecer do Reviewer

Revisão em 27/11/2026 do diff `main...HEAD` (commit `846606a`; reauditado o acréscimo das melhorias 2 e 3 do Advisor), feita pelo mesmo agente que implementou, sem delegação. Não é uma revisão independente.

**APROVADO** — nenhum item obrigatório.

- Resolvido depois do parecer do Advisor: a regra `leaf-stays-narrow` limita os imports de `internal/imageurl`, liberado para o armazenamento. Vale para imports diretos; `imagecache` não é limitado.
- [RISCO] `internal/relayoutbox/worker_test.go`, `TestWorkerPacesFailedClaims` — se o worker deixar de reconhecer a configuração inválida, o teste trava até o limite do Go em vez de falhar logo. Comportamento anterior a esta tarefa.
- [RISCO] `internal/db/sqlite/relay_worker_store.go`, `RelayQueueStatus` — continua devolvendo `sql.ErrNoRows`. Só o próprio pacote o chama; fora do escopo.

Conferido: as mensagens dos dois erros do relay não mudaram; `FailRelaySource` grava os mesmos motivos; o horário de vencimento continua definindo a ordem do lote, agora sem campo privado em `relay.Item`; a API de primeira atividade continua respondendo 500 para qualquer erro.

**Advisor:** o diff toca a fila de entrega do relay (mensageria). Pela tabela do `CLAUDE.md`, recomenda-se ao usuário acionar o Advisor; não foi acionado.

Mudança interna, sem entrada no `CHANGELOG.md`.
