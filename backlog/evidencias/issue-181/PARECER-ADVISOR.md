# Parecer do Advisor — ISSUE-181 (commit 4c2cae3, schema v16)

Modo A, effort high. Eixos: design, concorrência/robustez, escalabilidade, regras do projeto.
Base: migração 016, `ingest_review.go`, `finishOccurrence`/`occurrenceListenCandidates`, `finalizeActiveReceipt`, handler e rotas, evidência `VERIFICACAO.md`, contrato doc-039. Não executei código nem modifiquei arquivos; o item 1 vem de leitura e não tem teste que o cubra.

**Parecer do Advisor: REPROVADO COM RESSALVAS** (1 falha de correção a resolver antes do aceite; demais não bloqueantes)

## Falhas críticas
1. **Reentrega exata passa a contar duas vezes com o Revisar desligado** (`internal/db/sqlite/ingest_occurrence.go:finishOccurrence`, bloco `if review { ...occurrenceListenCandidates... }`).
   Antes desta issue o resolvedor por ocorrência sempre gerava `pending_duplicate` quando havia uma escuta do mesmo perfil e ocorrência no mesmo segundo (`stamp == r.unix`, razão `legacy_identity_collision`) ou numa janela de ±2 s. Agora, com `review=0` (o padrão), o ramo inteiro é ignorado e a entrega vira `accepted` com nova escuta. Clientes ListenBrainz não enviam ID de evento: um reenvio do mesmo lote após timeout (comportamento normal) duplica a contagem. O doc-039 diz que a segunda entrega indistinguível "não cria a segunda contagem" e vai para revisão; o resolvedor legado também preserva isso (`finalizeActiveReceipt`: `existing != ""` → `duplicate/same` com `review_deferred`, independente da flag). Portanto o caminho por ocorrência ficou mais permissivo que o legado exatamente no caso que o corte da ISSUE-172 vai tornar o único.
   Correção mínima: manter a consulta de candidatos sempre, mas diferenciar pelo `stamp`: colisão no mesmo segundo (mesma ocorrência) com `review=0` resolve como `duplicate` / decisão `same` / razão `review_deferred`, ligada ao evento existente, como no legado; candidatos só da janela de ±2 s com `review=0` podem entrar como `accepted` (`review_deferred`); com `review=1` tudo vai para `pending_duplicate`. Adicionar teste: mesma escuta enviada duas vezes no mesmo segundo, flag desligada, uma única escuta contada.

## Melhorias (ganho real)
2. **Fila sem teto nem ação em massa**: com Revisar ligado, recibos pendentes não entram nas estatísticas até alguém decidir, 25 por página e um a um. Uma fonte ruidosa (ex.: Navidrome reenviando) acumula centenas. Sugestão: "aceitar todas como duplicata"/"manter todas separadas" por perfil (mesmo padrão do "reprocessar todos" da ISSUE-182) e um contador visível no menu.
3. **Janela de 10 mil recibos em `healthSourcesSQL`**: fontes raras somem da lista, e a ISSUE-184 (Inbox por API key) herdaria o defeito. Preferir uma tabela de resumo por fonte/chave atualizada na admissão (escrita O(1)) a uma janela por recência.
4. **Autor da decisão**: gravar o ID da conta (dono ou root) em `ingest_revisions` (campo de evidência) em vez de só `author='user'`.
5. **Padrão da flag para o corte**: depois da correção do item 1, a decisão restante é produto (ligar Revisar por padrão ou não), registrada na ISSUE-172.

## Verificado e correto
Decisão, revisão, escuta e intenção de outbox na mesma transação sob `BEGIN IMMEDIATE`; CAS por `resolution_version` e estado pendente (decisão repetida ou oposta devolve conflito); candidato validado contra a evidência e o perfil; recibo de outro perfil inalcançável; POST + CSRF e campos fechados; índice parcial `idx_ingest_health_review` cobre contagem e página; migração aditiva com guarda de downgrade; consultas medidas (fontes 1032→44 ms, decisões 283→0,1 ms).

## Regras do projeto
Issue-first, CHANGELOG/docs/comentários em inglês: conforme pelo diff; não reauditei o texto.

## Correção alternativa
Ver item 1. Após o ajuste, rever só `finishOccurrence` e o teste novo.
