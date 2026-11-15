Parecer: **não aprovaria ainda**. Há um bug de comportamento na página de edição fora do diálogo e uma inconsistência no fuso horário. O resto são ajustes menores. Fiz a revisão lendo o diff e conferindo `enhance.js`, `actionSection` e a ordem dos `@layer` no repositório. Não rodei testes nem o script de validação.

## Obrigatórios

**1. Na página de edição fora do diálogo, "Adicionar artista" passou a recarregar a página inteira.**
Os formulários `artist-add` agora ficam dentro de `#artist-results`, que tem `data-section`. Em `assets/js/enhance.js:38`, `form.closest('[data-section]')` encontra `#artist-results`, mas o servidor responde a `artist-add` com o trecho `artists` (`actionSection`, em `internal/adminui/media.go:62`). Como os nomes não batem, `enhance.js:59` recarrega a página toda. No diálogo isso não aparece, porque toda resposta vira `task`.
Correção: deixar o `data-section` de fora quando o bloco é renderizado dentro de `#artists`, ou fazer `enhance.js` procurar a seção mais externa. Vale um teste e2e da página `/manage/track/N/edit` sem diálogo.

**2. Se o fuso salvo no `return_to` for inválido, o campo vazio dá erro, mas a dica mostra outro fuso.**
`webquery.Timezone` ignora um `tz` inválido e usa o cookie ou o padrão (há teste para isso: `TestTimezoneFallbackKeepsProcessLocalAndUsesValidCookieAfterInvalidQuery`). Por isso, ao abrir o formulário, a dica mostra um fuso válido. Já o POST, em `mediaAction`, valida o `tz` salvo com `time.LoadLocation` e recusa: o usuário deixa o campo vazio, como a dica sugere, e recebe "Fuso horário inválido".
Correção: remover essa validação extra e confiar no `webquery.Timezone(zoneRequest)`, que já resolve o fuso do mesmo jeito que o GET.

**3. A mensagem de erro da busca continua na tela depois que a busca volta a funcionar.**
Em `media-dialog.js`, quando a busca falha, o código faz `status.textContent = dialog.dataset.searchError`. Quando uma busca seguinte dá certo, `status` não é limpo, e "Não foi possível buscar" fica no topo do diálogo ao lado de resultados válidos. O script de validação não pega isso, porque só confere `#merge-results .section-error`.
Correção: limpar `status` quando a troca de resultados der certo.

## Recomendados

- **`media-dialog.css`:** no fim do arquivo, a regra `.manage-result small { flex-basis:100% }` está sem indentação. Além disso, `.manage-result span small { display:block }` dentro do `@media (max-width:480px)` repete a regra base e pode sair.
- **Troca de `@layer pages` para `components`:** `media.css` está em `pages` e, portanto, vence qualquer regra de `components`, seja qual for a especificidade. Hoje não vi conflito, mas qualquer seletor genérico de `button`, `input` ou `h2` adicionado a `media.css` vai passar por cima do diálogo. Deixe um comentário explicando a escolha.
- **`color-scheme`:** o padrão é `dark`, e só a Pearl vira `light`. Se aparecer outra paleta clara, os controles nativos (seletor de data, barra de rolagem) vão ficar escuros. Seria melhor derivar isso do tema do que fixar só para a Pearl.
- **Fuso digitado à mão:** `time.LoadLocation("Local")` é aceito e usaria o fuso do servidor. Vale recusar `Local` quando o usuário digita o fuso.
- **Busca de mesclagem já preenchida:** `len(v.Name) <= 100` conta bytes, e o `maxlength` do campo conta caracteres. Nomes com acentos perto do limite não vão vir preenchidos. É inofensivo, mas usar `utf8.RuneCountInString` deixa a regra coerente.
- **Formulário "Adicionar escuta":** o `<summary>` e o `<label>` dentro de `.listen-offset` repetem o mesmo texto, então o leitor de tela anuncia duas vezes. Também há um espaço sobrando em `<label >`.
- **Ruído no diff:**
  - Em `messages.go`, a chave nova foi posta no topo e o gofmt realinhou o bloco inteiro. No fim do bloco, o diff seria de uma linha.
  - Em `langs/*.json`, as chaves novas também foram para o topo, fora do agrupamento.
  - Em `media.html`, `title="{{datetime .}}"` e os `data-tip`/`title` ficaram com a indentação quebrada.
- **Testes que faltam:** o caso de `tz` inválido no `return_to` (item 2) e o envio de artista fora do diálogo (item 1).

## O que está bom

- A busca ao vivo não emite ingressos novos, é cancelada ao fechar o diálogo, descarta respostas antigas e checa o nome do trecho antes de trocar os resultados.
- O servidor resolve o fuso quando o campo fica vazio, e o contrato antigo do POST continua funcionando, com testes de horário inexistente e ambíguo.
- Sem JavaScript, as dicas continuam aparecendo pelo `title`.
- A entrada do CHANGELOG está adequada.

Posso corrigir os três itens obrigatórios. Também posso registrar este parecer em `backlog/evidencias/issue-140/`, que é onde a persona do Reviewer guarda os pareceres.
