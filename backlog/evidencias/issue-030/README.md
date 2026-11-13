# Issue 030 — retirada do React

Em 05/11/2026, após o aceite explícito da [026](<../../tasks/issue-026 - 026-—-Integrar-evidências-e-preparar-decisão-visual.md>), o cliente React foi removido. O servidor fornece apenas a interface Go; `KOITO_WEB_UI` aceita ausência ou `go` e recusa `react` com mensagem explícita. A resposta `/apis/web/v1/config` mantém `web_ui: go`. SQLite v3, migrações e rotas da API permanecem; `/theme-helper`, exclusivo do React, retorna 404.

Os 11 assets antes em `client/public` foram incorporados ao binário e continuam disponíveis sob `/public/`; os caminhos dos ícones e o início do manifesto foram ajustados. A prévia Go continua disponível sem o servidor de referência React. Build, Docker, CI, VS Code e verificação de versão não dependem do cliente React. Os scripts antigos de comparação foram preservados como histórico e identificados em [scripts/README.md](../../../scripts/README.md).

## Verificação

- `make check`: Go, vet e testes dos scripts locais. `make multiuser.check`: instalação nova e migração v2, login, escutas, perfis, import/export e integridade em bancos descartáveis.
- `go test -race ./internal/cfg ./internal/webui ./engine` passou.
- `validate_migration_030.py`: três migrações independentes, incluindo WAL confirmado, backup e restauração em cópia, 25 verificações de método/rota, formulários administrativos, jornada API e integridade. A restauração não foi executada com binário antigo.
- `validate_permissions_017.py`: 994 negativas Go e integridade; `validate_isolation_019.py`: perfis, duas abas, import/export, relay, cache, reinício e integridade nos cenários testados.
- `validate_standalone_030.py`: servidor em diretório inicialmente vazio, assets públicos, manifesto e ícones, 404 para rotas removidas e SQLite v3. `test_vscode_launch_027.py` e verificação de versão passaram.
- `docker build -t koito-issue030-local .` passou; um contêiner descartável respondeu com a interface Go, administração, assets e ícones do manifesto, e retornou 404 para `/theme-helper`. O contêiner foi encerrado após o teste.
- Navegador com dados fictícios e fontes externas bloqueadas: [página pública desktop](after-public-desktop.png), [conta desktop](after-account-desktop.png), [perfil desktop](after-profile-desktop.png), [página pública celular](after-public-mobile.png), [conta celular](after-account-mobile.png) e [perfil celular](after-profile-mobile.png). Foram conferidos navegação, preservação de perfil, tema, foco e caminhos sem JavaScript.
- Advisor (Sol, esforço médio): aprovou a implementação por leitura após corrigir os caminhos do manifesto e ampliar as verificações. Não executou testes.

As limitações de fontes, acessibilidade, fusos e concorrência registradas na 026 permanecem. Este trabalho não prova equivalência visual integral nem compatibilidade com binário anterior sobre backup restaurado.
