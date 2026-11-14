# 🛡️ Parecer do Advisor: REPROVADO COM RESSALVAS

O plano está bem direcionado, mas não pode ser implementado como está. Há um defeito real no `live.js` que derruba o AC#4, quatro lacunas de escopo e uma parte com engenharia demais. Foram lidos apenas os fontes autorizados, sem estudos privados, sem edições e sem delegação.

**Sobre as regras do projeto:** o `AGENTS.md` lido contém apenas `@AGENTS.md`, a mesma referência que está no `CLAUDE.md`. As regras não puderam ser conferidas, então o item 4 do protocolo ficou sem verificação.

## Falhas críticas

1. **Botão de exclusão trava depois de voltar pelo bfcache** (`assets/js/live.js:37-38,53,57`).
   - Existe um único `AbortController`, e ele é abortado no `pagehide` (`{once:true}`).
   - Quando a página volta do bfcache, todo `DELETE` falha na hora com `AbortError`.
   - Nesse caso o código faz `return` antes de `button.disabled=false`, e o botão fica travado.
   - A ISSUE-106 vai expor esse defeito, então a correção é obrigatória dentro dela.

2. **Retenção de uma geração e páginas lidas do cache HTTP.**
   - Com `private, no-cache`, voltar por fora do bfcache pode reaproveitar o HTML do cache HTTP. Isso depende do navegador e precisa ser medido.
   - Esse HTML antigo pode apontar para hashes de uma geração já trocada por reinício.
   - Na maioria das vezes, o CSS e o JS imutáveis continuam no cache do navegador.
   - Os carregamentos tardios falham com 404: o `themes.json` buscado pelo `ui.js` e os ícones `mask-image` do `admin.css`, que só carregam quando a regra é aplicada.
   - Se o próprio JS for despejado do cache, a verificação de geração feita em JS não roda.
   - Esse limite tem de ficar escrito no DEPLOYMENT.md e não pode ser tratado como resolvido.

3. **Rotas fora de `/web-assets` sem política definida.**
   - `/favicon.svg` (`engine/routes.go:190`) e `/public/*` (`engine/routes.go:194`) usam o mesmo `StaticHandler`.
   - Os templates usam `/public/favicon-96x96.png` (`live.html:8`, `shell.html:5`).
   - O `site.webmanifest` traz URLs internas, como o CSS. Ele precisa ficar com URLs fixas e `no-cache`+ETag ou ser reescrito, igual ao CSS.
   - Sem essa definição, o AC#1 fica ambíguo e a 106 pode sair com cache parcial.

4. **O caminho de erro do SSR herda o cabeçalho de sucesso.**
   - `Pages.render` (`internal/webui/handlers.go:238`) atende sucesso e `fail` 4xx/5xx ao mesmo tempo.
   - A política tem de ser decidida por `p.Status`: 200 recebe `private, no-cache`, e o resto recebe `no-store`.
   - O `http.Error` na falha de render também precisa de `no-store`.

5. **Nome inexistente em `asset` só aparece na execução.**
   - Hoje, um nome errado na função de template só falharia ao renderizar, gerando 500.
   - O `ValidateTemplates` deve ser ampliado para percorrer as chamadas `asset` com literal constante e falhar na inicialização. Argumento não literal também deve ser recusado.

## Melhorias arquiteturais

- **Catálogo** (`internal/webassets`): construir junto ao `Snapshot`, uma vez, dentro do `sync.Once`.
  - Cada entrada tem os bytes finais, o hash por arquivo, o MIME e um campo `public bool`.
  - O hash agregado (`assets.go:153`) fica só como geração e nunca é usado em URL.
  - O `themes.css` entra no catálogo gerado uma única vez.
  - O hash tem de ser calculado sobre os bytes *finais*, porque o resultado também depende de código Go (`themeNames`, `cssKey`), não só das paletas.
  - Isso substitui a duplicação entre `webui/assets.go:28-47` e `preview.go:234-253`.
- **Sem DAG por enquanto.** Hoje só existem referências CSS→folha (`admin.css:14`). O correto agora é:
  - recusar na inicialização `@import` e `url()` que aponte para `.css`;
  - recusar `url()` interno ausente do catálogo;
  - permitir apenas `data:` e as formas citadas entre aspas;
  - deixar a ordem topológica para a 068, se ela precisar. Fazer o DAG agora é YAGNI.
- **Handler único** baseado em `http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(b))`, com `ETag` definido antes da chamada.
  - Assim, `If-None-Match` (lista e `*`), `Range` e `If-Range` funcionam sem `FileServer`.
  - O 304 mantém `Cache-Control` e `ETag`.
  - Não inventar `Last-Modified`.
  - O mesmo handler é montado em `/web-assets`, `/preview-assets`, `/favicon.svg` e `/public`, mudando só o prefixo.
- **Regras de query:**
  - `RawQuery==""` recebe `no-cache`+ETag.
  - Exatamente um `v` igual ao hash do arquivo recebe `public, max-age=31536000, immutable`.
  - Qualquer outra query recebe 404 `no-store`.
  - Métodos diferentes de GET/HEAD recebem 405 `no-store`.
  - `LICENSE` e templates nunca entram no catálogo público.
