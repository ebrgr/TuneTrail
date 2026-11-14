**APROVADO.**

Os três bloqueios do parecer anterior estão corrigidos. O diff e o log `/tmp/issue066-html-uncached-check.log` confirmam isso.

## Bloqueios anteriores, conferidos

- **`internal/webui/graphs_issue021_test.go:18`**: a regex agora é `<td>([0-9]+)</td>\s*</tr>`.
- **`engine/i18n_ui_test.go:42`**: agora usa `<noscript>\s*<details class="language-settings-fallback">`. A posição vem de `FindStringIndex`, então a busca seguinte por `</noscript>` continua começando no lugar certo.
- **`cmd/api/settings_integration_test.go`**:
  - em `:257`, a regex `"\s*>` aceita whitespace antes do `>`, e o resto da mensagem continua sendo comparado exatamente, com `QuoteMeta`;
  - em `:396` e `:437`, `</dt>\s*<dd>` também aceita whitespace.
  - A asserção `"<code>TUNETRAIL_LOG_LEVEL</code>. "` continua passando, porque `{{else}}. {{end}}{{tr` ainda gera ". To edit" sem quebra de linha.
- **`navigation_issue023_test.go`**: `\s+` entre `href` e `aria-label` corresponde ao novo layout com um atributo por linha.

## Evidência no log

- O log mostra todos os pacotes com `ok` e o gate de referência com `exit_code: 0`.
- Os pacotes de UI estão `ok`, e o e2e deu 12/12.
- Com `-count=1` nos três alvos (normal, race e UI), o resultado não pode ter vindo do cache. Isso resolve a dúvida sobre `os.Root`.

## Ação necessária no commit

`backlog/evidencias/issue-066/html-formatado/` ainda não está no git. Agora o `ui.reference.check` usa esse diretório via `--html-baseline`. Se ele ficar fora do commit, o `make check` falha num clone limpo. Os dois JSONs de verificação também estão fora do git e devem entrar junto.

## Notas opcionais (não bloqueiam)

- Em `navigation_issue023_test.go`, as strings passaram a ser regex. Por isso o `.` de `JavaScript.` agora aceita qualquer caractere. O efeito é desprezível.
- Em `rewind.html`, o `</div>` que fecha `rewind-group-items` fica depois do range. Se um grupo vier sem itens, ele fecha o `rewind-group-content` antes da hora. Isso já acontecia antes da reformatação e agora está documentado no comentário.
