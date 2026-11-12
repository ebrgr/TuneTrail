# Issue 010 — contratos administrativos e bloqueadores de escrita

**Atualização na issue 015:** integração Go, jornadas reais e comparação visual
registradas em [ISSUE_015.md](ISSUE_015.md). As notas abaixo descrevem a entrega
anterior à integração; o aceite final do usuário permanece pendente.

Núcleo implementado e testado em 05/11/2026. [Issue](<../../tasks/iss-010 - 010-—-Estabelecer-contratos-administrativos-e-corrigir-bloqueadores-de-escrita.md>)
· [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

## Entrega

`internal/admin` reúne autenticação, autorização, alteração de conta, logout e
contratos de mesclagem, sem depender de handlers HTTP. Os adaptadores JSON de
conta/merge e o middleware usam essa base. As próximas páginas Go podem usá-la
diretamente; não há loopback HTTP ou invocação de handlers como serviços.

As sessões têm prazo fixo de 24 horas ou 30 dias conforme remember-me. Leituras
não estendem o prazo no banco; o cookie usa o vencimento efetivamente salvo.
Trocar apenas username preserva sessões. Trocar senha revoga todas na mesma
transação e limpa o cookie após sucesso; o cliente React atualiza o contexto e
volta ao login. Um login que validou a senha antiga não pode criar sessão depois
da troca: a inserção condicional verifica o hash e o papel ainda atuais no banco.
Falha de revogação faz rollback também da senha.

Merges de artista, álbum e faixa validam IDs positivos/distintos e a existência
de ambas as entidades dentro do tx. A leitura de artistas do álbum passou a usar
o mesmo tx, funcionando com a única conexão do SQLite em memória. Erros de Scan/
Rows são propagados. A deduplicação da faixa continua pessoal (perfil+instante).
JSON conserva métodos e payloads, mas passa a responder 400 para identidades
inválidas e 404 para origem/destino inexistentes, sem incluir SQL no erro público.

`internal/adminweb` prepara os contratos HTTP da administração:

- Autenticação por cookie único, no-store e cookies com os atributos existentes.
- Token HMAC-SHA256, chave aleatória em memória, versão/propósito/expiração e
  vínculo à sessão autenticada; comparação constante com hmac.Equal.
- Login com cookie de nonce separado, assinado, HttpOnly/SameSite Strict, até
  30 minutos e sem renovar o prazo em GET. Sucesso limpa o nonce.
- Parser POST URL encoded/multipart, campos repetidos rejeitados e uma única
  fonte de token (body); query e X-CSRF-Token são rejeitados.
- Multipart exige csrf_token textual primeiro, até 512 bytes, antes de criar
  staging. Somente após EOF, limites e duplicatas validados retorna PreparedForm.
  Falhas tardias removem os temporários; Close também limpa após erro de operação.
- Valores de credenciais preservam espaços. Erro de disco permanece erro interno;
  corpo excedido, campo inválido e CSRF têm respostas públicas distintas.
- Retorno interno restrito a páginas GET e query conhecida; URLs externas,
  escapes, duplicatas, segredos na query e caminhos de action não são aceitos.
- Mapeamento comum 400/422/401/403/404/405/409/413/500 sem detalhes de infraestrutura.

O [contrato para os próximos adaptadores](../../../internal/adminweb/README.md)
explica montagem com autorização/SameOrigin, validação antes da mutação,
rate limit de login, limpeza e redirect 303. Nenhuma rota/tela/action nova foi
ativada nesta issue. CSRF novo não é exigido nas APIs JSON existentes; será
aplicado às forms HTML das próximas entregas.

A chave CSRF é local ao processo: reiniciar invalida forms abertas. O contrato
atual é instância única, sem compartilhamento entre réplicas. Tokens podem ser
reutilizados até expirar e não dão idempotência; as operações precisam preservar
suas próprias invariantes ao receber uma submissão repetida.

## Evidências

A reprodução original foi repetida sem mudar sua fonte, somente sobre SQLite v3
em memória. [Antes](analysis-checks.json) · [Depois](issue010-regressions.json).

| Caso | Antes | Depois |
| --- | --- | --- |
| MergeArtists(1,1,false) | Sucesso; 4 → 2 escutas | Rejeitado; 4 escutas e dois artistas preservados |
| MergeAlbums(1,2,false), uma conexão | Deadline de 250 ms | Sucesso antes do deadline; quatro escutas preservadas |

O tempo observado no helper não é benchmark. As regressões de dados estão nos
testes SQLite, incluindo snapshot de tabelas antes/depois e triggers de falha.

- `go test ./...`: passou, incluindo testes HTTP em portas locais temporárias.
- `go test -race ./internal/admin ./internal/adminweb ./internal/db/sqlite`: passou.
- `go vet ./internal/admin ./internal/adminweb ./engine/handlers ./engine/middleware`:
  passou.
- Cliente: typecheck, nove testes e build de produção passaram.
- `scripts/validate_multiuser.py --binary /private/tmp/koito-issue010 --work-dir
  .local/issue010-multiuser`: passou nas bases nova e migrada v2, com reinício e
  verificação de integridade/chaves estrangeiras/esquema v3.

Cobertura nova: auto-merge de todos os tipos, IDs inválidos/ausentes, SQL failure
sem vazamento, origem inválida, chave sem admin, repetição de merge e rollback
após escrita parcial; duas pessoas com músicas no mesmo segundo; prazo de cookie
versus banco e usuário não-admin/expirado; revogação e login antigo concorrente;
falha de revogação com rollback; propósito/vínculo/expiração/reinício CSRF; login
pré-sessão; arquivo antes do token, token longo/inválido, duplicata tardia,
envelope truncado, tamanho excedido, staging limpo e retorno externo rejeitado.

## Revisão e limites

O Advisor em Sol, esforço alto, revisou a proposta antes dos patches e o código
posteriormente. As correções pedidas — login condicional ao hash atual, parser
completo antes de entregar arquivos, erro de disco 500 e preservação de espaços —
foram incorporadas e testadas. Parecer final: **aprovado**. As execuções de testes
registradas acima foram feitas pelo agente principal com cache Go temporário.

Por orientação do usuário, não houve abertura de navegador, inspeção de layout
ou screenshots. A comparação visual da etapa 4 ocorrerá somente no final da 015.
A integração dos helpers de formulário nas rotas reais é trabalho das próximas
issues, que deverão testar novamente os handlers efetivamente montados. A 010
não comprova jornadas visuais/admin Go ainda inexistentes.

Nenhuma migration nova, alteração em banco operacional, reinício da instância
do usuário, publicação, commit ou push. SQLite continua no esquema consolidado v3.
