# Parecer do Reviewer (Claude Opus) — ISSUE-110

Primeira rodada: **um item obrigatório**, resto aprovado.

1. **Obrigatório — `assets/js/theme-picker.js` (atalho `\`).** `showPopover()` sem botão de origem deixava o foco fora do painel: leitor de tela não anunciava o diálogo e o teclado precisava atravessar a página. **Corrigido:** o atalho guarda o elemento focado, move o foco para o tema ativo (ou o botão de fechar) e devolve o foco no `toggle` de fechamento. `painel_tema.py` passou a verificar esse caminho (foco entra no painel e volta ao link ao fechar).

Sem problemas: ids únicos por página; prévia mantém `feature-dialog` e o painel; nenhum atributo removido do `<body>` ainda é lido; atalhos `\` e `/` consistentes; cores do `palettes.json` no contexto CSS do html/template; chaves i18n removidas sem uso restante; CSS antigo removido por completo.

Sugestões opcionais:
1. `Info.svg` sem uso — mantido: volta como "Sobre" na ISSUE-123.
2. `ui.js_required` sem uso — **removida**.
3. `ThemeOptions` refazia o parse a cada página — **aplicada** (cache por geração do snapshot).
4. Navegadores sem `popover` (Safari < 17, Firefox < 125) ficam sem o seletor de idioma — aceito: são versões de 2023, anteriores ao Baseline do `popover`; o painel fica oculto em vez de quebrar a página.
5. Selo por `(not .Login)` — conferido: o login é a única tela da administração sem sessão.

Observação: uma execução do `painel_tema.py` excedeu o tempo limite sem mensagem; as três seguintes, já com tempo limite por ação, passaram. Não se reproduziu.
