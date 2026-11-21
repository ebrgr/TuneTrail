# Parecer do Reviewer — ISSUE-135 (commit cd023ca)

Escopo: diff `main...HEAD` (ranking por gravação/lançamento, leitores por ocorrência, UI e URL `group_by`).
Verificação: `go test -race` em `internal/webui`, `webquery`, `adminweb`, `internal/db/...` — OK. Codex reportou `make check` 13/13.
Nota: o arquivo anterior ("APROVADO") foi escrito pelo agente despachado; este parecer o substitui.

## Achados
- [RISCO] internal/db/sqlite/track.go:topTracksQuery — o agrupamento por lançamento usa B-tree temporária (193 ms em 600 mil escutas, contra 69 ms por gravação). Aceitável hoje; reavaliar se os históricos crescerem (a 177 trata leituras repetidas).
- [RISCO] internal/db/sqlite/track.go:CountTracks/CountNewTracks — os totais passaram a contar gravações, mesmo quando o ranking está "por lançamento": o contador da Home pode divergir da paginação nesse modo. Coerente com o padrão por gravação; documentar na tela se confundir.
- [RISCO] internal/db/sqlite/listen.go:GetListensPaginated — o histórico passou a ler de `occurrences` (título apresentado), comportamento visível: faixas sem título apresentado dependem do alias primário. Coberto por teste, mas vale conferir numa base real.
- [OK] `MIN(o.id)` só vê ocorrências ouvidas pelo perfil (filtro vem da junção com `listens` escopada); MBID nulo mantém gravações separadas; release nulo forma o balde "Lançamento desconhecido"; `group_by` validado (valor único, só na seção de faixas) e preservado no retorno de login; `client` é NOT NULL, sem risco no scan; sem migração.

Sem BUG obrigatório. Não toca criptografia nem mensageria; o Advisor foi pedido pelo usuário (effort high) e segue pendente.
