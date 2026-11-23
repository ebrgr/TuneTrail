# Matriz multiusuário

TuneTrail 0.72, schema 22. Instâncias descartáveis em loopback. A edição do
perfil passou pelo formulário real no Chrome com JavaScript ligado e desligado.
Os testes Go cobrem as rotas brutas, cabeçalhos, transições e serviços
diretos. Os resultados abaixo combinam essas verificações.

| Papel e alvo | HTML | JSON | Serviços `admin.*` |
| --- | --- | --- | --- |
| Visitante, perfil público | Diretório e escutas 200. Ação de gestão sem sessão 401. | Listagem e escutas 200. Gestão 401, inclusive com chave de ingestão. | Sem sessão humana, gestão recusada com `ErrUnauthorized`. |
| Visitante, perfil protegido | Diretório 200 com indicador. Escutas 404, mesmo documento do perfil ausente. | Escutas e derivados 404. | Gestão exige sessão humana também neste estado. |
| Comum, perfil próprio público | Leitura 200. Pode abrir Conta e Perfil, editar o próprio perfil e gerir histórico e chaves. | Leitura e gestão próprias autorizadas. Operações globais 403. | Pode editar perfil e avatar, gerir chaves, exportar histórico e remover relay próprios. |
| Comum, perfil próprio protegido | Leitura 200. Formulário de Perfil ativa a proteção e o dono mantém acesso. | Estatísticas e gestão próprias continuam acessíveis. | As operações próprias continuam autorizadas com a proteção ativa. |
| Comum, perfil público de terceiro | Leitura 200. Gestão do terceiro e Servidor 403. | Leitura 200, gestão 403. | Gestão do perfil alheio recusada. |
| Comum, perfil protegido de terceiro | 404 com a mensagem de página ausente. | Derivados 404, gestão 403. | Exportação recusada com `ErrForbidden`, sem escrever bytes. |
| Root, qualquer perfil | Leitura, gestão e Servidor 200. | Leitura e operações administrativas autorizadas. | Gestão de outros perfis, exportação e mudança de papel autorizadas. |
| Root rebaixado, cookie anterior | Servidor 403 e perfil protegido de terceiro 404. | Leitura protegida 404 e gestão do terceiro 403. | Exportação alheia retorna `ErrForbidden` com a identidade antiga. A gestão própria permanece disponível. |
| Sessão expirada | Perfil protegido 404. Entrada de Conta serve a tela de login com 200 e aviso de nova sessão. | Leitura protegida 404 e gestão 401. | `ErrUnauthorized` antes de produzir exportação. |

O login com credencial inválida retorna 401 com mensagem uniforme. O console
HTML tem entradas GET que apresentam o login com 200. As operações que exigem
sessão mantêm o 401. A leitura protegida negada e a leitura ausente foram
comparadas no corpo e nos cabeçalhos, inclusive com prefetch e requisição
condicional. A comparação de respostas JSON inclui estatísticas, interesse,
aliases, álbuns mais ouvidos e primeira atividade. O Chrome também verificou
o GET de confirmação de saída e o POST que encerra a sessão.

A cobertura de imagens inclui avatar público, capa por URL de perfil e URL
legada, GET/HEAD, ETag público e troca de ETag ao proteger. Dono e root
recebem resposta privada sem armazenamento após o gate. Uma requisição
condicional sem autorização recebe 404 sem ETag em vez de 304. Cópias
públicas já servidas podem ficar frescas por até 30 dias, conforme a decisão
registrada no contrato.

As suítes de contas verificam estados ativo e pendente, ativação, recuperação,
expiração e consumo único de token, troca de senha com revogação de sessões
e proteção concorrente do último administrador. O teste instrumentado de
login confere a tentativa bcrypt nas falhas de credencial. Um teste adicional
confirma que HTML e JSON consomem o mesmo limite compartilhado de dez
tentativas e recebem 429 na décima primeira. A cobertura de perfil público
de origem vem das fixtures e da migração. A continuidade do relay protegido
consta no relatório de regressão.

## Execução

- `engine/visibility_issue161_test.go`, `engine/authorization_issue160_test.go`,
  `internal/admin/authorization_issue160_test.go` e
  `engine/authorization_issue164_test.go`.
- Jornadas `multiuser`, `permissions` e `isolation` do e2e.
- `python3 scripts/validate_multiuser_browser.py --binary ./tunetrail`, com
  Playwright e Chromium instalados. Use `PLAYWRIGHT_MODULE` ou `CHROME_PATH`
  para indicar instalações fora do caminho padrão. O script isola dados e
  sessões por modo, bloqueia requisições de página para origens externas e
  encerra o navegador e o servidor ao sair.

A inspeção de teclado e leitor de tela ficou fora desta execução. O teste
de navegador cobre o formulário de proteção, leitura por papel, login,
rebaixamento, expiração e logout. As demais ações da tabela usam os testes
HTTP e de serviço.

A configuração global `TUNETRAIL_LOGIN_GATE` permanece sem efeito, conforme
a ISSUE-197. O guia orienta a usar a proteção de cada perfil.
