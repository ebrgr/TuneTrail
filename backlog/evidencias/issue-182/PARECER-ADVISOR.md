# Parecer do Advisor — ISSUE-182 (commit cc304ef, schema v18)

Modo A, effort high. Eixos: design, concorrência/robustez, escalabilidade, regras do projeto.
Base: migração 018, `claimNextIngest`, `ingest_retry.go`, `ResolveNextIngest`, handler/rotas, evidência do Codex e a 173 (013). Não executei código nem modifiquei arquivos; o item 1 vem de leitura (não verifiquei o timeout do cliente MusicBrainz).

**Parecer do Advisor: APROVADO** (sem falhas críticas; melhorias de robustez abaixo, nenhuma bloqueia o aceite)

## Falhas críticas
Nenhuma. Verificado: a decisão de estacionar acontece no mesmo UPDATE do claim (CAS por recibo, sem janela entre ler e gravar), respeita o CHECK da 013 (estado `retryable`, lease nula, `closed_at` nulo) e não emite novo claim; o reprocessamento reinicia os dois orçamentos juntos (`retry_attempt_base` e `claim_attempt_base` = `attempt_count`); lote de reprocessamento sempre termina (cada passo limpa `parked_at`); o `profile` do formulário precisa coincidir com o perfil autorizado; o ID da conta que agiu fica registrado.

## Melhorias (ganho real)
1. **O teto só dispara se a tentativa voltar.** `ResolveNextIngest` aplica 20 s só à prova do MusicBrainz (`proofCtx`); `catalog.ResolveListenTrack` e o resto usam o `ctx` do worker sem prazo. Uma chamada que trava (rede, banco) bloqueia o worker; a lease expira em 120 s, mas o worker nunca chega ao próximo claim, então o contador não avança e a fila inteira para. Sugestão: prazo por tentativa (por exemplo 100 s, abaixo da lease de 120 s) derivado de `context.WithTimeout`, para que a tentativa falhe, passe pelo `retryIngest` e o backoff e o teto funcionem. Sem heartbeat, qualquer tentativa acima de 120 s também perde silenciosamente seu `finishIngest`/`retryIngest` (ambos exigem `lease_until>now`) e é refeita.
2. **Reprocessar um recibo `lease_exhausted` devolve 25 claims.** Se o recibo é um "veneno" (trava ou derruba o processo), o botão "Reprocessar todos" o põe de volta e ele pode ocupar o worker por até 25 × 120 s antes de estacionar de novo. Sugestão: orçamento menor após reprocessamento manual (por exemplo 5 claims) ou excluir `lease_exhausted` do "todos" e deixar só a ação unitária.
3. **"Reprocessar todos" numa só transação** mantém o escritor até o commit (116 ms para 10 mil; cerca de 1 s para 100 mil). Um commit por lote é seguro (a idempotência vem de `parked_at`) e evita parar o ACK da ingestão.
4. **Auditoria só do último reprocessamento** (`reprocessed_at/by` sobrescritos): não há histórico se um recibo é reprocessado várias vezes. Uma linha em `ingest_revisions` (autor, motivo `reprocessed`) resolveria sem nova tabela.
5. O WARN de `lease_exhausted` usa o mesmo limitador de 1 minuto por perfil e motivo; correto, mas um contador visível na central (Falhas permanentes já lista) seria suficiente, sem alerta adicional.

## Regras do projeto
Issue-first, CHANGELOG e docs em inglês, comentários em inglês: conforme pelo diff; não reauditei o texto. Migração aditiva v18 com guarda de downgrade: conforme.

## Correção alternativa
Não se aplica (aprovado). Os itens 1 e 2 merecem uma issue de follow-up se não forem feitos agora.
