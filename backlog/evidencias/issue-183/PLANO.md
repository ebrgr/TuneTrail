# Plano ISSUE-183 — edição, exclusão, fusão e exportação por ocorrência

Recorte da ISSUE-135 (decisão do usuário em 21/11/2026). Contrato: doc-036, doc-039, decision-008. A 135 (v0.55) já leu por ocorrência; esta issue escreve e exporta por ocorrência.

## Objetivo
1. Editar ou remover um lançamento (ocorrência) não altera outro lançamento da mesma gravação.
2. Fusão de faixas/gravações não deduplica eventos por gravação/segundo: dois eventos simultâneos, de perfis ou de eventos distintos, sobrevivem.
3. Exportação versionada, com cursor por `event_id`, preserva ocorrência, lançamento e proveniência; reimportar o export reproduz o mesmo estado.
4. Duas escutas simultâneas de perfis distintos sobrevivem a edição, fusão, exclusão, export e reimport.

## Estado atual (conferir no código)
- `internal/db/sqlite/exports.go` `GetExportPage`: junta `listens`/`tracks`/`releases`, cursor `(listened_at, track_id)`, que não distingue dois eventos da mesma ocorrência no mesmo segundo; exige release.
- `internal/db/sqlite/track.go` `mergeTracksIn`: `UPDATE OR IGNORE` + `DELETE` por perfil/faixa/segundo, que descarta eventos simultâneos; depende de guarda `isOccurrenceGuard`.
- `DeleteTrack` e as rotas de edição em `internal/adminui` / handlers de mídia.

## Entradas obrigatórias da ISSUE-133 (comentário #2 da 135)
- Merge reversível de gravações para ocorrências equivalentes em lançamentos distintos; faixas com MBID nulo viram gravações separadas até a reconciliação.
- Confirmar reimportação e a compatibilidade integrada do corte (AC6/AC7 da 133 usam primitivas SQL em fixture pós-corte).
- Versionar a comparação HTTP do AC3 da 133 e reaplicá-la aos leitores ativados.

## Restrições
- Eventos nunca são descartados por gravação/segundo; só um mesmo `event_id`/recibo é idempotente.
- Escuta sem lançamento (release nulo) entra no export e no reimport, sem inventar álbum.
- Formato de export com campo de versão; leitores antigos continuam funcionando (manter a versão atual legível) e o cursor novo é por `event_id`.
- Migração só se necessária e aditiva (schema atual v13); sem tocar a ingestão (081/134/173).
- Isolamento por perfil em todas as operações; exclusão e edição exigem a mesma autorização atual.
- UI, se mexer: só `min-width` 40/64/90rem, CSS ≤ 400 linhas; regenerar a referência ISSUE-075.
- Medir planos de consulta do export em fixture grande (EXPLAIN QUERY PLAN) em `backlog/evidencias/issue-183/`.
- Tocam mensageria/relay? Não. Advisor: recomendado depois da implementação (fusão destrutiva e cursor de export), acionado pelo usuário.
