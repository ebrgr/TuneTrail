# Parecer do Advisor — ISSUE-173 (commits b881df9, 94da32e)

Modo A, eixos: design, concorrência/robustez, escalabilidade, regras do projeto.
Base: diff `main...HEAD`, `ingest_retry.go`, migração 013, claim/finish em `ingest_resolve.go`, testes -race já executados.

**Parecer do Advisor: APROVADO (com ressalvas não bloqueantes)**

## Falhas críticas
Nenhuma. Verificado:
- Park e retry são um único UPDATE com CAS (`resolution_version`, `work_version`, `lease_owner`, `lease_until>now`); recibo estacionado sai do claim por `parked_at IS NULL` e da fila pelo índice parcial.
- `ReprocessIngest` roda sob o writer da ingestão, relê as duas versões e incrementa `work_version`, então um claim antigo nunca sobrescreve a decisão nova; perfil errado é no-op.
- `retry_attempt_base` mantém `attempt_count` como contagem vitalícia e faz a lease expirada retomar a tentativa sem queimar orçamento; o CHECK `base<=attempt_count` é preservado em claim e reprocessamento.
- Migração aditiva (ADD COLUMN) com guarda de downgrade; nenhuma escuta confirmada é removida, pois o snapshot e a chave de idempotência permanecem.

## Melhorias (ganho real)
1. **Recibo que sempre excede a lease nunca estaciona** (`ingest_retry.go` retryIngest exige `lease_until>now`; claim não incrementa a tentativa quando a lease expirou). Um recibo "veneno" que trava o resolvedor por mais de 120 s repete para sempre. Sugestão: parar também quando `attempt_count` vitalício passar de um teto (ex.: 5×) ou contar recuperações de lease separadamente.
2. **Interrupção longa estaciona tudo**: a janela automática é 1+2+4+8 = 15 min. Uma queda de MusicBrainz/rede maior que isso estaciona toda a fila, e a UI lista 25 itens com reprocessamento unitário. Sugestão para a 181/seguinte: ação "reprocessar todos" por perfil e, se a causa for transitória de dependência externa, considerar não gastar orçamento.
3. `INDEXED BY idx_ingest_resolver_queue` falha com "no such index" se a consulta deixar de implicar o WHERE do índice parcial. Está coberto por teste; manter o teste como guarda.
4. Reprocessamento sem trilha de auditoria (quem/quando); registrar revisão na 181.
5. Comentário de `warnParkedIngest` diz "lifetime of the worker", mas a janela é 1 minuto.

## Regras do projeto
Issue-first, CHANGELOG em inglês, docs e comentários em inglês: conformes.
