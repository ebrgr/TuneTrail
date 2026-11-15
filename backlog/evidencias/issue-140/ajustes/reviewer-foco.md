A lógica central está correta. A precedência de `||`/`&&` dá o resultado certo, e quando o foco está no `h2` (fora de `controls`) o Tab vai para o primeiro controle e o Shift+Tab para o último. Encontrei quatro problemas reais e um ajuste de acessibilidade.

## Problemas

1. **`element.disabled` não detecta `<fieldset disabled>`.** Um botão dentro de um fieldset desabilitado tem `.disabled === false`, mas não recebe foco. Use `element.matches(':disabled')`.

2. **Elementos com `tabindex="-1"` entram na lista.** Os seletores `a[href]`, `button` etc. pegam também `<button tabindex="-1">`. Se esse elemento for o último da lista, o último elemento que realmente recebe foco nunca é igual a `last`. O Tab então não é interceptado e o foco sai do diálogo, indo para a barra do navegador com `showModal()`. Filtre por `element.tabIndex < 0`. Isso também permite trocar `[tabindex="0"]` por `[tabindex]`.

3. **Modificadores.** Alt+Tab, Ctrl+Tab e Cmd+Tab também disparam `key === 'Tab'` em alguns contextos. Ignore o evento quando houver modificador.

4. **Foco perdido ao aplicar `inert`.** Se o foco está no botão de envio e o formulário fica `inert`, o navegador move o foco para `<body>`. O trap se recupera no próximo Tab, mas o leitor de tela perde o contexto nesse intervalo. Chame `focusTitle()` *antes* de aplicar `inert` ao formulário.

Observação: com `showModal()`, o próprio navegador já torna inerte o que está fora do diálogo. O trap só impede a saída para a interface do navegador, o que é uma escolha válida. Com `show()` (não modal), ele é indispensável.

## Versão revisada

```js
const FOCUSABLE = 'a[href],button,input:not([type=hidden]),select,textarea,summary,[tabindex]';

// Recomputed on every Tab: disabled/inert/details state changes while the dialog is open.
function isTabbable(element) {
  // :disabled also covers controls inside <fieldset disabled>; tabIndex < 0 drops tabindex="-1".
  if (element.tabIndex < 0 || element.matches(':disabled') || element.closest('[inert]')) return false;
  // Chromium retains layout boxes for inputs inside closed details; those are not focusable.
  return element.checkVisibility({visibilityProperty: true});
}

dialog.addEventListener('keydown', event => {
  if (event.key !== 'Tab' || event.altKey || event.ctrlKey || event.metaKey) return;
  const controls = [...dialog.querySelectorAll(FOCUSABLE)].filter(isTabbable);
  const first = controls[0];
  const last = controls.at(-1);
  if (!first) { event.preventDefault(); focusTitle(); return; }
  const active = document.activeElement;
  const wrap = !controls.includes(active) ||
    (event.shiftKey && active === first) ||
    (!event.shiftKey && active === last);
  if (wrap) {
    event.preventDefault();
    (event.shiftKey ? last : first).focus({preventScroll: true});
  }
});
```

No ponto onde as requisições começam, inverta a ordem:

```js
focusTitle();          // move focus out before the form becomes inert
form.inert = true;
```

Se o diálogo tiver `iframe`, `audio[controls]`, `video[controls]` ou `[contenteditable]`, inclua esses seletores em `FOCUSABLE`. `checkVisibility({visibilityProperty})` exige Safari 17.4 ou mais recente. Se precisar suportar versões anteriores, use `getClientRects().length > 0` como fallback.

Quer que eu aplique isso em `assets/js/media-dialog.js` dentro da ISSUE-140?
