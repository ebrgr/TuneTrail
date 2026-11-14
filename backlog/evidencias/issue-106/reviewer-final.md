**APROVADO.** Nada bloqueia o commit.

## Itens do parecer anterior (`/tmp/issue106-reviewer.txt`)

| # | Item | Situação |
|---|---|---|
| 1 | Atributos `data-timezone`/`data-week-start` duplicados | Corrigido. Cada um aparece uma vez em `live.html:35-36`, há um teste em `TestDocumentAssetMetadata` e a referência foi regenerada (113 arquivos, `differences: []`). |
| 2 | Recarga em toda volta do histórico na seleção de perfil | Corrigido de outro jeito, e o jeito novo é melhor. `handlers.go:252-255` só envia `X-TuneTrail-Timezone`/`Week-Start` quando há perfil. Sem esses headers, `live.js:92-93` não compara nada. Os dois valores, no header e no atributo, vêm do mesmo `p` em `render`, e isso vale também para `serveMedia` (`media.go:42`) e `serveRewind` (`rewind.go:79`). Por isso não há como os dois divergirem. O roteiro do navegador cobre a seleção com semana igual a 1. |
| 3 | `scripts/README.md` desatualizado | Corrigido. |
| 4 | Comentários nas funções novas | Corrigido em `live.js` e em `validate_web_cache.cjs`. |
| Sugestões | Timeout do DELETE, fuso desconhecido, `image-set`/`src`, idioma das docs | Todas incorporadas. O timeout agora mostra o status e não repete o DELETE. O fuso é comparado com o valor resolvido pelo servidor e há teste para isso. `image-set`/`src` são recusados, com 3 casos de teste. A seção do `DEVELOPMENT.md` está em português. |

`data-assets` foi mantido de propósito: cinco scripts de validação (`validate_webui.py`, `validate_multiuser.py` e outros) usam esse atributo para reconhecer o modo Go. Aceito.

## Verificado nesta rodada

- **Fallback de capa:** `ui.js:82` usa `data-cover-fallback-url` com hash. Não sobrou nenhuma referência a `dataset.assets` em `assets/js`. O roteiro real confirma que `naturalWidth > 0` e que a URL tem `?v=`.
- **Concorrência em `live.js`:**
  - Ao abortar no `pagehide`, `active` vira `null` e o `catch` encerra sem mostrar nada.
  - O timeout com a mesma ativação ainda válida mostra o status.
  - O `requestScope` sempre aborta as requisições irmãs no `finally`.
  - O HEAD feito no restore envia cookies (`same-origin`), então o servidor resolve o mesmo contexto.
  - Se o HEAD falhar (offline, por exemplo), o `catch` remove os controles, e o Chrome testou esse caso.
- **Cache e headers:**
  - O header `Vary` combina `Accept-Language` (`i18n.go:313`) com `Cookie`.
  - Não há middleware de compressão no `engine`, então o ETag forte é coerente.
  - Respostas 412/416 passam pelo `assetResponse`.
  - No preview, `GET /preview-assets/cover/{id}` tem precedência sobre o mount genérico.
- **Evidências:** conferi os quatro logs.
  - `make check`: ok, incluindo a referência, `node --check`, os testes de tema e o e2e com 12 de 12 jornadas.
  - Chrome 157, Firefox 147 e HTTPS: PASS.

## Sugestões (não bloqueiam)

1. **Baixa, `assets/js/live.js:143`:** depois de `location.reload()` (linha 129), o `finally` reativa o botão por um instante, até a página descarregar. Um clique nesse intervalo abre outro `confirm`. Correção: guardar uma flag `reloading` e pular o `disabled = false` quando ela estiver marcada.
2. **Legibilidade, `internal/webassets/assets.go:282`:** o `|| path.Base(name) == "LICENSE"` nunca é alcançado, porque as linhas 275-277 já tratam esse caso. Pode remover.
3. **Legibilidade, `assets/js/theme.js:12`:** sobrou uma linha em branco no lugar do código que foi para `refresh()`. Pode remover.
4. **Legibilidade, `assets/js/admin.js:14-19`:** agora existem dois listeners de `pageshow`, o novo e o que reseta `form[data-pending]`. Podem virar um só, com o reset no ramo não persistido.
5. **Documentação, `assets/templates/public/live.html:33`:** vale um comentário curto, em template, avisando que `data-assets` só existe como marcador para os validadores. Assim ninguém remove o atributo achando que é código morto.

Nenhum desses pontos muda o comportamento que os testes de cache e histórico verificam. Não é preciso rodar os navegadores de novo se algum deles for aplicado.
