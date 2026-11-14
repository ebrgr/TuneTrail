# Revisão da ISSUE-066: reformatação dos 12 templates HTML

**Veredito: NÃO APROVADO.** A lógica Go, os atributos, o texto e o gate estão corretos. Mas a reformatação quebra três testes Go que não fazem parte do gate de referência. Cheguei a isso lendo os testes contra os templates novos; não executei os testes, porque o papel de Reviewer exclui editar e rodar. A evidência descrita cobre só o gate e o Chrome, não `go test ./...`.

## Problemas necessários

Os três testes procuram trechos de HTML colados. Agora há quebra de linha e indentação entre as tags, então a busca deixa de encontrar o trecho.

1. **`internal/webui/graphs_issue021_test.go:18`**: a regex exige `<td>([0-9]+)</td></tr>`. Em `graphs.html`, `</td>` e `</tr>` agora ficam em linhas separadas. A regex não acha nenhuma linha e o teste falha com `no rows in Activity values` nos três perfis.

2. **`engine/i18n_ui_test.go:41`** (`TestLanguageJourney`): procura `<noscript><details class="language-settings-fallback">`. Em `live.html` e `admin/shell.html`, agora há quebra de linha e `{{if .Locale}}` entre `<noscript>` e `<details>`. O teste falha com `no-JS settings missing`. Ele roda sempre, porque `newProfileTestApp` não tem skip.

3. **`cmd/api/settings_integration_test.go`** (`TestSettingsHTTPJourney`, também sem skip):
   - `:396` e `:437` procuram `<dt>Running</dt><dd>…</dd>`; agora há quebra de linha entre `</dt>` e `<dd>` em `settings.html`.
   - `:257` procura `server-setting-override-deprecated">Set by environment variable …`; o `>` do `<p>` foi para a linha seguinte, então surge whitespace entre a aspa e o `>`.

**Correção recomendada:** tornar as asserções tolerantes a whitespace entre tags. Por exemplo: `</td>\s*</tr>`, `</dt>\s*<dd>`, `<noscript>\s*<details …>`, e `"\s*>Set by` em `:257`, que precisa virar regex. Isso mantém a intenção estrutural dos testes e combina com o objetivo da issue. A alternativa é colar de novo essas tags nos templates e regerar as fixtures. Nos dois casos, rode `make check` e `go test ./...` completos antes do aceite.

## Sugestões (não bloqueiam)

1. **Chamadas aninhadas achatadas pioram a leitura.** Alguns trechos ficaram menos legíveis que o original, o que contradiz o objetivo da issue:
   - `(tr\n(printf\n"admin.action.%s"\n.Plan.Action)\n.Plan.Action)` em `catalog-review.html` e `manage.html`;
   - a condição do `{{if or (eq $key …) …}}` em `live-filters`, que virou 20 linhas;
   - o `tr2` de `interest-graph`.

   Mantenha cada subexpressão `(…)` numa linha só. O texto gerado não muda e as fixtures continuam válidas.
2. **Indentação estranha que sobrou da preservação de espaços:**
   - `   ·` no rodapé de `live.html` e `   Minutes`, `   Plays` etc. em `page.html`, com um espaço a mais;
   - `name="{{.Name}}"{{if .Overridden}}\n disabled{{end}}` em `settings.html`;
   - `{{if .Profile}}data-profile-id=…\n data-profile-slug=…{{end}}` no `<body>` de `live.html`.

   É seguro, mas confunde. No rodapé, `TuneTrail {{.Version}} · {{tr …}}` numa linha só ficaria mais claro.
3. **`rewind.html`:** o `<div class="rewind-group-items">` abre dentro de `{{if eq $i 0}}` e fecha depois do `{{end}}` do range. A estrutura veio do original, mas a indentação nova sugere outro aninhamento. Vale um comentário `{{/* … */}}` ou alinhar a indentação.
4. **Gate:** o overlay não confere se tem exatamente os 23 HTML esperados. Ter menos arquivos não é perigoso, porque o gate compara com a base 065 e falharia. Uma contagem fixa no teste documentaria a intenção.

## O que conferi e está correto

- **Comparador e gate:**
  - o overlay recusa arquivos que não são `.html` ou que não existem na base, e os assets continuam comparados à 065;
  - uma diferença `normalized` retorna exit 1, que também falha no `make`;
  - o novo teste cobre os casos de recusa.
- **Ações multilinha no Go:** são válidas com Go 1.25. A regex de `catalog_ci_test.go` usa `\s+`, então ainda encontra as chaves de tradução nas chamadas `{{trn\n "…"`.
- **Whitespace novo em locais sensíveis:**
  - `.avatar` é `inline-flex`, então os espaços em volta de `<img>` e do fallback não aparecem;
  - `admin.js` e `live.js` não usam `childNodes`, `firstChild` nem `textContent` na marcação renderizada;
  - o CSS não usa `:empty`;
  - `<textarea>` e `<style>` inline continuam intactos.
- **Outras asserções:** continuam passando:
  - `">Now Playing"` em `media_test.go`;
  - a regex `(?s)<select…disabled` em `:206`;
  - a linha de `<option …>` em `settings_test.go:84`, que não mudou.
- **Texto e espaços inline:** os pontos frágeis estão preservados. Isso vale para `{{else}}. {{end}}`, `.<br>{{else}}`, `</b> Minutes` (o espaço vira quebra de linha mas continua visível) e o `{{-`/`-}}` do avatar no admin.
