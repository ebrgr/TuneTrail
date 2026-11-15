**APROVADO.** O delta não traz itens obrigatórios. Não executei testes, como pedido.

**Por que a correção funciona** (`internal/adminui/media.go:219-233`, conferido com as linhas 305-388 e 404-417)
- **Causa do erro:** antes, `query = parsed.Query()` levava `year`, `week`, `page`, `limit` e `track_id` da URL pública. A lista de chaves permitidas em `media.go:315` recusava essas chaves, a função terminava em `h.fail` e o formulário se perdia.
- **Filtro:** agora só passam `tz`, `artist_query`, `merge_query`, `listen_at` e `history_return`. São as chaves que `media.go:315` aceita, menos `partial`, que nunca deve vir num `return_to`.
- **Retorno preservado:** `v.ReturnTo` continua vindo do `return_to` original e completo (`media.go:413`). O redirecionamento ao histórico filtrado segue intacto.
- **Tela renderizada:** o caminho `/second/listens` tem só 2 segmentos, então a tarefa vem de `actionTask("listen-delete")`, que é `"listens"` (`media.go:715`). Com isso a regra `task == "listen" && !q.Has("listen_at")` (`media.go:354`) não se aplica, e a página volta com 400, `id="listens"`, `Error` preenchido, `v.Values` do formulário e um ticket novo (`media.go:442-443`).
- **Fuso horário:** `tz=UTC` do histórico ainda é validado (`media.go:234-236`) e aplicado à tela. Um `tz` inválido continua dando falha, como antes.
- **Retornos de gestão:** quando o `return_to` aponta para a própria página de gestão, a lista cobre todas as chaves que o servidor gera ali. A única diferença é que chaves desconhecidas num `return_to` forjado agora são ignoradas em vez de dar 400. Não há risco, porque `SafeReturn` continua limitando o destino.

**Teste** (`engine/adminui_tasks_139_test.go:434-449`)
- Passa pelo roteador principal e confere que `return_to` é a URL do histórico, que a resposta é 400 com a seção `listens`, que o retorno aparece escapado, que há `csrf_token` e que o `submission_token` é novo.
- O GET final para `/listen?listen_at=100` com 200 confirma que a escuta não foi apagada.
- `html.EscapeString(back)` deve bater com o escape do `html/template`, pois `back` não tem `+`. Fica para o implementador confirmar ao rodar.

**Observação (não obrigatória)**
- Depois do erro, a página volta na tarefa `listens` e não na confirmação `listen?listen_at=100` em que o usuário estava. Isso acontece porque o contexto `listen_at`/`history_return` não está no `return_to` público. Atende à sugestão (mensagem, formulário, ticket e retorno mantidos), mas, se a ideia for reabrir a confirmação, seria preciso levar `listen_at` em outro campo do formulário. Isso é escopo de outra issue.
