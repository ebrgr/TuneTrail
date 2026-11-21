# Plano ISSUE-135 — ranking por gravação ou por lançamento

Decisões do usuário (21/11/2026): fatiar em duas issues (edição/exclusão/fusão/exportação vão para a ISSUE-183); escuta sem lançamento entra no balde "Lançamento desconhecido"; Advisor (effort high) depois da implementação.

## Objetivo
A tela de faixas mais ouvidas ganha a chave "por gravação" / "por lançamento", padrão **por gravação**. A soma por lançamento (incluindo o balde sem lançamento) é igual ao total por gravação. Rankings e novidades por perfil seguem o contrato da ISSUE-132 (doc-039, decision-008).

## Estado atual
`GetTopTracksPaginated` (`internal/db/sqlite/track.go`) agrupa `listens` por `track_id` (= ocorrência = por lançamento) em três ramos: sem filtro, por álbum e por artista. Filtros do ranking em `internal/webquery/html.go` (`RankingFiltersAllowed`), página em `internal/webui/handlers.go` (`ranking()`).

## Restrições
- Sem nova migração, salvo índice comprovado por EXPLAIN. Não alterar o escritor de ingestão.
- Ranking por gravação agrupa por `recording_id`; faixas com MBID nulo são gravações separadas (o ranking fica dividido até a reconciliação). Não fundir por título.
- Escuta com release nulo aparece como "Lançamento desconhecido" (i18n en/pt-BR) e conta só na gravação e no total; adaptar INNER JOINs que exigem release.
- Pendências não contam antes da resolução. Filtros de perfil e URLs antigas de ocorrência continuam valendo.
- Duas escutas no mesmo segundo, de perfis distintos, contam ambas.
- Chave na UI: parâmetro de URL, padrão por gravação, mesmo visual das demais chaves; só `min-width` 40/64/90rem; CSS ≤ 400 linhas por arquivo; regenerar a referência ISSUE-075 se a UI mudar.
- Medir planos (EXPLAIN QUERY PLAN) numa fixture grande e registrar em `backlog/evidencias/issue-135/`.
- Fora do escopo: edição, exclusão, fusão, exportação (ISSUE-183).
