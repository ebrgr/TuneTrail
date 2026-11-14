Condição final UTC aplicada e verificada; ver comparison.txt.

**Não aprovado ainda.** Há um item impeditivo, que se resolve com uma linha. O resto está correto ou é limitação já documentada.

### Item impeditivo

**1. O baseline muda conforme o fuso horário e a data da máquina (`Makefile:128`, alvo `ui.reference`)**
- `internal/adminui/settings.go:104-106` monta `LocalTimezone` com `time.Now()`. Esse valor aparece em `admin/settings.html` pelo template `settings.html:9`. O teste não controla esse relógio, e a máscara `volatile` não cobre esse texto.
- `/empty` é exportado sem `?tz=`, e `TimezoneWithForce` (`internal/webquery/legacy.go`) usa `time.Local` quando a requisição não traz fuso.
- Consequência: um baseline e um candidato gerados em máquinas com TZ diferente, ou em lados opostos de uma troca de horário de verão, divergem sem nenhuma mudança de código, e `ui.compare` termina com código 2. As duas execuções de `/tmp/issue065-finalcompare.json` saíram idênticas porque rodaram na mesma máquina e no mesmo dia, então não detectariam isso.
- Correção sugerida: definir `TZ=UTC` na receita do `ui.reference`, ao lado de `TUNETRAIL_UI_REFERENCE=...`. Opcionalmente, trocar a rota `"empty": "/empty"` por `"/empty?tz=UTC"`, como já é feito nas outras rotas públicas.

### Verificado, sem defeito
- **`reference_test.go`**: o diretório de saída precisa ser novo (`os.Mkdir`). As escutas são fixas, com `listened_at=1714564800`, e substituem as que `pageStore` grava com `time.Now()`. As células do preview usam a data fixa 2024-06-01 e a mesma fórmula de `makePage`. O cookie de sessão é escolhido pelo nome e descarta cookies vazios ou expirados. A lista de rotas é ordenada antes de gravar `routes.json`. Os arquivos de `static/public/` são exportados também na raiz, e o avatar vai para `image/preview-photo/128x128.webp`. A máscara cobre só os tokens voláteis dos formulários.
- **`compare_ui_reference.py`**: compara o conjunto de todos os arquivos das duas árvores. Arquivo ausente ou extra conta como diferença, e a normalização vale só para HTML. Os códigos de saída são 0 (idêntico), 1 (só indentação, apenas informativo) e 2 (outra diferença).
- **`test_compare_ui_reference.py`**: cobre os quatro modos de comparação, HTML ausente, CSS extra e fonte extra. O import alternativo funciona com `discover -s scripts`.
- **`Makefile`**: a segunda regra `ui.check: ui.reference.test` só acrescenta um pré-requisito, então o alvo continua um só. O teste unitário do comparador passa a rodar no `ui.check`, e portanto no `check`.
- **`capture_ui_base.cjs`**: o servidor só responde em loopback e bloqueia caminhos fora da pasta exportada. Só a origem local e as fontes do Google são liberadas. Imagens, folhas de estilo e scripts locais que retornem 404 fazem a captura falhar, com exceção de `/image/*`, cujo fallback é permitido. O tema aplicado é conferido em cada página, o relógio do navegador é fixo e a rewind usa `fullPage`. As fontes vêm de `CSS.getPlatformFontsForNode` via CDP, ou seja, são as realmente renderizadas. Não há fontes locais em `static/`, então não checar 404 de fontes locais não perde nada.

### Limitações documentadas (não bloqueiam)
- No fixture de render, `GetInterest` devolve um único bucket agregado; a distribuição real do gráfico de interesse não é exercitada.
- A captura cobre 11 páginas representativas das 23 páginas HTML exportadas.
- Diferenças só de indentação dão código 1, que é informativo e não serve como aprovação visual.

Depois de aplicar `TZ=UTC`, gere novamente o baseline. Com isso, o parecer é **APROVADO**.

Observação: não rodei o `graphify query` que o `CLAUDE.md` pede antes da leitura, porque não tenho shell nesta sessão. Li direto os cinco arquivos indicados e consultei de forma pontual `preview.go`, `handlers_test.go`, `adminui/settings.go`, `adminui/admin.go` e `webquery/legacy.go` para confirmar de onde vem o horário. Não modifiquei nada e não abri `estudos-locais`.
