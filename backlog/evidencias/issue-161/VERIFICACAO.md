# ISSUE-161 — verificação preliminar

## Decisão pendente sobre o cache legado

O doc-041 §3 exige que alternar `require_login` invalide as referências de cache que possam continuar expondo derivados de escutas. Também reconhece que cópias já baixadas ou publicadas por terceiros não podem ser retiradas.

A rota existente `/image/{image_id}/{filename}` registra um handler global em `engine/routes.go`. Em `engine/handlers/image_handler.go`, o handler lê o cache de disco ou baixa a imagem e responde com `Cache-Control: public, max-age=2592000` antes de `http.ServeFile`. São 30 dias de validade pública. A URL não inclui perfil nem geração. Mesmo que a nova versão recuse a URL antiga na origem, um navegador ou cache intermediário que já tenha uma resposta fresca pode reutilizá-la sem enviar pedido ao servidor. Alterar a geração, enviar novos cabeçalhos e validar condicionais na origem passam a valer quando a requisição chega ao TuneTrail.

Decisão necessária: tratar as respostas ainda frescas emitidas por versões anteriores como cópias já distribuídas, aplicando o gate a todas as requisições que chegarem à origem e impedindo novo armazenamento indevido, ou exigir um corte operacional com purga dos caches compartilhados sob controle do operador e tratamento da janela residual de até 30 dias para caches fora desse controle.

## Inventário inicial por superfície

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
