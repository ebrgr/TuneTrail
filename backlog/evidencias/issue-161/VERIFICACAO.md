# ISSUE-161 — implementação e verificação

## Decisão sobre o cache legado

O doc-041 §3 exige que alternar `require_login` invalide as referências de cache que possam continuar expondo derivados de escutas. Também reconhece que cópias já baixadas ou publicadas por terceiros não podem ser retiradas.

A rota existente `/image/{image_id}/{filename}` registra um handler global em `engine/routes.go`. Em `engine/handlers/image_handler.go`, o handler lê o cache de disco ou baixa a imagem e responde com `Cache-Control: public, max-age=2592000` antes de `http.ServeFile`. São 30 dias de validade pública. A URL não inclui perfil nem geração. Mesmo que a nova versão recuse a URL antiga na origem, um navegador ou cache intermediário que já tenha uma resposta fresca pode reutilizá-la sem enviar pedido ao servidor. Alterar a geração, enviar novos cabeçalhos e validar condicionais na origem passam a valer quando a requisição chega ao TuneTrail.

O usuário escolheu a opção 1 em 19/11/2026. A origem autoriza cada pedido antes de consultar o cache ou avaliar uma condicional. Respostas antigas ainda frescas são cópias já distribuídas. O usuário aceitou a validade residual de até 30 dias em caches fora do controle do servidor, dispensando corte operacional e purga. Decisão registrada na tarefa com autoria `ebrgr` e no doc-041 §3.

## Inventário antes da implementação

| Superfície | Caminho encontrado | Verificação na versão atual |
| --- | --- | --- |
| Diretório e documentos | `engine/routes.go`, `internal/webui/routing.go`, `internal/webui/handlers.go` | A raiz renderiza a seleção de perfis; os documentos com slug resolvem o perfil antes de despachar home, histórico, rankings, rewind e mídia. A leitura ainda não aplica `require_login`. As rotas legadas redirecionam para um perfil único ou para a seleção. |
| JSON | `bindProfileReads` em `engine/routes.go`, `engine/middleware/profiles.go` | O escopo resolve perfil explícito ou único. Leituras de detalhes, aliases, interesse, rankings, histórico, atividade, primeira atividade, estatísticas, resumo, busca e now-playing precisam do gate. Os endpoints de identidade pública do diretório têm contrato próprio. |
| Busca e fragmentos | `internal/webui/search.go`, `internal/db/sqlite/search.go`, `internal/db/sqlite/track.go` | O despacho de `partial=1` ocorre após resolução do perfil. As subconsultas observadas em `SearchArtists` e `SearchAlbums` verificam a existência de escutas sem usar o ID para selecionar as escutas do perfil. Um gate apenas no dispatcher não protege a associação nessa busca. |
| Imagens | `engine/handlers/image_handler.go`, `imagecache` | A rota global aceita UUID e nome de variante, consulta o cache e pode baixar da origem sem política de perfil. Avatares do diretório têm acesso público pelo contrato. |
| Gestão e exportação | `engine/routes.go`, `engine/handlers/export.go`, `internal/adminui/admin.go`, `internal/adminui/media.go`, `internal/adminweb` | As rotas de exportação HTML e JSON pertencem à gestão. A autorização existente distingue dono e root; a checagem ponta a ponta ainda está pendente. |
| Now-playing e live | `engine/handlers/now_playing.go`, `internal/listening/now_playing.go`, `internal/webui/handlers.go`, `assets/js/live.js` | Foram localizadas a leitura JSON e a renderização na home. A inspeção inicial das rotas e adaptadores não encontrou endpoint de widget, RSS ou SSE registrado. |
| Prefetch e cache de documentos | `internal/webui/handlers.go`, evidência da ISSUE-075 | O prefetch com cookie recebe 204 e o documento bem-sucedido usa `private, no-cache`. A regra atual considera a ausência de cookie; a elegibilidade por `require_login` ainda está pendente. |
| Ingestão e relay | `/apis/listenbrainz/1` em `engine/routes.go` | A entrada de máquina usa a chave de ingestão. O teste de entrada e saída com perfil protegido está pendente. |

A inspeção foi estática. A matriz visitante, usuário sem acesso, dono e root para perfis públicos e protegidos ainda não foi executada. Nenhum critério de aceite foi comprovado nesta etapa.


## Resultado por papel

