Revisei o diff da ISSUE-140, staged mais unstaged. Não encontrei nada crítico, mas há dois pontos que vale ajustar antes do commit. **Não rodei os testes:** os comandos de `go test`, `go vet` e `node --check` precisam da sua aprovação aqui.

Seguindo a persona Reviewer, fiz a revisão eu mesmo em vez de chamar outro `claude -p`.

## Parecer: APROVADO COM RESSALVAS

### [IMPORTANTE]

1. **`assets/templates/shared/head.html:28-30`**: todas as páginas `live` agora carregam `media-dialog.css`, `enhance.js` e `media-dialog.js`, inclusive para visitantes e para páginas que não são de mídia. Os scripts param logo no início, então não há bug, mas todo visitante baixa três arquivos de administração. Sugestão: carregar esses três só quando `.Admin` for verdadeiro e a página for de mídia.

2. **`assets/js/media-dialog.js`, handler `submit`, `if (!allowed(url)) return;`**: o `enhance.js` já chamou `preventDefault()` antes, então um formulário cujo `action` fique fora de `base` simplesmente não faz nada, sem mensagem. Hoje nenhum template gera esse caso. Ainda assim é melhor chamar `failure()` ou deixar o envio nativo seguir do que engolir o clique em silêncio.

### [SUGESTÃO]

3. **`internal/adminui/media.go:438-443`**: no modo diálogo, abrir as tarefas `merge` e `delete` cria um ticket novo, mas a página completa não cria. Isso gasta espaço no armazenamento de tickets sem necessidade. O teste `TestMediaTaskDialog140` exige o ticket nas cinco tarefas; se isso for intencional (o ticket seria usado pelas buscas de mesclagem), vale um comentário explicando.

4. **`internal/adminui/media.go`, `mediaAction` no caminho de erro**: agora ele passa `action` em vez de `""` para `mediaPage`. Isso também muda o fluxo sem JavaScript:
   - A mensagem de horário ambíguo em `listen-add` passa a aparecer (antes nunca aparecia). É uma correção.
   - Sem `return_to`, a página de erro mostra a tarefa em vez do índice.

   As duas mudanças parecem desejáveis, mas são visíveis ao usuário. Vale citá-las no comentário de verificação da issue ou no CHANGELOG.

5. **`media-dialog.js`, tempo limite de 15 s**: o upload de imagem em multipart usa o mesmo limite. Numa conexão lenta, a requisição é abortada, mas o servidor pode concluir a troca mesmo assim. A mensagem de erro já manda reabrir a tarefa, então o risco é baixo. Um limite maior só para multipart evitaria falsos erros.

6. **`assets/css/admin/media-dialog.css`**: a cor `#00538a` está fixa para o tema `pearl`. Seria melhor usar um token do tema, se houver um equivalente.

### Pontos que conferi e estão corretos
- As leituras do diálogo que carregam token (GET com `X-TuneTrail-Dialog`) agora passam pela política de mesma origem em `origin.go`. Há teste para origem externa retornando 403.
- `reviewCatalog` remove `partial` antes da lista de parâmetros permitidos e redireciona quando a requisição não vem do diálogo.
- `confirmCatalog` retorna 204 com `X-TuneTrail-Navigate`, e o cliente valida a origem e o perfil antes de navegar.
- Respostas atrasadas não aparecem depois de fechar e reabrir o diálogo, porque o `activation` é abortado.
- `pagehide` limpa os formulários abertos, e `pageshow` checa de novo a sessão com `X-TuneTrail-Admin` e `X-TuneTrail-UI-Generation`. Esses cabeçalhos e os atributos `data-admin`, `data-profile-slug` e `data-ui-generation` já existem em `live.html` e `handlers.go`.
- O `enhance.js` do documento ignora formulários dentro de `<dialog>`, então não há tratamento duplicado.

### Pendências fora do código
- A tarefa `issue-140…md` está com alteração não staged, e `backlog/evidencias/issue-140/` não está rastreado. Faltam adicionar ao commit e salvar o parecer em `backlog/evidencias/issue-140/`.
- O diretório temporário `engine/Hc6Z0UNx/` que aparecia no status já não existe.

Se você aprovar, rodo `go test ./engine -run '140|139|124|109' ./internal/...` e `make ui.check`. Também posso aplicar as correções 1 e 2 e salvar este parecer nas evidências. Quer que eu siga com alguma dessas?
