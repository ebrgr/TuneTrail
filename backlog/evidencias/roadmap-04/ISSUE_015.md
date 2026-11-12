# Issue 015 — integração e aceite técnico da administração Go

Implementada e aceita pelo usuário em 05/11/2026; issues 010–015 fechadas e resolvidas. [Issue](<../../tasks/iss-015 - 015-—-Integrar-administração-Go-e-registrar-jornadas-e-aceite-visual.md>) · [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

`KOITO_WEB_UI=go` agora entrega `/admin`, gestão por perfil, login/conta, chaves,
relay, importação/exportação, edição/capas/escutas e revisão de exclusão/merge.
O dispatcher compõe os documentos e actions explicitamente; API, imagens e assets
não viram páginas. React continua padrão, com seu build e fallback preservados.
Os manifests de [documentos](../../../test_assets/web-ui-routes.json) e
[métodos](../../../test_assets/admin-web-ui-routes.json) são testados em Go e TS.

O login HTML/JSON compartilha o limite global existente de dez tentativas/minuto.
Todas as respostas administrativas, inclusive 429, usam no-store. Os forms
mantêm sessão/CSRF e acrescentam ticket de uso único vinculado à sessão; duas
abas possuem tickets independentes. Isso evita repetição de mutações com ou sem
JavaScript. Reinício ou expiração de ticket exige recarregar o formulário.

A composição administrativa usa as fontes Jost/League Spartan, paletas e assets
públicos, com barra lateral, campos e cards responsivos. Preferências de tema
são preservadas. Busca global de artistas e destinos está atrás da sessão.
Campos são reapresentados em erro; DST ambíguo recebe instrução específica.
O fuso efetivo é respeitado também na confirmação de escuta. O link Manage do
histórico seleciona o registro exato sem JavaScript, retorna aos filtros e ajusta
uma página que tenha ficado vazia após exclusão. Detalhes de mídia retornam para
histórico com o filtro correspondente, incluindo semana ISO e fuso.

## Testes e evidências

- `go test ./...`, testes `-race` dos componentes administrativos, SQLite e
  páginas Go, e `go vet ./internal/adminui ./internal/adminweb ./internal/webui ./engine/...`: passaram.
- Cliente: typecheck, dez testes e build de produção passaram.
- `validate_multiuser.py`: bases nova e migrada v2→v3, concorrência, import/export,
  exclusões, reinício e integridade passaram.
- `validate_admin_webui.py`: servidor real, sessão/retorno, HEAD, origem/CSRF,
  ticket ausente e repetido, perfil/chave/relay local, cinco formatos, round-trip,
  campo inválido, DST, escuta exata, retorno com semana/paginação, busca/merge,
  plano obsoleto/repetido, zero perfis e troca de senha passaram. O orçamento de
  login foi testado em processos novos, nas duas ordens HTML/JSON.
- [Relatório das jornadas](issue015-journeys.json) · [regressão pública](issue015-public-regression.json).
- Chrome/Playwright: login→editor→ação sem JS, confirmação de registro histórico,
  skip-link por teclado, duas abas, clipboard negado com cópia manual e
  DataTransfer de imagem seguido de envio do formulário passaram.
- Advisor (Sol) aprovou os contratos da integração após revisão de rotas,
  limitador, nonces, CSRF e retornos. Não houve acesso ao banco operacional.

## Comparação visual

**55 imagens: 26 pares React/Go e três evidências de interação.** O
[manifesto](screenshots.json) informa rota, perfil fictício, modo, tema, fuso,
viewport e região. São recortes por viewport; a captura longa do Chrome produzia
tiling inadequado e foi descartada. Os pares usam cópias equivalentes de SQLite
v3; interações que alteram dados ocorrem depois das capturas pareadas.

| Estado Yuu | React desktop | Go desktop | React celular | Go celular |
| --- | --- | --- | --- | --- |
| Login | [React](screenshots/react-login-yuu-desktop.png) | [Go](screenshots/go-login-yuu-desktop.png) | [React](screenshots/react-login-yuu-mobile.png) | [Go](screenshots/go-login-yuu-mobile.png) |
| Perfis | [React](screenshots/react-profiles-yuu-desktop.png) | [Go](screenshots/go-profiles-yuu-desktop.png) | [React](screenshots/react-profiles-yuu-mobile.png) | [Go](screenshots/go-profiles-yuu-mobile.png) |
| Conta | [React](screenshots/react-account-yuu-desktop.png) | [Go](screenshots/go-account-yuu-desktop.png) | [React](screenshots/react-account-yuu-mobile.png) | [Go](screenshots/go-account-yuu-mobile.png) |
| Chaves ocultas | [React](screenshots/react-keys-yuu-desktop.png) | [Go](screenshots/go-keys-yuu-desktop.png) | [React](screenshots/react-keys-yuu-mobile.png) | [Go](screenshots/go-keys-yuu-mobile.png) |
| Importação/exportação | [React](screenshots/react-import-yuu-desktop.png) | [Go](screenshots/go-import-yuu-desktop.png) | [React](screenshots/react-import-yuu-mobile.png) | [Go](screenshots/go-import-yuu-mobile.png) |
| Relay | [React](screenshots/react-relay-yuu-desktop.png) | [Go](screenshots/go-relay-yuu-desktop.png) | [React](screenshots/react-relay-yuu-mobile.png) | [Go](screenshots/go-relay-yuu-mobile.png) |
| Exclusão de perfil | [React](screenshots/react-profile-delete-yuu-desktop.png) | [Go](screenshots/go-profile-delete-yuu-desktop.png) | [React](screenshots/react-profile-delete-yuu-mobile.png) | [Go](screenshots/go-profile-delete-yuu-mobile.png) |
| Editar artista | [React](screenshots/react-artist-edit-yuu-desktop.png) | [Go](screenshots/go-artist-edit-yuu-desktop.png) | [React](screenshots/react-artist-edit-yuu-mobile.png) | [Go](screenshots/go-artist-edit-yuu-mobile.png) |
| Editar álbum | [React](screenshots/react-album-edit-yuu-desktop.png) | [Go](screenshots/go-album-edit-yuu-desktop.png) | [React](screenshots/react-album-edit-yuu-mobile.png) | [Go](screenshots/go-album-edit-yuu-mobile.png) |
| Editar faixa | [React](screenshots/react-track-edit-yuu-desktop.png) | [Go](screenshots/go-track-edit-yuu-desktop.png) | [React](screenshots/react-track-edit-yuu-mobile.png) | [Go](screenshots/go-track-edit-yuu-mobile.png) |
| Mesclagem | [React](screenshots/react-merge-yuu-desktop.png) | [Go](screenshots/go-merge-yuu-desktop.png) | [React](screenshots/react-merge-yuu-mobile.png) | [Go](screenshots/go-merge-yuu-mobile.png) |

Pearl: perfis [React desktop](screenshots/react-profiles-pearl-desktop.png),
[Go desktop](screenshots/go-profiles-pearl-desktop.png),
[React celular](screenshots/react-profiles-pearl-mobile.png),
[Go celular](screenshots/go-profiles-pearl-mobile.png); editor de faixa
[React desktop](screenshots/react-track-edit-pearl-desktop.png),
[Go desktop](screenshots/go-track-edit-pearl-desktop.png),
[React celular](screenshots/react-track-edit-pearl-mobile.png),
[Go celular](screenshots/go-track-edit-pearl-mobile.png).

Interações: [editor sem JS](screenshots/go-nojs-editor-mobile.png),
[confirmação sem JS](screenshots/go-nojs-listen-confirm-mobile.png),
[foco de teclado](screenshots/go-keyboard-focus-desktop.png).

As 52 capturas pareadas não mostraram overflow horizontal nos viewports usados.
A comparação confirma fontes/paletas legíveis, acesso abaixo da dobra e estados
funcionais. **Não representa paridade visual exata:** React usa modais e navegação
superior em celular; Go usa documentos, sidebar lateral e forms mais extensos.
A conta Go tem rota própria; o React a mantém em detalhes. As revisões Go mostram
impacto global mais detalhado. Esses desvios estão visíveis nos pares e foram aceitos
pelo usuário em 05/11/2026, preservando a revisão visual transversal da etapa 6. Não houve medição de desempenho ou auditoria completa de
acessibilidade.

## Limites preservados

Importação continua síncrona/parcial e exportação é uma leitura dinâmica, sem
snapshot. Planos de catálogo usam fingerprint global conservador e podem exigir
revisão por mudança não relacionada; não há medição em bases grandes. Tokens e
planos são locais ao processo, sem implantação distribuída. Offsets históricos
com segundos não são suportados. Cenários de encerramento abrupto durante I/O
remoto e carga prolongada continuam parte da validação transversal do roadmap 5.
React, migrações 001–003 e esquema SQLite v3 foram preservados. Nenhuma issue foi
publicada e nenhum push foi realizado.
