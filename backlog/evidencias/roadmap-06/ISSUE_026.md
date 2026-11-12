# Issue 026 — integração e proposta de aceite

Execução em 05/11/2026. [Issue](<../../tasks/iss-026 - 026-—-Integrar-evidências-e-preparar-decisão-visual.md>) ·
[plano](<../../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>) · [matriz planejada](ISSUE_022_MATRIZ.md).
Este relatório distingue testes executados, observações históricas e decisões
propostas. **O usuário aceitou o resultado e as ressalvas em 05/11/2026.** Na execução da 026, Go era o padrão e React exigia seleção explícita; a retirada posterior é tratada na 030. Referências
históricas das 023/024 ao padrão anterior foram superadas pela 027.

## Matriz consolidada

G/R = Go/React; V/A = visitante/administrador; D/M = 1440×1000/390×844;
Y/P/B = Yuu/Pearl/Black. Os relatórios e manifestos indicados discriminam as
capturas efetivas: as dimensões não significam produto cartesiano completo.
As jornadas visuais 023/024 são evidência anterior, não repetidas nesta execução.

| Casos | Jornada, ator, perfil e estado | Evidência e resultado | Limites / destino |
| --- | --- | --- | --- |
| P01–P02 | V; 0/1/N, Alpha/Beta/Empty; seleção, troca, retorno e duas abas | G/R, D/M e temas amostrados; seleção/troca em navegador e G sem JS. [023](ISSUE_023.md) | Não cobre todas as combinações de perfil/tema/viewport. |
| P03–P05 | V; rankings, histórico grande página 2 e busca pessoal Alpha/Beta | G/R com filtros e navegação; G sem JS em seleção/busca; [023](ISSUE_023.md) | Go usa página de busca e React modal; não se afirma cobertura de todos os filtros/empates na UI. |
| P06–P07 | V; artista/álbum/faixa, imagens e gráficos Alpha/Beta | [023](ISSUE_023.md): 192 intervalos API e 96 tabelas G com oráculo independente, UTC/São Paulo; capturas G/R | Gráfico React comparado visualmente, sem extração numérica de cada ponto. |
| P08–P09 | V; retrospectiva mensal/anual/vazia, imagem ausente e título longo | [023](ISSUE_023.md): G/R D/M; períodos, tempo e estados selecionados; G sem JS no controle testado | Na amostra de largura, React transbordou em 320/360 px em páginas indicadas na 023; Go sem transbordamento do documento nas páginas testadas. Fontes remotas bloqueadas. |
| A01–A02 | V/A; login, perfis, avatar, chaves e relay, bases fictícias separadas | [024](ISSUE_024.md): ações de navegador G/R; relay local; G sem JS nos casos informados. Permissões repetidas na 026 | Não equivale a todos os atores, telas e estados. |
| A03–A05 | A; import/export, edição, cancelamento, delete e merge | [024](ISSUE_024.md): ações selecionadas G/R; cinco formatos, export e planos obsoletos/repetidos por HTTP Go; integridade verificada | Cinco formatos e confirmação de merge não foram todos executados por cliques React; imagem React do merge mostra abertura, não confirmação. |
| X01 | V/A; teclado, contraste, nomes acessíveis, no-JS | [024](ISSUE_024.md): foco/Escape selecionados, nomes e contraste amostrados; Pearl botão Go 6,66:1 após correção | Sem leitor de tela completo, zoom real de 200%, auditoria formal ou contraste de todos os estados. 720×500 é apenas aproximação de espaço. |
| C01–C02 | V/A; rotas/assets e quatro bases v2 migradas para v3 | [025](ISSUE_025.md), repetida na 026: 50 combinações modo/método/caminho, backup/restauração em cópia, uma base com WAL, navegador G/R e admin G por HTTP/R por API | A falha React do `/theme-helper` foi corrigida e testada posteriormente na [028](ISSUE_028.md); Go retorna 404. Restauração não foi testada com binário antigo; fixture migrada é mínima. |

As [120 imagens públicas](screenshots-023.json) e [91 administrativas](screenshots-024.json)
incluem pares Y/P/B em D/M e estados adicionais; o Advisor confirmou 211 entradas
únicas e arquivos existentes. Isso não constitui aprovação visual automática.
Exemplos: [home Go](screenshots-023/go-home-yuu-mobile.jpg) / [React](screenshots-023/react-home-yuu-mobile.jpg),
[edição Go](screenshots-024/go-track-edit-yuu-mobile.jpg) / [React](screenshots-024/react-track-edit-yuu-mobile.jpg).
Outros nove temas tiveram verificação de aplicação, sem amostra suficiente de
legibilidade e contraste. Fallbacks de fonte diferiram: paridade tipográfica
permanece inconclusiva. Gráficos, cabeçalhos, datas e controles também diferem;
os relatórios 023/024 detalham as diferenças a avaliar.

