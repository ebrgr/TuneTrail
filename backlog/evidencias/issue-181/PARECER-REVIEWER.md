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
