# ISSUE-063 — Inventário dos scripts de validação por issue

**Data:** 13/11/2026 · **Binário:** TuneTrail 0.8 (`make api.build`) · **Suíte:** `scripts/e2e.py` (`make e2e`, parte de `make check`)

## Método

Cada um dos 37 scripts com sufixo de issue (`*_0NN.py`, `*_0NN.cjs`) foi lido e, quando era um validador Python executável com `--binary`, rodado contra o binário atual em diretório descartável. Os `.cjs` usam navegador (Playwright) e, com exceção de três, comparavam a interface Go com o React removido na ISSUE-030; eram evidências visuais pontuais, não testes de regressão.

Resultado da execução antes da reorganização: passaram `validate_permissions_017`, `validate_isolation_019`, `validate_shutdown_020`, `validate_standalone_030`, `validate_migration_030` e `test_vscode_launch_027`; falharam `validate_migration_025` (sobe o modo React), `validate_ui_modes_027` e `validate_theme_helper_028` (React). Não foram rodados: `validate_container_034` (Docker), `validate_graphs_023` (exige servidores Go e React), `validate_images_033` (exige build legado) e `capture_sidebar_029.py` (navegador).

Fora dos 37, dois scripts sem sufixo estavam quebrados sem que ninguém percebesse, porque não rodavam no CI: `validate_webui.py` lia `koito.db` em vez de `tunetrail.db` e ainda subia um servidor React; `serve_roadmap06.py` tentava iniciar o modo React. Ambos foram corrigidos.

## Destino de cada script

| Script | Destino | Motivo |
| --- | --- | --- |
| `audit_accessibility_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_accessibility_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_admin_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_admin_states_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_below_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_images_033.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_public_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_settings_041.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_sidebar_029.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `capture_sidebar_029.py` | Histórico → `scripts/historico/` | captura da barra lateral em navegador na transição Go/React |
| `capture_states_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `contrast_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `keyboard_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `test_modal_a11y_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `test_vscode_launch_027.py` | Mantido → `scripts/test_vscode_launch.py` | jornada `vscode` da suíte e2e; passou em 13/11/2026 |
| `validate_admin_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_admin_io_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_container_034.py` | Mantido → `scripts/validate_container.py` | ferramenta manual (exige Docker); o job `container` do CI cobre a partida da imagem |
| `validate_graphs_023.py` | Histórico → `scripts/historico/` | compara gráficos Go e React (exige os dois servidores); intervalos de interesse seguem cobertos por validate_webui.py |
| `validate_images_033.py` | Histórico → `scripts/historico/` | compara derivados de imagem do worker legado libvips e do Go; exige build legado |
| `validate_isolation_019.py` | Mantido → `scripts/validate_isolation.py` | jornada `isolation` da suíte e2e; passou em 13/11/2026 |
| `validate_layout_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_media_react_024.cjs` | Histórico → `scripts/historico/` | edição de apelidos no React |
| `validate_migration_025.py` | Histórico → `scripts/historico/` | migração Go + React; a parte Go e os auxiliares migrate/route_matrix foram levados para validate_migration.py |
| `validate_migration_030.py` | Mantido → `scripts/validate_migration.py` | jornada `migration` da suíte e2e; passou em 13/11/2026 |
| `validate_migration_browser_025.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_pagination_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_permissions_017.py` | Mantido → `scripts/validate_permissions.py` | jornada `permissions` da suíte e2e; passou em 13/11/2026 |
| `validate_profiles_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_public_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_relay_ui_024.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_rewind_023.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_shutdown_020.py` | Mantido → `scripts/validate_shutdown.py` | jornada `shutdown` da suíte e2e; passou em 13/11/2026 |
| `validate_standalone_030.py` | Mantido → `scripts/validate_standalone.py` | jornada `standalone` da suíte e2e; passou em 13/11/2026 |
| `validate_theme_helper_028.cjs` | Histórico → `scripts/historico/` | captura/inspeção em navegador para a evidência visual da issue; substituída pela referência da ISSUE-065 |
| `validate_theme_helper_028.py` | Histórico → `scripts/historico/` | regressão do editor de tema React; falha desde a remoção do React |
| `validate_ui_modes_027.py` | Histórico → `scripts/historico/` | alternância Go/React; falha desde a remoção do React (KOITO_WEB_UI=react recusa iniciar, coberto por validate_standalone.py) |

## Suíte e2e

| Jornada | Script | Cobre |
| --- | --- | --- |
| multiuser | `validate_multiuser.py` | bancos novo e migrado: administração, chaves, ingestão concorrente, leitura pública, importação/exportação, revogação, reinício |
| admin | `validate_admin_webui.py` | administração HTML: login, CSRF/origem, perfis, chaves, relay, formatos de importação, edição e revisão do catálogo |
| public | `validate_webui.py` | páginas Go: escopo por perfil, paridade API/HTML, filtros inválidos, mídia, retrospectiva, busca, limites de interesse |
| permissions | `validate_permissions.py` | autorização, sessões, origem e tratamento de segredos |
| isolation | `validate_isolation.py` | isolamento entre perfis em escutas, chaves, caches e agregados |
| migration | `validate_migration.py` | migrações v2→v3 (com WAL), restauração do backup, matriz de rotas e jornadas após migrar |
| standalone | `validate_standalone.py` | binário a partir de diretório vazio |
| shutdown | `validate_shutdown.py` | encerramento com receptor de relay travado |
| vscode | `test_vscode_launch.py` | configuração de execução do VS Code usa o servidor Go |
| tools | `test_version`, `test_issue`, `test_roadmap06_fixture` | testes unitários dos scripts de versão, do fluxo de tarefas e da fixture |

Todas as jornadas usam servidores em `127.0.0.1`, dados fictícios e MusicBrainz, Deezer e Cover Art Archive desligados; nenhuma lê `.env` nem o banco operacional. Execução em 13/11/2026: **10/10 jornadas aprovadas em cerca de 14 s** (sem contar o build).

## Escopo

As capturas em navegador não fazem parte da suíte: a referência visual passa a ser a da ISSUE-065. Os scripts históricos são mantidos para reprodução das evidências antigas e não são executados.
