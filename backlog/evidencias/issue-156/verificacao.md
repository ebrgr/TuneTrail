# ISSUE-156 — verificação

Implementação da alternativa B aprovada na ISSUE-155. Revisão própria do diff com os critérios de reviewer/deslop; nenhuma delegação. Parecer: [review.md](review.md).

- `make check`: verde, incluindo `ui.check`, testes Go, vet, JavaScript, comparação exata da referência e 12/12 jornadas E2E.
- `make ui.check`: executado também isoladamente; saída em `ui-check.txt`.
- `PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright node scripts/validate_navigation_071.cjs backlog/evidencias/issue-156/reference <saída>`: 270 vistas aprovadas; resultados em `navigation.json`.
- Matriz: visitante, admin e prévia; Yuu/Pearl; com e sem JS; ícones/rótulos; larguras 320, 360, 390, 639, 640, 641, 768, 1023, 1024, 1025, 1440 e 1920; adicionalmente 320×480, 640×320 e 844×390.
- Verificados: grupos e ações por papel, seleção atual, links com profile, erro 429 antes da autenticação, consulta de período/fuso preservada, ausência da segunda barra, margem invariável 58/224, painel limitado, textos visíveis, teclado, retorno de foco do tema, alvos de 44px, expansão nativa na árvore AX e footer.
- Orca: inspeção visual e confirmação de retorno do foco ao invocador visível após fechar o tema com More recolhido, inclusive na administração. Seis capturas, somente menu (+) aberto em Yuu: `menu-yuu-{visitante,admin}-{390,768,1440}.png`. Fixtures fictícias, sem dados reais.
- Referência regenerada em `reference/`, preservando as evidências anteriores. `ui.reference.check` aponta para esta referência.
- Grafo AST atualizado fora do repositório.

ACs 1, 2 e 4 verificados. AC3: teclado, foco, links sem JS e AX automatizado verificados; leitura manual com VoiceOver/outro leitor de tela ainda pendente, portanto não marcado. AC5: capturas prontas; aceite visual é do usuário e não foi marcado.

Nenhum Sair adicionado ao menu; o formulário administrativo existente continua disponível. Nenhuma persistência no conf.toml ou limpeza da chave antiga no armazenamento. Sem push, merge, publicação ou Done.
