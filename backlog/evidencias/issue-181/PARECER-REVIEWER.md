# Parecer do Reviewer — ISSUE-181 (commit 4c2cae3, schema v16)

Escopo: migração 016, `ingest_review.go`, `ingest_health.go`, mudança em `finishOccurrence`/`finishIngest`, rotas e handler de `/admin/health`.
Base: leitura do diff e dos arquivos centrais; os testes focados foram disparados por mim (resultado no fim).

## Achados
- [RISCO] internal/db/sqlite/ingest_occurrence.go:finishOccurrence — a consulta de candidatos passou a depender da flag `review`, desligada por padrão. Antes o resolvedor por ocorrência sempre gerava `pending_duplicate`; agora, com a flag desligada, uma escuta compatível com outra do mesmo perfil é aceita como separada (razão `review_deferred`). Duplicatas reais vindas de duas fontes (ex.: relay + Navidrome) passam a contar duas vezes até alguém ligar o Revisar. Isso importa para o corte da ISSUE-172: decidir o padrão antes de remover o legado.
- [RISCO] internal/db/sqlite/ingest_health.go:healthSourcesSQL — `LIMIT 10000` nos recibos recentes: fontes ou clientes cuja última entrega ficou fora dessa janela somem da lista "Fontes e última entrega". O Codex documentou; vale mostrar um aviso na tela ou agregar por fonte antes da janela (a ISSUE-184 reorganiza essa área).
- [RISCO] ResolveIngestReview grava `author='user'` sem o ID de quem decidiu; a trilha não diz qual dono/root decidiu.
- [RISCO] db/migrations_sqlite/016_ingest_review.sql — recria o índice único legado (O(n) em `listens` na migração) e acrescenta `review_distinct`; há backup e guarda de downgrade.
- [OK] Decisão em transação única sob `BEGIN IMMEDIATE`, com CAS por `resolution_version`, estado pendente, candidato validado dentro da evidência e do perfil; "mantida separada" cria a escuta e a intenção de outbox juntas; decisão repetida ou oposta concorrente devolve conflito (teste de corrida, 10 execuções com -race); POST + CSRF; flag por perfil com `SetIngestReview`; medição: fontes 1032 → 44 ms e decisões 283 → 0,1 ms com 300 mil recibos.

Sem BUG confirmado. Os dois primeiros riscos merecem decisão do usuário (padrão da flag; janela de fontes).

## Reauditoria após as correções (commit b0c1b66, schema v17)

Escopo: só o que mudou desde 4c2cae3 (`finishOccurrence`/`occurrenceListenCandidates`, `ingest_review_batch.go`, migração 017, `ingest_health.go`). Testes disparados por mim (resultado ao final desta seção).

- [RISCO ALTO, não executado] internal/db/sqlite/ingest_occurrence.go:occurrenceListenCandidates — o braço `l.listened_at=?` aceita como candidato exato qualquer escuta do mesmo perfil e ocorrência no mesmo segundo, **sem** a exclusão `NOT EXISTS` que protege as repetições legítimas (IDs externos confiáveis distintos na mesma fonte; mesma linha repetida num arquivo com ordinais distintos). Antes da correção esse caso ia para `pending_duplicate`; agora, com o Revisar desligado, vira `duplicate/same` fechado e a escuta não é contada. O doc-039 manda preservar repetições dentro do arquivo e eventos distintos do mesmo segundo com IDs distintos. Sugestão: aplicar a mesma exclusão ao braço exato (só é "exato" se não houver evidência de eventos distintos) e cobrir com teste de arquivo com duas linhas idênticas e de dois IDs externos confiáveis no mesmo segundo.
- [RISCO] migração 017 — gatilho em cada INSERT de `ingest_events` (UPSERT no resumo) e gatilho em cada DELETE; o backfill percorre todo o histórico (2,06 s na fixture de 300 mil). Há backup e guarda de downgrade. Aceitável.
- [RISCO] ingest_review_batch.go — a seleção percorre as pendências até achar 25 elegíveis, sob o escritor; com fila grande e poucos elegíveis o tempo cresce. Documentado pelo Codex.
- [OK] ação em massa presa a um digest calculado no servidor (reler a seleção e comparar; qualquer mudança devolve conflito e desfaz o lote inteiro); candidato único validado; transação única; ID da conta em `evidence.decided_by`; resumo de fontes sem janela, paginado em 100, 0,1 ms com 300 mil recibos; FK do resumo tem a mesma semântica diferida da tabela de recibos.

Resultado dos testes: ver comentário na tarefa.

## Reauditoria final (commit 247a0f8)

Escopo: só o que mudou desde b0c1b66 (`occurrenceListenCandidates`, `ingest_trusted_replay.go`, `receiveActiveItem`, testes novos). Testes disparados por mim; resultado no comentário da tarefa.

- [OK] occurrenceListenCandidates — a exclusão de evidência de eventos distintos (ID externo confiável distinto na mesma fonte; ordinal diferente no mesmo arquivo) agora vale também para o mesmo segundo; o RISCO ALTO da reauditoria anterior está resolvido. Testes novos cobrem os casos (a) a (f) pedidos (`TestOccurrenceDistinctSameSecond`, `TestOccurrenceTrustedReplay`).
- [RISCO] internal/db/sqlite/ingest_trusted_replay.go — com o Revisar ligado, o reenvio do **mesmo ID externo confiável** abre um recibo de revisão `pending_duplicate` (um por original, id determinístico, sem inundar a fila). Esse comportamento veio do caso (f) que eu pedi no prompt; um mesmo ID confiável é por definição a mesma entrega, então tratá-lo como candidato a revisão é discutível (a idempotência já resolve). Decisão de produto: manter (útil para auditar reenvios) ou devolver o resultado idempotente sem fila. Não é bug.
- [RISCO] ingest_review_test.go — o Codex trocou as chaves de idempotência dos testes ("first", "repeat", "manual") por vazio para exercitar o caminho sem ID; duas chaves diferentes no mesmo segundo continuam candidatas porque só IDs externos e ordinais de arquivo contam como evidência de eventos distintos. Coerente com o contrato; vale um teste nomeado para esse caso.
- [OK] índice único do ID externo continua dono do recibo original (a cópia de revisão entra com `external_id_trusted=0`); `ON CONFLICT(receipt_id) DO NOTHING` garante idempotência da pendência.

Sem BUG. Falta apenas a decisão de produto sobre o reenvio do mesmo ID confiável com Revisar ligado.
