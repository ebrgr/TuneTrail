**APROVADO**

Revisei só o `.gitattributes`, e é o único arquivo desse tipo no repositório.

- **Restrito aos snapshots:** o padrão `backlog/evidencias/issue-066/html-formatado/**` tem barra, então é ancorado na raiz do repositório. Ele vale só para os arquivos dentro de `html-formatado/`, incluindo `admin/`, `preview/` e `public/`. Os templates em `assets/templates/` e os outros arquivos de evidência, como os logs, o JSON e o `html-formatacao.md`, continuam com as verificações normais.
- **Mínimo:** quando o atributo `whitespace` recebe uma string, o Git parte das regras padrão e só aplica as mudanças listadas. Aqui só sai `blank-at-eol`. Continuam valendo `space-before-tab` e `blank-at-eof`. Atenção: `blank-at-eol` cobre espaços no fim de qualquer linha, não só de linhas vazias. Nesses fixtures exatos isso é aceitável.
- **Não esconde diffs nem altera bytes:** o atributo `whitespace` só afeta `git diff --check` e `git apply --whitespace`. Ele não usa `-diff`, `binary`, `text`, `eol` nem filtros, então não há normalização de checkout ou commit. O conteúdo continua aparecendo no diff, e a comparação byte a byte e o gate continuam como antes.
- **Comentário:** explica bem por que a regra existe.
