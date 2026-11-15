Revisei a lógica Go, os testes, os scripts e a cascata, incluindo o `admin/admin.css`. Não encontrei nenhum problema bloqueante. Ficam quatro ajustes pequenos, mais um ponto para resolver antes da ISSUE-070.

**Não rodei os testes:** `go test`, `go vet` e os testes Python precisavam de uma permissão que não foi dada. Eles ainda precisam ser executados.

## Lógica Go (`internal/webassets/assets.go`)
A lógica está correta:
- `manifestSources` só aceita listas para `public/static/*.css`. Rejeita lista vazia e partes duplicadas.
- Cada parte ainda passa por `validEntry`, que confirma que um `.css` só recebe partes `.css`, e por `read`, que impõe o limite por arquivo.
- O bundle concatenado tem o seu próprio limite (`maxFileBytes`), e o hash da geração é calculado sobre os bytes finais. Por isso, editar qualquer parte muda a URL versionada.
- Um valor `null` no manifesto vira `""` e é rejeitado depois por `validEntry`. Funciona, mas a mensagem de erro sai genérica.

## Cascata
Conferi os casos que a comparação de estilos computados não alcança (hover, foco, noscript):
- **`themes.css` fica sem camada.** Ele só define as variáveis de cor (`--color-*`) em `[data-theme=…]` e não colide com nada do `tokens`.
- **`!important` dentro de camadas tem a precedência invertida.** Os únicos casos são `[hidden]`, `.cover-failed` e o bloco de movimento reduzido, e nenhum compete com outro `!important`. O `style="display:…"` em `live.html` é uma declaração normal, então o `[hidden]` continua vencendo.
- **Admin na camada `pages`.** O nav, o painel de tema e o cabeçalho de perfil ficam fora de `.admin-main`, e as classes com hover do `ui.css` não aparecem dentro de `.admin-main`. Nenhuma regra de hover ou foco muda de vencedora.
- **`:focus-visible` na camada `base`.** O `admin.css` carrega depois do `ui.css` (`shared/head.html:16-20`), então ele continua depois de `:is(a,button,…):focus-visible`. A comparação é a mesma de antes.
- **`<style>` do noscript em `nav.html:32`.** Ficar sem camada só reforça o efeito que ele já tinha.

## Ajustes sugeridos
1. **Médio, antes da ISSUE-070: `url()` dentro das partes.** O catálogo resolve `url()` a partir da pasta do arquivo servido (`catalog.go:147`), não da pasta da parte. Uma fonte escrita como `url(../../fonts/x.woff2)` dentro de `css/ui/pages/…` seria resolvida errado. Vale registrar isso no `scripts/README.md` e cobrir com um teste que coloque um `url()` numa parte aninhada.
2. **Baixo: o `admin.css` depende da declaração de ordem do `ui.css`.** Se ele for carregado sozinho, as camadas ficam na ordem em que aparecem (`pages` antes de `base`), e a `base` passa a vencer. Repetir `@layer reset, tokens, base, layout, components, pages, utilities;` no topo do `admin.css` não muda nada hoje e elimina essa dependência.
3. **Baixo: `bundle_test.go`.**
   - O teste de concatenação refaz o mesmo algoritmo do código. Seria mais útil checar pontos que o algoritmo pode errar, como uma parte sem `\n` no final.
   - A mensagem diz "does not start with the layer order statement", mas o teste só usa `Contains`. Ele deveria verificar que a declaração aparece antes do primeiro bloco `@layer … {`.
4. **Baixo: `scripts/README.md`.** O texto diz que o `admin.css` fica "na camada `pages`", mas o `:focus-visible` dele fica na `base`. Vale mencionar essa exceção.
5. **Baixo: as ferramentas de evidência são versionadas e trazem caminho absoluto local.** `conflicts.py` e `split.py` usam `ROOT = '/Users/<usuário>/...'`, o que expõe o nome de usuário e quebra em outra máquina. Melhor derivar o caminho com `git rev-parse --show-toplevel` ou receber como argumento. Além disso, `split.py` faz `sys.path.insert(0, '.')` e só funciona se rodar de dentro da própria pasta.

Os outros arquivos do diff estão coerentes: `check_web_style.py` (busca recursiva e limite de 400 linhas), `validate_web_distribution.py` e os caminhos atualizados nos testes de catálogo e de distribuição.

Quer que eu aplique os itens 2 a 5? Para o item 1, posso escrever o teste do `url()` em parte aninhada.
