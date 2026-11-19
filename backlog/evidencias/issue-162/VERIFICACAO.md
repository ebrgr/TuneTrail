# ISSUE-162 — Verificação

A conferência começou sobre a main `35d9fa1` (v0.40). A última migração em `db/migrations_sqlite` é a 006. A 007 está livre.

## Decisões de ebrgr em 19/11/2026

- Ativação expira em 24 horas e recuperação em 30 minutos. Os prazos são constantes no código.
- Um root pode emitir token para qualquer outra conta, inclusive para outro admin. A política central veda emitir para si mesmo. O único administrador recupera acesso localmente, sem token.
- O root pode excluir sua própria conta e perfil mediante confirmação quando houver outro administrador ativo. Continuam valendo `ErrLastAdmin` e os triggers existentes. A exclusão encerra as sessões da conta excluída.

## Recuperação local — decisão de ebrgr em 19/11/2026

O comando local `reset-password` gera a senha, revoga as sessões e publica a nova senha no log do processo. O operador aceitou o risco e deve proteger o acesso aos logs. O AC #3 abrange a gestão web e API da ISSUE-162.

## Rotas e permissões

| Rota | Visitante | Usuário comum | Root |
| --- | --- | --- | --- |
| `GET /admin?profile={id}` | Formulário de login | Gestão existente do próprio perfil | Gestão de qualquer perfil e dados operacionais da conta |
| `POST /admin/profile/create` | 401 | 403 | Cria conta pendente, perfil público e chave inicial na mesma transação |
| `GET /admin/profile/{id}/delete-review` | Login exigido | 403 | Exibe o impacto da exclusão conjunta |
| `POST /admin/profile/{id}/delete` | 401 | 403 | Confirma nome, slug e contagem e exclui a dupla. A autoexclusão encerra a sessão e exige outro admin ativo |
| `POST /admin/profile/{id}/account-role` | 401 | 403 | Promove ou rebaixa. Preserva o estado da conta e bloqueia o rebaixamento do último admin ativo |
| `POST /admin/profile/{id}/account-token` | 401 | 403 | Emite para outra conta, inclusive admin. Emissão para si recebe 403 e erro tipado |
| `GET /admin/redeem` | Formulário público | Formulário público | Formulário público |
| `POST /admin/redeem` | Token, CSRF e senha nova | Mesma regra | Mesma regra |
| `POST /apis/web/v1/profiles` e `DELETE /apis/web/v1/profiles/{slug}/` | Sessão exigida | Negados | Cadastro e exclusão pelos mesmos serviços da gestão HTML |
| `GET /apis/web/v1/profiles` e `GET /apis/web/v1/profiles/{slug}/` | Diretório público | Diretório público | Diretório público |

Os formulários autenticados reutilizam CSRF e tickets de envio. Cadastro, exclusão, emissão e alteração de papel revalidam a sessão root dentro da transação de escrita. A alteração de papel identifica o alvo pelo perfil persistido. A exclusão exige confirmação e remove histórico, sessões, tokens, chaves e relay vinculados. O catálogo compartilhado e os demais perfis são preservados.

## Ciclo de vida do token

1. A emissão gera 32 bytes aleatórios, codificados em base64url. A migração 007 guarda somente o digest SHA-256, associado à conta, à finalidade e à expiração. Emitir outro token invalida o anterior. O segredo aparece exclusivamente na resposta HTML da emissão, com `no-store` e `no-referrer`.
2. O titular informa usuário, token e senha nova no formulário público. O resgate aceita os dados no corpo do POST e reutiliza a proteção CSRF de pré-autenticação. A rota usa `Cache-Control: no-store` e `Referrer-Policy: no-referrer`. O redirecionamento de sucesso contém apenas `/admin`.
3. A verificação compara digests de tamanho fixo em tempo constante e usa um digest de tamanho idêntico quando não há candidato. O bcrypt da nova senha roda antes do lock de escrita, inclusive para tentativas sintaticamente válidas com conta ausente ou token rejeitado. Token inválido, expirado, consumido ou pertencente a outra conta recebe a mesma resposta.
4. Consumo, alteração da senha, ativação de conta pendente e revogação de todas as sessões dessa conta ocorrem na mesma transação. O resgate não cria sessão. A exclusão da conta remove o eventual token pela FK.
5. A migração é aditiva, preserva 001–006, gera backup da base v6 antes da atualização e valida o layout v7 no bootstrap.

## Cobertura executada

- `TestAccountTokenWeb162` cobre formulários com CSRF e tickets, matriz de autorização, emissão única, substituição, conta alheia, login após ativação, reset de outro admin, revogação das sessões, promoção, confirmação, autoexclusão, proteção do último admin e ausência de credenciais no HTML de listagem e nos logs.
- `TestAccountTokenRoutes162` exercita o roteador de produção e verifica as respostas JSON e os cabeçalhos do resgate.
- `TestAccountTokenTransaction162` cobre TTLs, expiração, reuso, falhas de gravação e rollback, exclusão conjunta e isolamento de histórico, catálogo, sessões e relay.
- `TestAccountTokenWAL162` disputa consumo e autoexclusão em conexões WAL independentes. O teste `TestLastActiveAdministrator160` também disputa exclusão, rebaixamento e desativação.
- `TestAccountTokenUpgrade162` verifica backup v6, migração, reinício e rejeição de v7 sem a tabela de tokens.

## Resultado

- `make check` passou. Inclui Go vet, testes Go, comparação de 169 referências HTML e 13 jornadas E2E.
- `go test -race` passou em `internal/admin`, `internal/adminui`, `internal/adminweb`, `internal/db`, `internal/db/sqlite` e `engine`. A rodada focada com a versão final dos testes também passou para a disputa em WAL.
- A revisão local do diff, da política de autorização e do grafo verificou o escopo dos alvos e as transações. As referências públicas mudaram somente no identificador de geração dos assets. `git diff --check` passou.
- AC #1 comprovado pela matriz de rotas, pelos testes de rollback e isolamento e pela proteção concorrente do último admin. AC #2 comprovado pelos testes de validade, consumo único e revogação. AC #3 comprovado pela inspeção das respostas HTML/JSON e do log HTTP capturado durante o fluxo.
