# Parecer do Advisor (Claude Opus, effort medium) — ISSUE-110

**REPROVADO COM RESSALVAS.** Direção correta; quatro lacunas com correção simples, sem mudar o escopo.

## Falhas críticas
1. A grade de temas (troca, `aria-pressed`, Redefinir, JSON personalizado) vive em `ui.js`, que o admin não carrega. Extrair para `js/theme-picker.js` (defer em todos os tipos de head, entrada no manifesto) e remover o `<select data-admin-theme>` de `admin.js`.
2. "Sair" não pode ser formulário nas páginas públicas (sem token CSRF). Conta vira link para `/admin?section=account`. O `POST /language` exige exatamente `language` e `return_to` (`i18n.go:283`); não acrescentar campos.
3. Sem suporte a `popover`, o painel aparece inline. CSS `[popover]{display:none}` + `[popover]:popover-open{display:block}`; checagem mínima `'popover' in HTMLElement.prototype` para a engrenagem navegar a `/admin`.
4. `engine/i18n_ui_test.go:42-60` procura `settings-dialog`, `admin-settings` e o `<noscript>` de idioma; reescrever na mesma etapa.

## Melhorias
- Tema personalizado: manter recolhido em `<details>` no fim do modal (é escolha de tema; `theme.js` aplica o `custom-theme` salvo). O admin passa a tê-lo.
- `popover="auto"` (Esc, fechar ao clicar fora, foco devolvido); evitar `commandfor`/`show-modal` e CSS anchor positioning.
- `<noscript>` curto no painel de tema; remover os `<noscript>` de idioma do rodapé.

## Desenho
- Struct comum `webassets.Nav` (`internal/webassets/nav.go`), método `Nav()` em `Page`, `LivePage` (explícito, por causa de `Admin`) e no `view` do admin; `{{template "partial/sidebar" .Nav}}`.
- `assets/templates/shared/nav.html` (sidebar, avatar, language-picker, theme-panel, admin-menu), montado nos dois conjuntos.
- `icon`, `initials`, `avatarIcon` em webassets; o admin deixa de usar SVG inline e máscaras.
- `ThemeOption`/paletas movidas para webassets.

## Passos
1. Subtarefa só de refatoração (Nav, nav.html, funções compartilhadas), provada pela referência.
2. ISSUE-110: theme-picker.js, Palette + painel de tema, menu de administração com selo, remoção do settings-dialog, do #admin-settings e dos noscripts.
3. CSS de popover e fallback; atalho `\`.
4. i18n, testes, scripts de validação, CHANGELOG, capturas desktop e celular.

## Perguntas ao usuário
1. Separar a refatoração da barra numa ISSUE própria?
2. Na prévia, o menu de administração mostra os links ou um aviso?
3. Manter o atalho `\` abrindo o modal de tema?
