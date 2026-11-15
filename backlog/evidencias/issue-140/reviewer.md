Revisei o diff staged da ISSUE-140 sem alterar nenhum arquivo. Não consegui rodar os testes: o pedido de permissão para executar `go vet`, `go test ./engine -run '140|139|124|109'` e `make ui.check` não foi aprovado. Esta revisão foi feita só lendo o código.

## Parecer: não aprovado ainda. Há 1 item importante a decidir

### [IMPORTANTE]
1. **Os ícones podem cobrir o título no desktop** — `assets/css/ui/pages/media.css:14`. `.media-actions` usa `position:absolute` no canto superior direito de `.media-heading`, mas no desktop nada reserva esse espaço. O texto do cabeçalho (`.media-heading-text`, com `flex:1`) vai até a borda direita, então um título longo passa por baixo dos 4–5 ícones (cerca de 220px). No celular isso já foi resolvido com `padding-top:56px`. Solução: reservar espaço à direita do texto no desktop, por exemplo com um `padding-right` em `.media-heading-text`, ou colocar a barra no fluxo normal da página. Vale conferir nas capturas com um nome de álbum ou artista longo.

### [SUGESTÃO]
2. **`enhance.js` foi generalizado sem necessidade.** Com `options.submit`, o `install(root, options)` pula toda a lógica de seção. As mudanças de `document` para `root` (em `h1`, `status`, `base` e `artist_query`) só têm efeito quando `root === document`. Além disso, o script passou a conhecer o id `#media-task-dialog`. É mais simples manter o `enhance.js` como estava, com uma linha a mais para ignorar formulários dentro do diálogo, e deixar o `media-dialog.js` registrar o próprio `submit`. O retorno antecipado original (`if (!document.querySelector('form[data-enhance]')) return;`) também deixou de existir.
3. **`actionSection("artwork") → "image"` não tem efeito** — `internal/adminui/media.go:66`. Para pedidos do diálogo a seção é sempre trocada para `"task"`. Fora do diálogo, o formulário de imagem não tem `data-enhance`, e `manage/image` não tem um invólucro `<div id="image" data-section>`. Se algo mandar essa requisição, o fragmento vem quebrado. Isso também contradiz a decisão anterior de manter o recarregamento completo da página no envio de imagem. Recomendo remover.
4. **Redirecionamento em `catalog_review.go:46`:** `r.URL.Path+"?"+q.Encode()` usa o caminho já decodificado e pode terminar em `?` vazio. O `media.go:167` já faz o mesmo com `(&url.URL{Path:…, RawQuery:…}).RequestURI()`. Vale usar a mesma forma nos dois.
5. **`media-dialog.js`:**
   - Em `request()`, um `const request = new AbortController()` tem o mesmo nome da função. Renomear para `controller` facilita a leitura.
   - A checagem do destino depois de um `204` usa `if` aninhados. Fica mais clara como uma condição só: aceitar exatamente `/slug` ou um caminho que comece com `/slug/`.
6. **`shared/head.html:28`:** o CSS e os scripts do diálogo carregam em todas as páginas públicas, inclusive para visitantes. O script termina logo (não há `<dialog>`), mas os arquivos são baixados mesmo assim. Dá para condicionar a `.Admin`, se o template tiver acesso a esse valor.
7. **Detalhes pequenos:**
   - `internal/adminui/dialog.go` importa pacotes padrão do Go depois dos do projeto; o `goimports` inverteria a ordem.
   - Sobrou uma linha em branco em `public/media.html:41`, onde ficava o link removido.

### Ponto para você decidir
- O link "Manage this item" saiu, e a página pública não leva mais à página geral de gestão (`/manage/<tipo>/<id>`) nem à lista de escutas (`/listens`). A exclusão de escutas continua disponível pelos botões do `live.js` no histórico. Se a tarefa pede só o padrão do Koito, isso está certo.

### O que conferi e está correto
- **Segurança:** sessão, CSRF, perfil e ticket de uso único continuam sendo verificados no servidor. A política de mesma origem passou a valer também para leituras do diálogo que trazem tokens. O `confirm` só manda o navegador para outra página depois que a mudança dá certo, e uma segunda confirmação recebe `409`.
- **Cache:** as páginas públicas usam `private, no-cache` com `Vary: Cookie`, então o HTML de administrador não fica em cache compartilhado.
- **`body`:** os atributos `data-admin`, `data-profile-slug` e `data-ui-generation` e os cabeçalhos `X-TuneTrail-Admin` e `X-TuneTrail-UI-Generation` existem, em `public/live.html` e `webui/handlers.go:275`.
- **Respostas atrasadas e repetição:** fechar o diálogo cancela os pedidos em andamento, e nenhuma mudança é reenviada depois de erro ou tempo esgotado. O fragmento é validado (id, um único elemento, `data-section`) antes de ser inserido.
- **Lista de escutas:** a tarefa `add-listen` não mostra a lista, então o diálogo nunca chega a pedir `/listens`, que o servidor recusaria nesse modo.

Quer que eu aplique as correções 1, 3 e 4 (e a 2, se concordar com a simplificação)? Também posso salvar este parecer em `backlog/evidencias/issue-140/` e tentar rodar os testes de novo, se você aprovar os comandos.
