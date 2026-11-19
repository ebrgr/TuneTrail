# ISSUE-075 — implementação e verificação

Executor Codex, 18/11/2026. Escopo reduzido, sem delegação. Status **Em andamento**; aceite visual do usuário pendente.

## Comportamento

- View Transitions declarativas em navegação completa; `prefers-reduced-motion: reduce` troca `navigation` para `none` e remove a animação. Nenhuma interceptação de formulário, substituição de `main`, alteração de `history` ou fragmento novo.
- JSON inline `type=speculationrules`, somente `prefetch`, `source=document`, `eagerness=moderate`. Allowlist exata: `/{slug}`, `/{slug}/listens`, três rankings `/{slug}/chart/top/{artists,albums,tracks}` e mídia `/{slug}/{artist,album,track}/{id numérico}`. URLPattern herda a origem do documento. `search=""` exclui **todas** as queries, inclusive filtros legítimos (opção conservadora). Downloads, links com target e rel external são excluídos pelo seletor.
- Sem prefixo amplo nem wildcard de caminho: gestão, logout, exclusão, admin, APIs, outros perfis/origens, subrotas e queries de ação não entram. Search e Rewind também ficam fora da allowlist conservadora.
- Conferência das rotas: `engine/routes.go` encaminha GET público para `webui.Pages.ServeHTTP`; `internal/webui/handlers.go` despacha Home, rankings e histórico para leituras; `internal/webui/media.go` lê mídia/atividade/interesse. Ações de gestão e POST pertencem a outro handler e não são candidatas. HEAD conserva os mesmos contratos. Não foram alteradas autenticação, autorização, rotas ou páginas de gestão.
- Regras somente em documentos 200 de perfil e **sem cookie de sessão**, independentemente do papel da conta. Se uma aba anônima permanecer aberta durante login em outra aba, um prefetch com cookie recebe **204, no-store, corpo vazio** antes de consultar sessão ou renderizar HTML. Prova em `session-prefetch-headers.txt` e teste de regressão. Nenhum gate de visibilidade da ISSUE-161 foi antecipado.
- Headers do projeto e resposta real não têm CSP; portanto não existe política de nonce/hash a ampliar. O navegador aceitou o JSON sem avisos de sintaxe/CSP. `public-headers.txt` registra os headers. Não foi acrescentado `unsafe-inline`, nonce, hash ou relaxamento de segurança. Uma CSP `script-src 'none'` bloqueia também as regras, preservando navegação normal.
- Cache da ISSUE-106 preservado: documentos públicos continuam `private, no-cache`, `Vary: Cookie` (e idioma), geração da UI e headers de estado existentes. Assets continuam com seus contratos de cache. Prefetch nunca executa scripts da página antecipadamente.
- Cada intervalo do interesse tem um círculo SVG com `<title>` contendo início, fim, inclusão do último limite e contagem. Tooltip HTML/CSS usa o mesmo `data-tip` e superfície compartilhada do heatmap; não cria paradas de foco. `details`, captions, cabeçalhos e tabelas continuam disponíveis; heatmap existente preservado.

## Validação

