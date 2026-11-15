Revisei o diff da ISSUE-139. Com `TUNETRAIL_WEB_DIR=$PWD/assets`, que é como o Makefile roda, os testes de `internal/adminui`, `internal/adminweb` e `engine` passam. Sem essa variável eles falham porque não acham os assets; isso é do ambiente, não do código.

Encontrei dois problemas que os testes não cobrem.

### 1. A mesclagem pode terminar em 404 — `internal/adminui/catalog_review.go:156-161`
Depois de confirmar uma mesclagem, agora o redirecionamento vai para `/<perfil>/<tipo>/<To>`. A página pública chama `GetTrack`, `GetAlbum` ou `GetArtist` sem `AllowUnplayed` (`internal/webui/media.go:66-78`). Por isso, `getTrackByID` executa `requireRelated` e devolve `ErrNotFound` quando o perfil nunca ouviu o item.

A edição global continua liberada para perfis que nunca ouviram a faixa, e o próprio teste novo confirma isso com `/default/manage/track/12/edit`. Então a falha acontece assim: o administrador está em `/default`, mescla 12 em 13, o catálogo muda e ele cai num 404. O comentário que foi removido protegia exatamente esse caso ("even for an unplayed survivor").

**Correção sugerida:** antes de redirecionar, ver se o sobrevivente existe no perfil (por exemplo, com `requireRelated` ou uma consulta de contagem). Se não existir, voltar para `/<perfil>`. Também vale um teste partindo de `/default`.

### 2. Um formulário antigo de exclusão de escuta cai em 404 — `internal/adminui/media.go:639-681`
Isso afeta um formulário renderizado antes do deploy, sem `history_return`, com `return_to=/second/manage/track/12?listen_at=100&tz=UTC`.

- O caminho é igual a `expected`, então o código acrescenta `/listens`, e o destino fica `/listens?listen_at=100&tz=UTC`.
- O bloco "A removed selected listen…" só remove `listen_at` quando o caminho é `expected+"/listen"`.
- Ao abrir `/listens?listen_at=100`, a validação da linha 350 procura a escuta que acabou de ser apagada e responde 404.

A exclusão funciona, mas o usuário vê um erro. Isso contraria a decisão de 15/11 de manter os endereços antigos e as ações POST.

**Correção sugerida:** naquele bloco, remover `listen_at` sempre que `u.Path` começar com `expected` (cobrindo `/listen` e `/listens`), e incluir esse `return_to` antigo em `TestManagementTaskMutationSecurity139`.

Tentei confirmar os dois com um teste temporário em `engine/`, mas a permissão para criar o arquivo foi negada. Por isso as duas conclusões vêm só da leitura do código. Ainda assim, o caminho até o 404 é direto nos dois casos.

### Pontos menores
- **`media.go:347`:** `listen_at` passou a ser aceito em qualquer tarefa da faixa, como `/edit?listen_at=100`, e ainda dispara a busca da escuta. Não é um problema de segurança, mas a tela de edição não usa isso. Daria para limitar a `listen`.
- **`routing.go`:** o import de `adminweb` ficou no mesmo bloco que a biblioteca padrão. O `gofmt` aceita, mas o `goimports` separaria.
- **Ordem do ticket em `media.go`:** o redirecionamento de endereço antigo acontece antes de gerar o ticket, então não se perde nada. Está certo, só deixo registrado que conferi.

Ainda não salvei este parecer em `backlog/evidencias/issue-139/` nem alterei o código. Quer que eu corrija os dois problemas e acrescente os testes?
