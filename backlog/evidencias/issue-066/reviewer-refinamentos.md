**APROVADO**

As três sugestões estão aplicadas corretamente e não encontrei falhas reais.

- **Nome do chamador no diagnóstico** (`internal/webassets/templates.go:48`): o erro passa a sair como `template "caller": undefined or bodyless template "absent"`. O `%w` mantém o erro original acessível para `errors.Is`/`As`. Os testes têm só um template com corpo em cada caso, então a ordem aleatória de `Templates()` não deixa o resultado instável.
- **Ramos else** (`templates.go:31,36,41`): `visit(n.ElseList)` cobre `if`, `range` e `with`, e `ListNode` nil volta sem erro (`templates.go:15`). Os três novos subtestes `* else` exercitam cada um desses caminhos.
- **Template associado sem corpo** (`templates.go:24`): o projeto usa `html/template`, em que `template.New("admin")` já registra "admin" no conjunto de templates com `Tree == nil`. Depois do `Parse` de "caller", "admin" continua no conjunto com `Tree` nil, e `Lookup("admin")` o retorna. Então `TestTemplateReferenceWithoutBody` passa de fato pelo teste `target.Tree == nil` e não pelo `target == nil`. A regressão fica coberta de verdade.

Esta revisão foi só por leitura: não rodei `go test ./internal/webassets/` e vale rodar antes do commit.
