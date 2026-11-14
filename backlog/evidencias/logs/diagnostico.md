# Logs do TuneTrail — como funcionam e o que registram

**Data:** 14/11/2026 · **Versão:** 0.9 (`dbc8eb7`) · **Tarefa:** ISSUE-112

## Como funciona

- Biblioteca **zerolog** (`internal/logger/logger.go`), um logger global criado em `logger.Get()` e configurado em `engine.initLogger` (`engine/engine.go:31`).
- **Destino único: a saída padrão.** Não há arquivo, rotação nem retenção; no Docker os logs ficam no driver do contêiner (`docker logs`), no Unraid na tela de logs do contêiner. Ao recriar o contêiner, o histórico se perde.
- **Formato:** console colorido com hora RFC 3339 por padrão; JSON quando `KOITO_ENABLE_STRUCTURED_LOGGING=true`.
- **Nível:** `KOITO_LOG_LEVEL` = `debug`, `info` (padrão), `warn`, `error` ou `fatal` (`internal/cfg/cfg.go:267`). Também editável pela página de configurações (exige reinício).
- **Requisições** (`engine/middleware/middleware.go:63`): cada requisição recebe um `request_id` de 8 caracteres, devolvido no cabeçalho `X-Request-ID`. Ao terminar, registra `Received <método> <caminho> - Responded with <status> in <ms>` com `type=access`: em **info** para `/apis/*`, em **debug** para páginas. A query não é registrada (pode conter credenciais). Pânicos são capturados e registrados com a pilha (`type=error`).
- Logger por requisição injetado no contexto; `logger.FromContext` devolve, quando o contexto não tem logger, um **logger novo sem nível nem formato** (`logger.go:53`), que escapa da configuração.

## O que é registrado (código, sem testes e scripts)

| Nível | Chamadas | Uso típico |
| --- | ---: | --- |
| Debug | 290 | detalhes de handlers, catálogo, imagens e importação |
| Info | 66 | acesso às APIs, partida, importação, exportação |
| Warn | 15 | falhas recuperáveis |
| Error | 31 | falhas de handlers e serviços |
| Fatal | 8 | partida (`engine/engine.go`): diretório, usuário padrão, servidor |

Por área: `engine/handlers` 184, `internal/catalog` 91, `engine/engine.go` 52, `internal/images` 43, `internal/importer` 22.

## Achados

| # | Achado | Risco |
| --- | --- | --- |
| 1 | `engine/reset_password.go:46` registra `New password: <senha>` pelo logger | A senha nova vai para o log; qualquer arquivo ou visualizador de log a guardaria |
| 2 | Logs de debug do catálogo e da ingestão registram títulos de álbuns, faixas e artistas (`internal/catalog/associate_album.go` e outros), sem perfil | Histórico de escuta no log, misturando perfis; uma tela de logs precisa ser só do administrador |
| 3 | `logger.FromContext` cria um logger cru quando falta contexto | Linhas fora do nível e do formato configurados, e fora de qualquer captura |
| 4 | `RequestIDHook` (`middleware.go:20`) nunca é registrado e usa chave de contexto de tipo diferente | Código morto |
| 5 | Mensagens montadas com `Msgf` em vez de campos estruturados | Difícil filtrar por perfil, entidade ou tipo |
| 6 | Não há como ver logs pela interface; o administrador depende do acesso ao host | Diagnóstico difícil em Unraid e instalações remotas |

Pontos positivos: query string fora do log de acesso; tokens de API não aparecem nos logs (só ID ou contagem); `request_id` devolvido ao cliente.