- **ETag forte** no formato `"<sha256hex>"`. Se a 068 ou a 070 adicionarem compressão no Go, o ETag precisa variar por codificação (registrar como dependência).
- **Pré-visualização:** usar `data-themes-url` com a URL completa e versionada, e remover o fallback `'/preview-assets'` do `ui.js:11`.
- **Vary:** acrescentar `Cookie` com `Header().Add`, sem `Set`, porque o i18n já faz `Add("Vary","Accept-Language")` em `i18n.go:313`. O ganho é pequeno: com `private, no-cache` e sem validador, a página sempre volta inteira do servidor. Manter, mas sem contar como otimização.
- **Administração:** já é `no-store` pelo wrapper (`admin.go:168`); não mudar.
  - Recomendo acrescentar ao `admin.js`: `pageshow` com `persisted` faz `location.reload()`.
  - Motivo: desde 2025 o Chrome pode guardar no bfcache páginas `no-store` enquanto os cookies não mudarem. Uma sessão expirada sem mudança de cookie restauraria o painel antigo.
- **Retenção:** recomendo guardar **somente a geração atual**.
  - O DEPLOYMENT já diz que o SQLite exige parar a instância antiga. Logo não existe rolling deploy e o argumento de multi-instância não se aplica.
  - Guardar entre versões exigiria distribuir os assets antigos ou gravar em diretório gravável. Isso contraria a regra "trusted directory, separate from writable storage" (DEPLOYMENT.md:203).
  - **A decisão continua com o usuário.**

**Detecção de mudança ao voltar do bfcache.** A proposta de HEAD em segundo plano na URL atual é aceitável **somente em `pageshow.persisted`**. Como esse evento é raro, o custo de SSR e SQL fica limitado.

- A requisição usa `cache:'no-store'`, `credentials:'same-origin'` e um tempo limite com abort.
- Ela compara:
  - `Content-Language` com `<html lang>`;
  - o novo cabeçalho de geração com `data-ui-generation`;
  - o cookie `tz` com `data-timezone`, só quando o fuso não é forçado e não há `?tz`.
- Se algum cabeçalho faltar (o catálogo de idiomas é nil e `Wrap` não faz nada), **não** recarregar.
- O cookie de idioma é HttpOnly (`i18n.go:297`), então comparar só no cliente não basta para o idioma. Para o fuso, a comparação local dispensa requisição.
- O HEAD não deve ganhar atalho que diverge do GET (por exemplo, 200 para perfil inexistente).

## Plano de correção alternativo (ordem e ACs objetivos)

1. **`webassets`: catálogo e testes unitários.**
   - Hash por arquivo e `themes.css` com bytes estáveis.
   - Reescrita de `url()` com erro nos casos inválidos.
   - Mudar um byte de `admin.css` troca só o hash dele. Mudar um ícone troca o hash do ícone **e** o do `admin.css`.
2. **Handler único com testes Go.**
   - GET/HEAD com hash correto: 200 e `immutable`.
   - Sem `v`: `no-cache` e ETag.
   - `If-None-Match` igual, em lista e `*`: 304 com `Cache-Control` e `ETag`.
   - `Range` e `If-Range`: 206.
   - Hash antigo, `v` duplicado, query extra ou arquivo ausente: 404 `no-store` sem os bytes atuais.
   - POST: 405 `no-store`.
   - `LICENSE` e templates: 404.
   - Ajustar `engine/web_ui_default_027_test.go`.
3. **Função `asset` em public, admin e prévia**, com validação de literais na inicialização e `data-themes-url`. Teste: todo `href`/`src` local dos três shells sai versionado.
4. **SSR público por status:**
   - 200 GET/HEAD: `private, no-cache` com `Vary` contendo `Cookie` e `Accept-Language`;
   - 4xx/5xx: `no-store`;
   - admin, login, `/language` e APIs protegidas continuam `no-store`.
   Teste de tabela cobrindo as rotas.
5. **`live.js`:**
   - `pageshow.persisted` remove os controles de administração;
   - cria um controller novo a cada ativação;
   - consulta de novo `/user` com `no-store`;
   - não duplica handlers;
   - `AbortError` reabilita o botão.
   Junto entram o cabeçalho de geração (versão+`snapshotHash`) e o HEAD condicional.
6. **`admin.js`:** `pageshow.persisted` faz `reload`.
7. **Verificação nos navegadores.**
   - Chromium sem `page.route` e sem a flag padrão `--disable-back-forward-cache`, conferindo `pageshow.persisted===true` e `notRestoredReasons` vazio.
   - Firefox de versão real anotada: Juggler não vale como prova do AC#4.
   - Logout e sessão expirada no Chrome e no Firefox.
   - Segunda navegação com CSS e JS "(memory/disk cache)" no DevTools.
   - Atrás de proxy HTTPS, confirmar que a query e o `Cache-Control` sobrevivem.
8. **DEPLOYMENT.md:** retenção de uma geração e seu limite, proxy sem remover query nem reescrever cabeçalhos, e prévia.

**Ligação com outras issues:** a 068 reaproveita o catálogo e a reescrita, a 070 entra como folha WOFF2, a 075 herda o `pageshow`, e a 109 deve reutilizar o mesmo indicador de papel, sem duplicar `/user`.

**Decisões pendentes do usuário:**
- (a) ficar só com a geração atual e aceitar o limite descrito em "Falhas críticas" 2;
- (b) tratar `/public/*` e o `site.webmanifest`: versionar ou deixar com `no-cache`+ETag;
- (c) aceitar o custo do HEAD com SSR apenas em `persisted`.

Nada foi implementado e o aceite não foi declarado.
