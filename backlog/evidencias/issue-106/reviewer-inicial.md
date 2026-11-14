**Veredito: ainda não aprovado. Há 4 correções pequenas e localizadas antes do commit.** O plano em si não tem problema: catálogo, política HTTP, lexer CSS, ciclo de vida da página no histórico e a separação entre público e admin estão corretos. Não encontrei cache poisoning, corrida ou vazamento de autoridade.

## Correções necessárias

1. **Atributos duplicados em `assets/templates/public/live.html:34-37`.** `data-timezone` e `data-week-start` aparecem duas vezes cada. O HTML fica inválido; os navegadores usam a primeira ocorrência, mas isso é resto de edição. Esse HTML errado também já foi gravado no snapshot `backlog/evidencias/issue-106/reference/`. Remova as linhas 36-37 e regenere a referência.

2. **Recarga em toda volta do histórico na página de seleção de perfil.** Em `internal/webui/handlers.go:97-101`, a página sem perfil é renderizada antes de ler o cookie `tunetrail-weekstart` (linhas 104-108), então `.WeekStart` sai sempre `0`. Em `live.js` (`activate(true)`), quem tem semana começando na segunda (cookie `"1"`) compara `"1" !== "0"`. Ao voltar para `/`, a página recarrega sempre, com SSR extra, e a restauração do histórico nunca funciona ali. Os testes de navegador não pegam isso porque usam `weekstart=0`. A correção é emitir `data-week-start` só quando o valor foi lido do cookie, ou ler o cookie antes do `return` da seleção.

3. **Documentação desatualizada em `scripts/README.md:74`.** A linha ainda diz que `make ui.reference.check` compara com "os 113 arquivos aceitos da 065". Isso contradiz o novo parágrafo da linha 78 e o `Makefile`. Atualize para a referência da ISSUE-106.

4. **Faltam comentários em inglês em funções novas (convenção do projeto).** Em `live.js`: `requestScope`, `addControls` e `activate`; o comentário existente está acima de `slug`, não das funções. Em `validate_web_cache.cjs`: `observe`, `chrome` e `run`.

## Sugestões (não bloqueiam)

- **DELETE com timeout de 10 s.** O `AbortError` causado pelo timeout é descartado sem mensagem, e o botão volta a ficar ativo em silêncio. O usuário pode tentar de novo sem saber se a primeira exclusão foi aplicada. Vale separar o timeout do abort por `pagehide` (enquanto `active === controller`) e mostrar a mensagem de erro e status.
- **Timezone desconhecida no servidor.** Se o cookie `tz` traz uma zona que o tzdata do servidor não conhece e o servidor cai para outro valor, `stored.tz !== data-timezone` recarrega a cada volta. É o mesmo efeito do item 2, mas num caso raro. Uma opção é comparar com o cookie que o servidor efetivamente aceitou.
- **URLs em CSS fora de `url()`.** `image-set("x.png" 1x)`, `-webkit-image-set` e `src("...")` usam strings como URL e passam pelo lexer sem versionamento nem rejeição. Hoje não há uso, e o resultado degradaria para um alias `no-cache`, sem quebrar nada. Rejeitar `StringToken` dentro dessas funções deixaria a regra "somente `url()` literal" fiscalizada pelo código, como já está documentada.
- **`data-assets="/web-assets"` em `live.html:32`.** O atributo ficou sem uso depois de `data-themes-url`, e o `grep` não acha leitor em `assets/js`. Pode remover.
- **Idioma das docs.** A nova seção em `DEVELOPMENT.md` e o parágrafo em `scripts/README.md` estão em inglês, num arquivo que está em português.
- **Porta do Firefox em `validate_web_cache.cjs`.** A porta é escolhida fechando o listener e reabrindo depois (TOCTOU). É só código de teste, com risco baixo de colisão.

## Verificado sem problemas

- **Catálogo:** é imutável e montado antes de aceitar requisições. Alterar um valor durante o `range` é seguro, e as dependências não-CSS ficam com digest final, então a ordem não importa. Colisões com `themes.css`/`themes.json` são rejeitadas, e o `LICENSE` não entra no catálogo. Há guarda de extensão em `validEntry`.
- **HTTP:**
  - Só `v=<digest>` exato recebe `immutable`.
  - Hash antigo, query extra, `?` vazio ou alias com query dão 404 `no-store`.
  - Erros 412/416 passam por `assetResponse`.
  - O 304 mantém ETag e `Cache-Control`.
  - A busca é por mapa, sem caminho de disco.
  - A precedência do `ServeMux` deixa `/preview-assets/cover/{id}` antes do mount genérico.
- **CSS:**
  - O `url("...")` com aspas vira `URLToken` no tdewolff.
  - `url(` escapado e `@import` escapado são rejeitados.
  - A saída usa `url.URL.String()`, que escapa `"`, `\` e quebras de linha, então não há injeção de sintaxe.
  - Path traversal falha na busca do catálogo.
  - O caminho relativo é calculado a partir da URL publicada.
- **Templates:** a validação cobre condições de `if`/`range`/`with`/`template`, chains e pipes, e rejeita argumento não literal.
- **Navegação com bfcache (`live.js`):**
  - Cada ativação tem seu próprio controller, e o handler de clique delegado lê a ativação corrente.
  - Corridas entre ativações são descartadas por `active !== controller`.
  - Os controles são removidos no `pagehide`, em falha e em timeout.
  - Não há repetição automática do DELETE.
- **Admin:** `[hidden]{display:none!important}` em `ui.css` garante que o `body.hidden` realmente esconde a página. Restaurar uma página persistida sempre recarrega.
- **Documentos:** `Vary: Accept-Language` vem de `i18n.go:313` e `Vary: Cookie` de `render`. Erros e admin ficam `no-store`. `Content-Language` existe para a checagem do HEAD.

Assim que as correções 1-4 forem feitas e a referência regenerada (com o comparador passando), aprovo sem nova rodada de navegadores: nenhuma delas muda o comportamento que os testes de cache e histórico verificam.
