# Issue 024 — administração, ações e acessibilidade

Execução em 05/11/2026. [Plano](<../../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>) ·
[matriz](ISSUE_022_MATRIZ.md) · [manifesto de capturas](screenshots-024.json).
Usamos Chrome 156.0.8078.80, Node 24.15.0, servidores loopback e cópias
SQLite v3 com perfis e credenciais fictícios. As capturas finais usam bases
novas equivalentes antes das mutações; informações sensíveis foram mascaradas ou
mantidas fechadas. Fontes externas foram bloqueadas e a paridade tipográfica
permanece inconclusiva. A árvore tinha alterações locais não commitadas.

## Correções

- No tema Pearl, o botão principal administrativo Go tinha contraste medido de
  **2,69:1**; a regressão de navegador falhou antes da alteração. Após o ajuste
  de CSS, a mesma amostra passou a **6,66:1**. Links e contorno de foco também
  receberam cor mais escura no tema. Amostras dos botões Yuu e Black foram
  respectivamente 6,87:1 e 8,93:1. Isso não é auditoria de todos os componentes,
  temas, estados ou combinações de cores.
- No editor React de mídia, campos de alias, artista e MusicBrainz dependiam de
  placeholders, e botões de remoção exibiam apenas ícones sem nome acessível. A
  regressão falhou antes da correção; agora cada campo/ação tem nome, inclusive
  a etapa de confirmação. Não foi adicionada dependência ou comportamento
  JavaScript novo na interface: foram usados atributos acessíveis e o estado de
  confirmação que já existia.

## Jornadas executadas

- Login pela interface, alteração de nome do perfil e preservação da contagem do
  outro perfil nos dois modos. Em Go, login, edição de alias e cancelamento de
  exclusão global funcionaram com JavaScript desativado; foi verificado que o
  alias do catálogo compartilhado aparecia também no outro perfil.
- Go e React: criação/revogação de chave, upload de avatar e importação Spotify
  pela interface; exportação validada via HTTP e confirmação de que o outro
  perfil continuava com duas escutas. Tokens e chaves não foram impressos ou
  fotografados. Criação/remoção de alias React em duas etapas foi verificada pela
  interface e no catálogo compartilhado, sem alterar o histórico Beta.
- Relay Go e React: testar rascunho sem salvá-lo, salvar e remover pela interface,
  com receptor fictício local e sem encaminhamento a serviço externo.
- Validador HTTP `validate_admin_webui.py`: login/retorno, origem/CSRF, HEAD,
  repetição de ticket, perfis/chaves, relay local, cinco formatos de importação,
  export/round-trip, edição, DST manual, exclusão pessoal, merge obsoleto e
  repetido, zero perfis, senha e integridade. Os cinco formatos e a confirmação
  de merge foram verificados pelo validador HTTP Go, não por cliques React.
- `validate_permissions_017.py`: 994 negativas Go e 498 React/API, integridade e
  ausência dos marcadores fictícios nos logs testados. `validate_isolation_019.py`
  repetiu jornadas de duas abas, chaves, relay, cache, import/export e reinício
  nas fixtures. Não equivale ao produto cartesiano de todos os atores/estados.
- Teclado: skip-link Go visível e destino correto sem JS; Escape e retorno de foco
  no diálogo de busca Go e no modal de edição React. Amostra automática de campos
  e botões visíveis em administração, edição de faixa e home não encontrou nome
  ausente segundo a verificação limitada do script. O modal React foi testado à
  parte após carregar dados; não se alegam todos os estados de todos os modais.
- Viewport móvel 390×844 e amostra 720×500 não tiveram transbordamento do
  documento nessas páginas. A segunda medida é uma **aproximação de espaço**,
  não teste de zoom real de 200%. Tabelas internas, percurso completo de leitor
  de tela e conformidade formal de acessibilidade permanecem para a 026.

## Verificações técnicas

`go test ./...`, `go test -race ./internal/webui ./internal/adminui
./internal/adminweb ./engine`, `go vet ./...`, `npm run typecheck`, `npm run test`
(10 testes) e `npm run build` passaram. Alguns testes Go usaram cache. Os scripts
Playwright de regressão verificaram contraste, nome acessível e confirmação do
botão sem remover dados. `PRAGMA integrity_check` e `foreign_key_check` passaram
nas cópias de navegador; processos temporários foram encerrados.

## Comparação visual

[91 imagens e metadados](screenshots-024.json): 44 pares e três evidências
individuais de foco/no-JS/modal; os campos de senha, tokens e códigos foram
mascarados onde presentes. Pares em 1440×1000 e 390×844, temas Yuu/Pearl/Black
para login, perfis, chaves, import/export, relay e edição de faixa; pares Yuu
adicionais para edição de artista/álbum, revisão de exclusão e mesclagem.

| Estado | Go | React |
| --- | --- | --- |
| Login, celular Pearl | [Go](screenshots-024/go-login-pearl-mobile.jpg) | [React](screenshots-024/react-login-pearl-mobile.jpg) |
| Perfis, desktop Yuu | [Go](screenshots-024/go-profiles-yuu-desktop.jpg) | [React](screenshots-024/react-profiles-yuu-desktop.jpg) |
| Chaves, desktop Yuu | [Go](screenshots-024/go-keys-yuu-desktop.jpg) | [React](screenshots-024/react-keys-yuu-desktop.jpg) |
| Importação, celular Black | [Go](screenshots-024/go-import-black-mobile.jpg) | [React](screenshots-024/react-import-black-mobile.jpg) |
| Edição de faixa, celular Yuu | [Go](screenshots-024/go-track-edit-yuu-mobile.jpg) | [React](screenshots-024/react-track-edit-yuu-mobile.jpg) |
| Revisão de mesclagem, desktop Yuu | [Go](screenshots-024/go-merge-review-yuu-desktop.jpg) | [React](screenshots-024/react-merge-review-yuu-desktop.jpg) |

[Editor Go sem JS](screenshots-024/go-nojs-editor-yuu-mobile.jpg) ·
[Foco no skip-link](screenshots-024/go-keyboard-skip-yuu-mobile.jpg) ·
[Modal de edição React](screenshots-024/react-edit-modal-yuu-desktop.jpg).
As interfaces mantêm diferenças: Go usa documentos e revisão explícita com
impacto, React usa modais e seções expansíveis. As diferenças são registradas para
aceite posterior na 026, não tratadas como autorização automática de remoção React.
A captura de mesclagem React mostra abertura do modal, não uma operação confirmada.

## Limitações e encaminhamentos

Não foi feita auditoria independente completa por leitor de tela, zoom real,
contraste de todos os estados ou cliques em todas as combinações de edição e
exclusão nos dois modos. As operações não cobertas por UI foram distinguidas do
validador HTTP acima. O caso de arquivos com falha de disco e jornadas web/admin
sobre bases migradas permanece para a 025; diferenças visuais, fontes efetivamente
carregadas e aceite do usuário permanecem para a 026. React continua padrão e
fallback. As atividades não usaram dados operacionais nem serviços musicais reais.