Os resultados abaixo usam sessões reais e o roteador de produção. **V** é visitante, **O** é usuário de outro perfil, **D** é dono e **R** é root.

| Superfície | Público V / O / D / R | Protegido V / O / D / R |
| --- | --- | --- |
| Diretório HTML e identidade pública | disponível para todos | disponível para todos, com cadeado, sem escutas, capas ou contagens |
| Home, histórico, mídia, rankings, gráficos e retrospectiva | 200 / 200 / 200 / 200 | 404 / 404 / 200 / 200 |
| JSON de escutas, estatísticas, atividade, resumo e mídia | 200 / 200 / 200 / 200 | 404 / 404 / 200 / 200 |
| Busca HTML, JSON e fragmento `partial=1` | 200 / 200 / 200 / 200 | 404 / 404 / 200 / 200 |
| Now-playing JSON e home | 200 / 200 / 200 / 200 | 404 / 404 / 200 / 200 |
| Exportação JSON | 401 / 403 / 200 / 200 | 404 / 404 / 200 / 200 |
| Exportação HTML | 401 / 403 / 200 / 200 | 401 / 404 / 200 / 200 |
| Capa associada, URL com perfil, GET e HEAD | disponível para todos | 404 / 404 / disponível / disponível |
| Avatar do diretório | disponível para todos | disponível para todos |
| Editar visibilidade em gestão | 401 / 403 / permitido / permitido | 401 / 403 / permitido / permitido |

- `AuthorizeAction` resolve `ReadListening` antes do despacho HTML e JSON e antes de ler imagens de um perfil. A autorização de dono/root reaproveita `ManageProfile`. As exportações ainda exigem autorização de gestão. Rotas antigas sem slug também aplicam a política quando o perfil único pode ser resolvido.
- A busca de artistas, álbuns e faixas usa a CTE que limita `listens` ao perfil. O teste consulta um perfil público por artistas, álbuns e faixas presentes apenas no perfil protegido. O corpo e os cabeçalhos da negativa de leitura são equivalentes aos de perfil inexistente.
- A URL atual de capa inclui o ID do perfil. A origem autoriza o perfil e verifica a associação antes de acessar disco, gerar variante ou buscar da fonte. Um evento now-playing ativo também pode justificar a imagem no seu perfil. A URL legada serve avatares e capas associadas a escutas públicas. O root autenticado pode servir imagens do catálogo em suas operações editoriais por essa URL, com cache privado quando a sessão for necessária.
- Capas públicas revalidam na origem. Capas protegidas retornam `private, no-store`. O ETag inclui o estado da visibilidade e o arquivo servido. A troca do checkbox produz um validador diferente para os mesmos bytes, sem aceitar validadores antigos de data. O teste aquece o cache, envia GET/HEAD condicionais, confirma 404 antes de 304 para visitante, recebe 304 para o dono com o ETag protegido e confirma o retorno a acesso público.
- A ISSUE-075 continua anunciando Speculation Rules apenas em documentos públicos bem-sucedidos sem cookie de sessão. Negativas não contêm regras. O roteador atual tem now-playing e seus consumidores HTML e JSON, sem endpoint de widget, RSS ou SSE.
- Ingestão ListenBrainz por chave e relay de saída foram exercitados com perfil protegido e destino HTTP de teste. O teste verifica token, caminho e faixa do evento encaminhado, a escuta persistida e a negativa de leitura com a chave de máquina.
- A referência determinística consumida por `ui.reference.check` foi atualizada para incluir cadeado em um perfil de teste, checkbox de gestão e URLs de capas com escopo. As mudanças restantes nos documentos públicos são a geração de templates recalculada. As migrações publicadas permanecem intactas.

## Validação final

- `make check` passou com go vet, testes Go sem cache, comparação exata da referência e 13/13 jornadas E2E.
- `go test -race -count=1 ./engine/... ./internal/admin ./internal/adminui ./internal/adminweb ./internal/catalog ./internal/db/... ./internal/models ./internal/webui -timeout 10m` passou com `CGO_ENABLED=1` e os caminhos de assets e idiomas da worktree.
- `git diff --check` passou. Graphify AST atualizado na worktree, com 4.410 nós e 19.373 arestas. A consulta posterior percorreu a política, os adaptadores e as verificações de imagens.
- A matriz e os testes de cache comprovam o AC #1 nas superfícies existentes, o diretório e a gestão comprovam o AC #2 e a entrada ListenBrainz com relay de saída comprovam o AC #3.