## Regressões desta entrega

Na árvore local, com dados fictícios e servidores encerrados após os testes:

- `go test ./...`, `go vet ./...` e `go test -race` dos pacotes web, admin,
  listening, db, engine, relay e queue passaram; alguns resultados usaram cache.
- `npm run typecheck`, `npm test` (10 testes) e `npm run build` passaram.
- `validate_ui_modes_027.py` confirmou Go padrão, React explícito, rotas e v3;
  `test_vscode_launch_027.py` validou configuração F5, sem acionar a UI do VS Code.
- `validate_permissions_017.py`: 994 negativas G e 498 R/API, integridade e
  ausência dos marcadores testados nos logs. `validate_isolation_019.py`: duas
  abas, import/export, relay, cache, revogação, reinício e integridade nas fixtures.
- `validate_admin_webui.py`: formulários HTTP Go, cinco importadores, export,
  relay local, DST, merge obsoleto/repetido e integridade passaram.
- `validate_migration_025.py`: quatro migrações, backups, 50 combinações HTTP,
  navegador G/R, admin G/React API e integridade passaram; inicialmente reproduziu
  `THEME_HELPER=context-error`. Após correção na 028, a repetição encontrou o
  editor (`THEME_HELPER=editor-visible`); detalhes e testes adicionais na [028](ISSUE_028.md).

Base Git `72e4be2e582a67d83c37a339be30c94fe8ee8a5d`; havia muitas alterações
locais rastreadas e não rastreadas, de modo que o commit não identifica a árvore
integral. SHA-256 do binário de regressão:
`009878b955748fc3d2daefa8274b45479bfa8a293d4be66e415da0a579a8cd15`.
O hash identifica o binário, não as capturas históricas nem a árvore completa.

## Decisões propostas e pendências

| Tema | Proposta para decisão, não uma aprovação de comportamento não testado |
| --- | --- |
| 018: fusos e estatísticas | Manter documentados datas civis inteiramente removidas e offsets históricos manuais em segundos como limites; preservar a semântica de dias ativos históricos e contagem filtrada. Não se testaram todas as datas IANA. |
| 019/021: isolamento | Aceitar apenas o escopo dos cenários públicos e adaptadores testados; catálogo/imagens são globais e contexto interno sem perfil pode ler globalmente. Não afirmar prova universal de isolamento. |
| 020: import/export, relay e concorrência | Preservar contratos: importação pode ser parcial, exportação dinâmica e relay best-effort; cancelamento é cooperativo. Falha real de disco, reinício durante importação e interleavings não são exaustivos. |
| Fontes, temas, zoom e acessibilidade | Manter como verificação futura; as amostras não comprovam paridade tipográfica nem conformidade abrangente. |
| React e `/theme-helper` | Manter React disponível por seleção explícita. A [028](ISSUE_028.md) corrigiu e testou o helper com prévia fictícia; Go responde 404 para essa rota. |
| Distribuição e remoção React | Não propor remoção nesta issue. `/public` depende de `client/public`; build, Docker, CI e ferramentas ainda usam React; inventário em [025](ISSUE_025.md). Remoção requer proposta e autorização separadas. |
| Rollback | Selecionar React reverte a interface, não o esquema ou gravações; restauração de backup v2 exige plano próprio e perde alterações posteriores ao snapshot. |

Não há falha crítica de dados, autorização ou integridade **identificada nos
cenários testados**; isso não certifica casos não executados. Propõe-se ao usuário
aceitar o conjunto Go com as diferenças e limites acima; o
usuário pode rejeitar ou pedir mais verificações. Isso não autoriza retirar React.

## Parecer Advisor

[Advisor](../../../.agents/advisor.md), Sol esforço alto: **reprovado com ressalvas
para aceite integral**. Não identificou evidência nova de falha crítica de dados
ou autorização por leitura; não executou jornadas. Apontou necessidade desta
matriz, regressões, tratamento explícito da 028, fontes, nove temas, zoom real,
limites da UI React e rastreabilidade. O relatório e as regressões respondem à
consolidação; as lacunas funcionais/visuais permanecem expressas acima. A revisão
não equivale a aceite integral, nem substitui a decisão do usuário. Após ler este
relatório e os resultados comunicados, o Advisor **aprovou a apresentação do
aceite condicionado**, sem reexecutar as regressões e sem identificar novo
bloqueio documental. O fechamento continua condicionado ao aceite do usuário.

## Aceite posterior

O usuário aceitou a 026 com as ressalvas acima e autorizou separadamente a retirada do React em 05/11/2026. A [issue 030](<../../tasks/iss-030 - 030-—-Retirar-React-e-consolidar-interface-Go.md>) registra essa alteração posterior.
