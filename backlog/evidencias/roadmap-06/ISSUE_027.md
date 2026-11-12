# Issue 027 — Go padrão e F5 no VS Code

Implementada em 05/11/2026, por solicitação do usuário. A configuração anterior
do F5 aguardava a tarefa `Koito: frontend dev` e abria `localhost:5173`, embora
o usuário quisesse iniciar a interface Go. Isso foi identificado por leitura de
`.vscode/launch.json`; não se observou a sessão original do VS Code.

A configuração principal agora inicia o depurador Go, define `KOITO_WEB_UI=go`,
não usa `preLaunchTask` e abre `http://127.0.0.1:<porta>` a partir do log de
prontidão. O Vite antigo desta pasta foi encerrado e a porta 5173 ficou livre.
O perfil separado de React configura `KOITO_WEB_UI=react` explicitamente; ele
não é iniciado pelo F5 principal. Arquivos de lançamento e tarefas do VS Code
passaram a ser compartilháveis, enquanto as demais preferências continuam ignoradas.

Sem configuração de modo, a aplicação seleciona Go. No modo Go, rotas não
reconhecidas não caem na SPA React; retornam 404. O favicon foi incorporado aos
assets Go para funcionar sem o build React. `KOITO_WEB_UI=react` continua sendo
uma opção explícita e usa a SPA existente. React não foi removido; as comparações
e o aceite visual da etapa 6 continuam pendentes. O esquema SQLite continua v3.

## Testes

- Regressões de configuração e roteamento reproduziram o padrão React e o fallback
  antigo antes da mudança; depois passaram para Go padrão, favicon, rotas e 404.
- `go test ./...`, `go test -race ./internal/cfg ./engine ./internal/webui` e
  `go vet` dos pacotes alterados passaram; algumas suítes usaram cache.
- `validate_ui_modes_027.py` passou com Go em diretório vazio sem build React e
  com React explicitamente selecionado, verificando favicon, assets, fallback e
  integridade SQLite v3 em bases descartáveis.
- `validate_webui.py`, `validate_admin_webui.py` e `validate_multiuser.py`
  passaram; este último inclui banco novo e migração v2→v3. O validador
  multiusuário foi adaptado para não presumir que a interface seja uma SPA.
- `test_vscode_launch_027.py` verificou o perfil F5, a ausência de tarefa Vite e
  o endereço derivado do log. Delve 1.27.2 iniciou a aplicação em uma base
  descartável; o HTML Go respondeu e o padrão de prontidão extraiu a porta.
  Isso valida Delve e a configuração, mas não é uma execução do F5 pela interface
  do VS Code do usuário.
- Advisor (Sol, esforço médio) aprovou a separação dos modos e recomendou favicon
  independente, cobertura de GET/HEAD e manutenção das rotas React explícitas.

Nenhum banco operacional foi usado nos testes. Para aplicar a configuração local,
encerre a sessão de depuração anterior, selecione **Koito: Desenvolvimento Go
(localhost:4110)** e pressione F5; o endereço esperado é `http://127.0.0.1:4110`.
