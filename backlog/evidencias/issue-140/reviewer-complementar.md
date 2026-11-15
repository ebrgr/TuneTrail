Fiz a revisão do diff. Não encontrei nada que impeça o commit. Há um ponto sobre o que está staged e alguns ajustes pequenos que vale considerar. Não consegui rodar a consulta ao graphify nem os testes, porque esta sessão não tem shell. Por isso li só os trechos de `internal/adminui/media.go` e `assets/templates/public/media.html` necessários para conferir o que o diff não mostra.

## Confirmado no código
- **`review` não cai no novo `case` de diálogo.** Ele é tratado antes do `switch`, em `internal/adminui/media.go:143`, então o fragmento `review` do teste está correto.
- **Mudar `""` para `action` na chamada de sucesso** (`media.go:655`) é seguro. O bloco de `metadata` e `listen-add` em `media.go:468-479` só roda quando `actionErr != nil`, e nas seções antigas o `task` continua sendo zerado (`media.go:204`).
- **Tickets:** a busca no diálogo reaproveita o ticket via header. `merge` e `delete` não criam ticket nem na busca, já que o header vem vazio e o `task` é `merge`. Abrir uma tarefa cria um ticket novo, como na navegação de página inteira.
- **Revalidação após voltar ao cache:** o `HEAD` já envia `X-TuneTrail-Admin` e `X-TuneTrail-UI-Generation` (`internal/webui/handlers.go:275-277`), e o teste da ISSUE-109 cobre isso.
- **Origem:** o `X-TuneTrail-Navigate` é validado no cliente dentro do slug do perfil, e a checagem de origem cobre os pedidos de diálogo (`internal/adminweb/origin.go:16`).

## Atenção antes do commit
1. **O `live.html` não está staged.** No git status, `assets/templates/public/live.html` aparece como ` M` (alterado só no working tree). Já `media-dialog.js`, `enhance.js`, `media.css` e `head.html` estão `MM`/`AM`, com uma parte staged e outra não. Se o commit levar só o índice, a página pública não carrega o `media-dialog.js` e o diálogo nunca abre. Rode `git add` nesses arquivos antes de commitar.

## Ajustes recomendados (pequenos, não bloqueiam)
2. **`media-dialog.js`, clique dentro do diálogo:** `event.target.closest('a')` também pega um `<a>` sem `href`, e aí `new URL(link.href)` lança `TypeError`. Use `closest('a[href]')`. Também vale ignorar links com `target` ou `download`, para não capturar um link de mesma origem feito para abrir em outra aba.
3. **`clear()` não restaura o `content.inert`.** Se o diálogo for fechado no meio de um pedido, o `finally` não entra no ramo `owner === activation` e o conteúdo fica `inert`. Hoje isso não aparece porque todo `open()` faz um novo pedido, mas é frágil. Coloque `content.inert = false` em `clear()`.
4. **Fechar com Esc durante uma alteração.** O pedido é abortado, mas o servidor pode já ter aplicado a mudança, e o cabeçalho fica desatualizado sem nenhum aviso. Uma opção é guardar uma flag "alteração enviada" e chamar `refresh()` no `close` quando ela estiver ativa.
5. **Escopo do `refresh()`.** Ele só troca o `.media-heading`. Depois de adicionar uma reprodução, o contador atualiza, mas "Last played", o gráfico e a atividade continuam antigos. Se isso for aceitável, registre a decisão na issue.

## Script de validação
- `scripts/validate_media_dialogs.cjs` tem fixos o caminho do Chrome no macOS e o endereço `127.0.0.1:4111` do Koito. Para um script de desenvolvimento tudo bem, mas vale citar no `Makefile` ou na evidência como rodá-lo.

Se quiser, aplico os itens 2 e 3 agora. Depois disso, o próximo passo pela convenção do repositório é o parecer do Reviewer em `backlog/evidencias/issue-140/` e rodar `analisar-tunetrail.sh`.