- **`make check`: aprovado**, incluindo Go vet/testes, verificações CSS/JS/templates, referência determinística e **13/13 jornadas E2E**. Log: `make-check.log`. A primeira tentativa apontou uma linha de template com 166 caracteres; corrigida antes da rodada final.
- Testes novos em `internal/webui/navigation_075_test.go`: JSON real renderizado válido; allowlist exata sem URLs proibidas; perfil correto; exclusão de cookie de conta inclusive vazio/inválido; ausência em seleção/erro; cache intacto; corrida de login/prefetch; títulos, tooltips e tabela/details por intervalo.
- `speculation-browser.json`: URLPattern real no Chromium do Orca aceita as seis amostras públicas e rejeita dez amostras de outro perfil/origem, ação, gestão/admin/APIs e query. Exclusão de download validada no JSON e teste Go. Não é uma alegação de que o navegador decidiu efetivamente baixar cada candidato: prefetch permanece opcional e depende das heurísticas do navegador.
- `reduced-motion.json`: emulação nativa no Orca, media query verdadeira, regra `navigation:none`, animação calculada `none`; preferência restaurada depois.
- JavaScript: **51.786 bytes brutos; 17.458 bytes gzip**, soma por arquivo de todos os `assets/js/*.js`, nível 9, mtime zero. **Zero alterações em assets/js**. `javascript-budget.json` detalha a medição; JSON declarativo não é JavaScript executável.
- `make web.cache.check` extra: **Chrome 157 passou**. A etapa Firefox interrompeu com `SyntaxError: missing ) in parenthetical` no avaliador BiDi existente (`scripts/validate_web_cache.cjs:129`, chamado em :264 com duas instruções dentro de uma expressão). Não foi alterado esse verificador fora do escopo. Resultado completo da matriz Chrome/Firefox/HTTPS **não aprovado**; ver `cache-check.log`. A rodada HTTPS separada **passou no Chrome 157** (`cache-https.log` e `cache-https-browser.json`).
- Grafo AST atualizado com `analisar-tunetrail.sh "$PWD"`: 4.272 nós, 18.526 arestas, sem API/extração semântica. Avisos esperados sobre tokens.css e scripts exportados sem símbolos. Diff revisado diretamente; `git diff --check` aprovado para código e evidências autorais. O export determinístico conserva whitespace gerado dos templates/arquivos SVG existentes; o hook de commit o sinaliza, sem modificar essa saída para não invalidar a referência.

## Capturas pelo Orca

Dados fictícios de `scripts/serve_roadmap06.py`: Alpha 7, Beta 2, Empty 0 escutas; bancos/configurações descartáveis em `/tmp`. Nenhum dado pessoal ou sessão real foi usado nas requisições registradas. Para evitar cookies antigos de outras fixtures em 127.0.0.1, as capturas anônimas usam localhost.

- `track-desktop-tooltip.png`: 1440×1000, tooltip CSS com período e quantidade; títulos também expostos na árvore de acessibilidade. `tooltip-state.json` registra opacity/conteúdo após a transição CSS.
- `track-320-no-script-graphs.png`: 320×900, gráficos e details, largura do documento 320px. `no-script-state.json`: 16 títulos, dois details e `TuneTrailEnhance` ausente.
- `track-320-no-script-table.png`: tabela aberta com foco no summary e Enter, 17 linhas (cabeçalho + 16 intervalos). `no-script-details.json` registra abertura/linhas e ausência de scripts.
- `search-320-no-script.png`: formulário de busca enviado normalmente, dois resultados. `no-script-search.json` registra `q=Artista`; `no-script-back.json` registra retorno à URL anterior por Voltar (o navegador preserva o valor editado do campo, comportamento nativo).

Bloqueio de scripts: proxy HTTP local de GETs da fixture, acrescentando `Content-Security-Policy: script-src 'none'`. O proxy não altera HTML, formulários, cache ou CSS. A instrumentação Orca apenas lê estado, posiciona foco/rolagem e envia teclas/cliques; scripts do aplicativo e Speculation Rules permanecem bloqueados. Isso comprova as jornadas representativas acima, **não uma matriz completa de todas as jornadas em navegador sem suporte**.

## Referência e critérios

`Makefile` agora compara com `ISSUE-075/reference`, exportada por `make ui.reference`; a referência da ISSUE-073 foi preservada. `reference-changes.txt` registra as 25 diferenças esperadas: CSS compartilhado, círculos/títulos/tooltips e JSON público, URLs com novos hashes e geração global da UI. HTML administrativo exportado só acompanha hashes/geração; nenhuma página de gestão foi implementada ou modificada. Os arquivos JS copiados no export são idênticos aos existentes.

- AC #2 comprovado e marcado.
- AC #1 **não marcado**: E2E e jornadas públicas representativas sem scripts passaram, mas não foi comprovada a totalidade em navegadores sem suporte; a matriz extra do Firefox tem a limitação acima.
- AC #3 **não marcado**: implementação, tabelas e capturas verificadas; aceite visual pertence ao usuário.

Referências técnicas consultadas: [HTML Standard — Speculative loading](https://html.spec.whatwg.org/multipage/speculative-loading.html) (predicados, URLPattern e JSON) e [Chrome — Speculation Rules improvements](https://developer.chrome.com/blog/speculation-rules-improvements) (document rules e origem relativa). A política aplicada é deliberadamente mais restrita que o conjunto suportado pelo padrão.
