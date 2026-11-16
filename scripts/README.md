# Scripts

## Suíte e2e

`make e2e` constrói o binário e roda `scripts/e2e.py`, que executa as jornadas de ponta a ponta contra o binário real, cada uma com servidores em `127.0.0.1`, dados fictícios em diretório descartável e serviços externos desligados. Nenhuma lê `.env` nem o banco operacional. A suíte faz parte de `make check`, portanto do CI e de `make issue.review`/`issue.publish`.

```sh
make e2e                                               # todas as jornadas
python3 scripts/e2e.py --list                          # o que cada jornada cobre
python3 scripts/e2e.py --binary ./tunetrail --only admin public
```

| Jornada | Script |
| --- | --- |
| multiuser | `validate_multiuser.py` (também a biblioteca `Server` usada pelos demais) |
| admin | `validate_admin_webui.py` |
| public | `validate_webui.py` |
| permissions | `validate_permissions.py` |
| isolation | `validate_isolation.py` |
| migration | `validate_migration.py` |
| standalone | `validate_standalone.py` |
| legacy-env | `validate_standalone.py --legacy-env` (nomes `KOITO_*` descontinuados ainda iniciam, com aviso) |
| shutdown | `validate_shutdown.py` |
| vscode | `test_vscode_launch.py` |
| tools | `test_version.py`, `test_issue.py`, `test_roadmap06_fixture.py` |

Para acrescentar uma jornada, crie o script com `--binary` e diretório descartável e registre-o em `JOURNEYS` no `e2e.py`. Não use número de issue no nome: o histórico fica na tarefa.

## Ferramentas manuais

- `validate_container.py --image <imagem>` — sobe a imagem Docker com volume novo; exige Docker.
- `validate_upgrade.py` — atualização sobre uma **cópia somente leitura** de um banco legado; nunca aponte para o banco em uso.
- `validate_rollback.py --binary ./tunetrail --previous-binary <build 0.5>` — rollback para a última versão que lê `koito.db` e volta a partir do `.bak`, em dados fictícios (instruções de build no próprio script).
- `serve_roadmap06.py --binary ./tunetrail --work-dir <novo>` — mantém um servidor com a fixture fictícia para capturas.
- `capture_admin_webui.cjs` — capturas da administração em navegador.
- `validate_listen_dialogs.cjs <origem descartável> <evidências> <banco descartável>` — verifica confirmação e detalhes de escutas nas cinco superfícies de histórico, Yuu/Pearl, desktop/celular, foco/Tab/Esc, rede, exclusão, contagens, estado vazio, fallback sem JavaScript e expiração real da sessão; requer fixture nova de `serve_roadmap06.py` e `PLAYWRIGHT_MODULE`. O banco deve estar em `/tmp/tunetrail-141-*`; nunca use dados operacionais.
- `validate_media_dialogs.cjs <origem descartável> <diretório de evidências>` — verifica barra, diálogos, foco, atualização de cabeçalho/capa, rede, sessão e histórico em Chrome. Usa a fixture de `serve_roadmap06.py`; `PLAYWRIGHT_MODULE` indica o Playwright instalado. A comparação opcional com Koito em 4111 requer `REFERENCE_USER` e `REFERENCE_PASSWORD` em memória e faz somente login e capturas, sem mutações.
- `validate_media_dialog_refinements.cjs <origem descartável> <evidências>` — testa temas/contraste, fuso opcional, tooltips e buscas dinâmicas de artista/mesclagem; exige fixture nova por execução e os mesmos módulos/variáveis de referência do script anterior. Captura os diálogos em Yuu/Pearl e desktop/celular; no Koito faz apenas leitura e seleção de revisão, sem confirmar alterações.

## Scripts históricos

`scripts/historico/` guarda 30 scripts das issues 017–041 que comparavam a interface Go com o React removido ou capturavam evidências visuais pontuais. Eles não rodam na versão atual e não fazem parte da suíte; servem para reproduzir as evidências em `backlog/evidencias/`. Inventário e destino de cada um: `backlog/evidencias/e2e/ISSUE_063.md`.

Nomes antigos dos scripts mantidos (ISSUE-063):

| Antes | Agora |
| --- | --- |
| `validate_permissions_017.py` | `validate_permissions.py` |
| `validate_isolation_019.py` | `validate_isolation.py` |
| `validate_shutdown_020.py` | `validate_shutdown.py` |
| `validate_standalone_030.py` | `validate_standalone.py` |
| `validate_migration_030.py` (+ auxiliares do `_025`) | `validate_migration.py` |
| `test_vscode_launch_027.py` | `test_vscode_launch.py` |
| `validate_container_034.py` | `validate_container.py` |

## Ciclo das tarefas

`issue.py` automatiza o ciclo do Backlog.md descrito na skill `issue-first`:
`make issue.new`, `issue.start`, `issue.review` e `issue.publish`. A publicação
só aceita tarefa Done com comentário de aceite, roda `make check`, incrementa a
versão com `version.py`, cria a seção da versão no `CHANGELOG.md`, faz
fast-forward no `main` e cria a tag; `DRY=1` simula e `PUSH=1` envia.
`make issue.test` testa o fluxo em repositórios descartáveis.

