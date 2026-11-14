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