## Referência de interface (ISSUE-065)

`make ui.reference OUT=/tmp/tunetrail-reference-new` exporta HTML/CSS/JS e assets de handlers reais com SQLite em memória e dados fictícios. O diretório deve ser novo; uma referência existente nunca é sobrescrita. Não lê `.env`, bancos ou sessões operacionais. O relógio das páginas públicas é fixo; a fixture agrega o interesse num intervalo fixo mantendo sua contagem real. Tokens CSRF/submissão/planos são mascarados só nos respectivos campos. A fixture da prévia usa os templates reais e células de calendário fixadas em 01/06/2024, sem alterar o relógio do runtime.

`make ui.compare BASE=/tmp/base NEW=/tmp/nova` compara os bytes exportados: saída 0 = idênticos; 1 = só indentação entre tags diferente (não é aceite visual); 2 = diferença de conteúdo ou arquivo ausente/extra. Espaços inline, CSS e ordem das regras são preservados na comparação. Metadados de rota também participam. A suíte `python3 -m unittest discover -s scripts -p test_compare_ui_reference.py` verifica o comparador.

Para capturar a referência exportada: `PLAYWRIGHT_MODULE=/caminho/node_modules/playwright node scripts/capture_ui_base.cjs /tmp/base /tmp/capturas`. Usa Chrome headless instalado (`CHROME_PATH` pode indicar outro Chromium), servidor efêmero em loopback, dados fictícios, seis larguras e dois temas. Não abre a instalação operacional. Requisições de fontes ao Google permanecem permitidas porque fazem parte do runtime atual; falhas são registradas. O manifesto inclui overflow e fontes efetivamente renderizadas via CDP, além dos estilos computados da retrospectiva em três larguras. Não confundir fallback por falha externa com o alvo de fontes locais da 070. Capturas não são teste automático de igualdade de pixels entre versões de navegador.

Para provar que uma reorganização do CSS manteve a cascata (ISSUE-068): `PLAYWRIGHT_MODULE=/caminho/node_modules/playwright node scripts/compare_computed_styles.cjs /tmp/base /tmp/nova /tmp/relatorio.json`, com duas referências exportadas por `make ui.reference`. Compara os estilos computados de todos os elementos (e de `::before`/`::after`) nas onze páginas, dois temas e cinco larguras, com fontes externas bloqueadas. Saída 0 = nenhuma diferença. Estados como hover, foco e diálogos abertos não são exercitados.

O `ui.css` servido é a concatenação, na ordem do `assets/manifest.json`, das partes em `assets/css/ui/` (`reset`, `tokens`, `base`, `layout/`, `components/`, `pages/`, `utilities`), cada uma dentro da sua `@layer`; a ordem das camadas é declarada no início de `reset.css`. O `admin.css` fica em `assets/css/admin/` na camada `pages`, salvo o `:focus-visible` genérico, mantido na `base` para continuar perdendo por especificidade para o foco do `ui.css`. Um `url()` dentro de uma parte é resolvido a partir do arquivo servido (`ui.css`, na raiz dos assets públicos), não da pasta da parte: escreva `url(fonts/x.woff2)` ou `url(/web-assets/fonts/x.woff2)`; um caminho que saia da raiz falha ao carregar os assets. Uma regra que precisa competir por especificidade com outra deve ficar na mesma camada que ela. `check_web_style.py` limita cada parte a 400 linhas.

## Distribuição externa (ISSUE-066)

`validate_web_distribution.py --binary ./tunetrail` integra a jornada `web-assets` de `make e2e`. Copia binário/assets/langs para diretório descartável, inicia via symlink com cwd `/`, verifica snapshot, headers e rotas protegidas e recusa raízes inválidas/binário sem assets antes de tocar no diretório de dados. As demais jornadas recebem os caminhos do checkout explicitamente; os nomes legados de configuração continuam sendo testados.

`make ui.reference.check` compara a renderização atual com os 113 arquivos de referência da ISSUE-106, com URLs versionadas e representações finais e faz parte de `make ui.check`/CI. O relógio e a versão da fixture ficam fixos em 2024/0.10 para que publicações futuras não invalidem a base apenas pelo número de versão. Somente tokens conhecidos são mascarados; mudanças reais de HTML/CSS continuam falhando.

A formatação dos templates na ISSUE-066 tem uma base complementar somente de HTML em `backlog/evidencias/issue-066/html-formatado/`. O gate daquela etapa usava `compare_ui_reference.py --html-baseline` para compará-la byte a byte e verificava os demais arquivos contra a base original da ISSUE-065.

`validate_web_cache.py` / `.cjs` verificam cache e histórico em sessões descartáveis dos navegadores reais (`make web.cache.check`). O Firefox usa WebDriver BiDi nativo; o proxy HTTPS confia apenas na chave do certificado efêmero dentro do perfil temporário do Chrome. O proxy registra o tráfego dos assets sem cookies/senhas. Requisitos e opções estão em DEVELOPMENT.md. O gate atual usa a referência completa da 106; o overlay da 066 permanece como evidência histórica.
