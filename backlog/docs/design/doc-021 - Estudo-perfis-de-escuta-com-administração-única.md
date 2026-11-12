---
id: doc-021
title: 'Estudo: perfis de escuta com administração única'
type: specification
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - design
---
> Importado de [docs-dev/MULTIUSER_DESIGN.md](../../../docs-dev/MULTIUSER_DESIGN.md).

# Estudo: perfis de escuta com administração única

Versão histórica deste documento: **Koito 0.3.2-SUM-RC3**. A versão atual do TuneTrail é controlada por `internal/version/VERSION`.

Status: passos 1 a 5 implementados e validados para o MVP local. O fluxo completo
foi exercitado em instalação nova, migração consolidada v2 → v3 e cópia do banco
de exemplo. O esquema vigente é a versão 3; registros históricos que mencionam
a versão 4 descrevem a organização anterior à consolidação autorizada abaixo.
A navegação usa perfis explícitos; a gestão é uma aba dentro de
**Settings → Administration**, com `/admin` disponível para acesso direto,
e exige sessão administrativa independente.
Limites da validação e compatibilidade legada estão registrados no passo 5.

## Objetivo e experiência

Separar os registros de reprodução (scrobbles), sem armazenar áudio e sem
exigir cadastro/login de cada ouvinte. Preservar as telas, gráficos e temas do
Koito. Cada pessoa tem um perfil público com nome de apresentação, identificador
de URL e foto ou ícone. Somente a administração exige senha.

- `/`: seleção de perfis, com avatar e nome. Continua mostrando a seleção a
  cada acesso explícito à raiz, mesmo que exista um último perfil usado.
- `/default`: perfil que recebe o histórico existente em uma instalação com
  um único usuário.
- `/<user>`: página inicial desse perfil.
- `/<user>/listens`, `/<user>/rewind` e
  `/<user>/chart/top/artists`: telas atuais com dados desse perfil.
- `/<user>/artist/123`, `/<user>/album/456` e
  `/<user>/track/789`: catálogo compartilhado com métricas pessoais.
- `/admin`: login da administração e gestão dos perfis/chaves. O acesso pelo menu
  fica na aba **Settings → Administration**, com o conteúdo no próprio painel,
  sem ícone administrativo separado na barra principal.

Adicionar um item de perfil à barra lateral estreita existente. O avatar abre
um seletor com foto/ícone e nome, sem alargar permanentemente a barra. No celular,
usar o mesmo seletor na barra de navegação atual. Exibir o nome do perfil no
cabeçalho para tornar inequívoco de quem são os dados.

Trocar o perfil atualiza a URL e todas as consultas. Preservar tela e período
quando possível; fechar modais de edição ao trocar. Usar a URL como fonte de
verdade, permitindo duas abas com perfis diferentes. Não usar um cookie global
de "perfil atual" para determinar os resultados do backend.

## Identidade: administrador, perfil e API key

Separar os conceitos:

1. Conta administrativa: autentica ações de gestão com senha e sessão.
2. Perfil de escuta: identidade pública, sem senha ou sessão própria.
3. API key: credencial de ingestão vinculada a exatamente um perfil.

Um perfil pode ter várias chaves: por exemplo, "Navidrome" e "Celular". Todas
contribuem para o mesmo histórico. Criar uma chave inicial junto com o perfil.
Gerar, consultar e revogar credenciais somente na administração; nenhuma resposta
pública de perfil deve incluir tokens. Selecionar um perfil não concede permissão
de escrita.

O envio continua usando `/apis/listenbrainz/1/submit-listens` e
`Authorization: Token ...`, preservando compatibilidade com os players.
O servidor resolve `API key → perfil → registro`; não aceita um perfil escolhido
pelo payload para substituir o vínculo da chave.

Revogar uma chave bloqueia novos envios e preserva o histórico. Substituir uma
chave também preserva as reproduções. No MVP, uma chave não pode ser transferida
entre perfis: gerar uma nova evita ambiguidade sobre o histórico anterior.

## Modelo de dados proposto

Criar `listening_profiles`, separado da tabela `users`, para não criar senhas
fictícias de ouvintes ou transformar perfis públicos em contas administrativas.

| Campo | Tipo/regra | Finalidade |
| --- | --- | --- |
| `id` | INTEGER PK | Identidade interna estável |
| `slug` | TEXT NOT NULL UNIQUE | Segmento de URL: `<user>` |
| `display_name` | TEXT NOT NULL | Nome visível, com espaços e acentos |
| `avatar_image_id` | TEXT opcional | Referência a foto gerenciada localmente |
| `avatar_icon` | TEXT opcional | Ícone de uma lista permitida |
| `created_at` | INTEGER NOT NULL | Criação em Unix segundos |

Usar foto quando existir, senão ícone, senão iniciais do nome. O slug usa letras
minúsculas ASCII, números e hífen, sem espaços; reservar nomes de rotas técnicas
como `admin`, `apis`, `image`, `public`, `assets` e caminhos legados. O nome de
apresentação pode mudar sem alterar a URL. Conforme a regra adicionada ao passo 2,
a administração também pode editar o slug explicitamente, junto com o nome ou
separadamente. O ID do perfil e os vínculos com histórico/chaves permanecem estáveis.
A URL antiga passa a retornar 404; não há redirecionamento para outro perfil.

Alterações no restante do banco:

- `api_keys.profile_id`: FK obrigatória para `listening_profiles.id` após o
  preenchimento dos registros existentes.
- `listens.profile_id`: FK obrigatória para `listening_profiles.id`.
- PK de `listens`: `(profile_id, track_id, listened_at)`.
- Índice de histórico: `(profile_id, listened_at)`; avaliar também
  `(profile_id, track_id, listened_at)` nas consultas por faixa.
- `sessions` continuam ligadas à conta administrativa, nunca ao perfil visitado.
- Artistas, álbuns, faixas, aliases e capas continuam compartilhados.

Os campos legados `user_id` exigem uma migração explícita. Durante a transição,
podem permanecer como vínculos legados, mas todas as novas consultas de escuta
devem usar `profile_id`. Remover esses vínculos de `listens` e `api_keys` na
conclusão da migração; não usar a exclusão de uma conta administrativa para
excluir o histórico de um perfil. Não modificar migrações já aplicadas.

Não é necessário guardar a chave em cada reprodução para separar os perfis:
`profile_id` é suficiente. Como evolução, `listens.api_key_id` opcional permitiria
auditoria por origem. Nesse caso, a exclusão da chave deve usar `ON DELETE SET NULL`
e o `profile_id` da reprodução permanece preservado. Nunca armazenar o token
secreto dentro de cada reprodução.

## Leitura e permissões

API pública proposta:

```text
GET /apis/web/v1/profiles
GET /apis/web/v1/profiles/{slug}
GET /apis/web/v1/profiles/{slug}/listens
GET /apis/web/v1/profiles/{slug}/stats
GET /apis/web/v1/profiles/{slug}/top/artists
GET /apis/web/v1/profiles/{slug}/now-playing
GET /apis/web/v1/profiles/{slug}/summary
```

Estender esse padrão aos demais endpoints de leitura. Resolver slug para ID
interno no backend. Perfil ausente deve retornar 404, sem cair em dados globais
ou no perfil default. Busca pública mostra o catálogo relacionado às escutas do
perfil; os gráficos e métricas usam exclusivamente suas reproduções.

| Ação | Visitante | Chave de ingestão | Sessão administrativa |
| --- | --- | --- | --- |
| Listar perfis e ler estatísticas | Sim | Não necessária | Sim |
| Enviar scrobble / playing now | Não | Somente ao perfil vinculado | Não substitui a chave |
| Criar/editar/excluir perfil e gerenciar chaves | Não | Não | Sim |
| Editar catálogo, aliases, imagens e merges | Não | Não | Sim |
| Excluir reproduções/importar/exportar arquivos de histórico | Não | Não | Sim, perfil explícito |
| Limpeza global | Não | Não | Ação administrativa separada |

Na proposta, os dados de escuta são públicos para quem tem acesso à instância,
inclusive a lista de perfis e o estado "tocando agora". Isso atende à navegação
sem login; perfis privados ficam fora do MVP. Downloads de histórico ficam na
administração na primeira versão, embora as consultas públicas exibam escutas.

Autenticação não deve mais significar que uma API key de player autoriza edição
global. Criar middleware administrativo que exija sessão e papel admin. Rever
as rotas atuais protegidas por `AuthModeSessionOrAPIKey`. Proteger também as
mutações administrativas por cookie contra requisições de outras origens.

As credenciais Subsonic para imagens podem continuar globais. O relay ListenBrainz
deve ter configuração por perfil se utilizado; não encaminhar reproduções de todos
os perfis para uma conta externa única de forma implícita. Na migração, associar
o relay legado explicitamente ao perfil original.

## Cadastro, exclusão e dados existentes

Cadastro administrativo: nome → slug sugerido/editável → foto/ícone → criar.
Apresentar a chave inicial e o endereço de scrobbling para copiar. Oferecer novas
chaves e revogação na edição do perfil. Não pedir senha ao ouvinte.

Exclusão: confirmar nome e quantidade de reproduções afetadas; excluir perfil,
chaves, reproduções, avatar próprio e estado temporário de playing now. Não
excluir entidades do catálogo que outras pessoas usam. Limpeza de órfãos deve
considerar referências de todos os perfis. Permitir excluir o último perfil e
mostrar um estado vazio; a administração continua acessível. `default` é o perfil
da migração, não um destinatário automático de chaves inválidas ou dados sem dono.

Migração de instalação atual:

1. Fazer cópia do banco antes da migração e aplicar o preenchimento em transação.
2. Se existir um único usuário com dados/chaves, criar o perfil `default` e
   associar a ele o histórico e as chaves existentes, sem trocar seus tokens.
3. Se houver vários usuários existentes, criar um perfil correspondente para
   cada um; resolver slugs conflitantes e preservar todas as associações.
4. Validar totais, FKs e ausência de reproduções/chaves sem perfil.
5. Recriar a chave primária de `listens` e adaptar os índices.
6. Preservar a conta administrativa e sua senha. Contas legadas não administrativas
   não devem ganhar privilégios; não criar login de ouvinte na nova experiência.
7. Redirecionar links antigos de telas para o perfil legado definido. Em bases
   com vários perfis, exigir uma escolha de destino em vez de combinar históricos.

Se dados de várias pessoas já tiverem sido gravados sob um mesmo `user_id`, o
banco atual não registra a chave de origem de cada reprodução. Essa separação
retroativa não pode ser reconstruída com certeza a partir das API keys; requer
fontes externas ou uma regra de atribuição escolhida pelo administrador.

## Impactos identificados no código

- `client/app/routes.ts`: layout de perfil com `:slug`, seleção na raiz e `/admin`.
- `client/app/components/sidebar/Sidebar.tsx`: seletor de perfil no desenho atual.
- `client/app/providers/AppProvider.tsx`: separar `adminSession` de `activeProfile`.
- `client/api/api.ts` e loaders: endpoints com perfil explícito.
- Links absolutos atuais (`/track/...`, `/rewind` etc.): preservar o prefixo do perfil.
- React Query: incluir ID do perfil em todas as chaves de consultas pessoais;
  cancelar requisições antigas e impedir exibição temporária de dados do outro perfil.
- `engine/routes.go` e `engine/middleware`: separar leitura pública, ingestão e admin.
- `internal/db/opts.go`, `db.go` e `sqlite`: escopo obrigatório de perfil nas
  consultas pessoais, inclusive nas métricas de páginas de mídia e buscas.
- `internal/summary`: usar efetivamente o escopo nas consultas; hoje recebe um
  `userId` sem aplicá-lo e o handler fornece `1`.
- `engine/handlers/now_playing.go`: remover leitura fixa da chave `"1"`.
- `internal/catalog`: chave do cache de playing now passa a representar o perfil.
- `internal/importer` e `RunImporter`: destino de perfil explícito; remover IDs
  fixos. Preferir importação pela administração; diretórios automáticos, se
  mantidos, devem ter associação explícita ao perfil.
- Exportação: selecionar o perfil administrado, não o usuário da sessão admin.
- Exclusão/merge/correções do catálogo: preservar scrobbles de perfis diferentes
  na mesma faixa/horário e invalidar caches afetados.

## Plano de implementação e validação

1. Banco e migração: perfis, associações e unicidade das reproduções.
2. Backend: identificação pela chave, escopo das consultas, permissões e remoção
   dos IDs fixos, incluindo importação e exportação. Permitir editar o nome do
   perfil e seu slug sem alterar a identidade interna nem as chaves associadas.
3. Interface: layout por slug, seletor lateral, raiz com escolha e cache por perfil.
4. Administração: cadastro, avatar/ícone, chaves, exclusão e destino de importação.
   - Ponto de atenção resolvido e validado no passo 5: alteração de nome e senha
     administrativos com novo login; nome/slug do perfil são editados separadamente.
5. Validação do fluxo inteiro com banco migrado e instalação nova.

Critérios de aceite:

- Dois perfis registram a mesma faixa no mesmo segundo sem perda de eventos.
- Duas chaves do mesmo perfil alimentam o mesmo histórico.
- Histórico, rankings, atividade, retrospectiva e playing now respeitam o perfil.
- Acesso direto, atualização da página e duas abas mantêm o perfil correto.
- Trocar de perfil preserva o visual e não mostra dados antigos do cache.
- Visitantes e chaves de ingestão não executam administração.
- Revogar chave preserva o histórico; excluir perfil não apaga dados dos demais.
- Migração preserva senha administrativa, tokens e reproduções existentes.
- Seleção pública nunca revela tokens ou dados de sessão.

Fora do MVP: login por ouvinte, cadastro público, perfis privados, permissões
individuais de edição, personalização do catálogo por pessoa e auditoria de
reproduções por chave específica.

## Registro da implementação — passo 1

- Migração `003_listening_profiles.sql`, transacional; migrações 001/002 preservadas.
- `listening_profiles` com metadados públicos, slugs únicos/validados e IDs sem
  reutilização automática após exclusão.
- Perfil `default` para uma conta legada; `user-<id>` para múltiplas contas,
  preservando nomes visíveis e evitando conflitos entre nomes reservados ou similares.
- `profile_id` obrigatório em `api_keys` e `listens`, com exclusão em cascata
  somente a partir do perfil e unicidade `(profile_id, track_id, listened_at)`.
- `legacy_user_profiles` e trigger de criação de perfil para contas legadas:
  compatibilidade temporária com o bootstrap do administrador e chamadas atuais.
  A tabela de ponte não confere permissões administrativas ao perfil.
- `user_id` permanece como atribuição legada opcional nas duas tabelas, sem FK
  que apague dados ao excluir a conta. Métodos de escrita aceitam `ProfileID`;
  chamadas atuais com `UserID` resolvem a ponte, sem fallback para `default`.
- Antes de migrar um banco existente, `sqlite.New` gera um snapshot consistente
  via `VACUUM INTO`, incluindo WAL, ao lado do banco:
  `koito.db.pre-profiles-*.bak`, com permissão 0600. Falha no backup impede a migração.
- Não há downgrade SQL destrutivo: para voltar, pare todas as conexões e restaure
  um backup completo com um procedimento de restauração SQLite. A restauração
  representa o estado anterior à migração, não inclui gravações posteriores.
- Testes cobrem instalação nova, migração simples/múltipla, integridade, duplicatas,
  preservação de credenciais, backup, rollback e ciclo de exclusão.
- Teste opcional em cópia do banco de exemplo (original aberto somente para leitura):

  ```sh
  KOITO_MIGRATION_TEST_DB="$PWD/.local/koito/koito.db" go test ./internal/db/sqlite -run TestProfileMigrationExampleCopy -v
  ```

Registro histórico do passo 1: sua migração é aplicada automaticamente ao reiniciar
a API. Na conclusão do passo 1, as rotas de página `/default` e o seletor lateral
ainda estavam pendentes; foram implementados no passo 3 abaixo; a identificação de chaves sem conta legada e o isolamento de consultas
foram implementados no passo 2, abaixo.


## Registro da implementação — passo 2

### Identificação e autorização

- A API ListenBrainz usa `IngestionKey`: valida o token diretamente em `api_keys`
  e resolve `profile_id`, sem depender de `users` ou criar senha para o ouvinte.
  O payload não escolhe o destino. `/validate-token` retorna o slug atual do perfil.
- A administração exige cookie de sessão válido e `role=admin`. Uma chave de
  ingestão, mesmo legada e associada ao administrador, não autoriza gestão de
  chaves, exportação, importação, exclusões ou alterações do catálogo.
- O login aceita somente contas administrativas. Removido o middleware antigo
  que combinava autorização por sessão e chave de API. `KOITO_LOGIN_GATE` não
  transforma as leituras por perfil em login: os perfis são públicos neste modelo.
- Mutações web validam `Origin` e `Sec-Fetch-Site`; cookies administrativos usam
  `SameSite=Strict` e `Secure` quando servidos por HTTPS. O proxy Vite preserva
  `Host` (`changeOrigin=false`) para manter essa validação no desenvolvimento.
- Revogação e alteração do rótulo de chaves verificam o perfil selecionado e
  retornam 404 quando a chave pertence a outro perfil.

### Contrato HTTP disponível

Todos os caminhos abaixo têm o prefixo `/apis/web/v1`:

| Método e rota | Acesso e efeito |
| --- | --- |
| `GET /profiles` | Público: lista metadados públicos, sem tokens |
| `GET /profiles/{slug}` | Público: resolve o perfil ou retorna 404 |
| `GET /profiles/{slug}/listens`, `/stats`, `/listen-activity`, `/first-activity` | Histórico e métricas do perfil |
| `GET /profiles/{slug}/top/tracks`, `/top/albums`, `/top/artists` | Rankings pessoais |
| `GET /profiles/{slug}/artist/{id}`, `/album/{id}`, `/track/{id}` | Metadados compartilhados com métricas pessoais; mídia sem escutas no perfil retorna 404 |
| Subrotas de aliases, artistas e interesse das mídias | Mesmo contexto de perfil; aliases/associações verificam relação com o perfil |
| `GET /profiles/{slug}/search` | Busca restrita ao catálogo ouvido pelo perfil |
| `GET /profiles/{slug}/now-playing`, `/summary` | Estado atual e retrospectiva do perfil |
| `PATCH /profiles/{slug}` | Admin: altera `display_name` e/ou `slug` |
| `GET /profiles/{slug}/export` | Admin: exporta somente esse perfil |
| `POST /profiles/{slug}/listens`, `DELETE /profiles/{slug}/listens` | Admin: grava/exclui escutas somente nesse perfil |
| `GET/POST /profiles/{slug}/user/apikeys` | Admin: lista/gera chaves do perfil |
| `PATCH/DELETE /profiles/{slug}/user/apikeys/{id}` | Admin: renomeia/revoga a chave vinculada |
| `POST /profiles/{slug}/import` | Admin: importa um arquivo previamente colocado no diretório do perfil |
| `PUT/DELETE /profiles/{slug}/relay` | Admin: configura/remove relay próprio; não retorna o token |
| `GET /profiles/{slug}/relay` | Admin: lê configuração salva, incluindo token visível; `Cache-Control: no-store` |
| `POST /profiles/{slug}/relay/test` | Admin: valida credenciais do formulário sem salvar ou enviar escutas |

As rotas de leitura antigas sem perfil continuam disponíveis somente quando há
exatamente um perfil. Com nenhum ou vários perfis, retornam 409 solicitando uma
escolha explícita, nunca dados globais. As operações globais de catálogo e a limpeza
completa continuam nas rotas antigas, protegidas por sessão administrativa.
As rotas de páginas no navegador foram adaptadas no passo 3, registrado abaixo.

Exemplo de alteração administrativa:

```http
PATCH /apis/web/v1/profiles/default
Content-Type: application/json
Cookie: koito_session=...

{"display_name":"Everton Berger","slug":"evertonberger"}
```

O frontend deve passar a usar o slug retornado. A URL anterior retorna 404; tokens,
reproduções, diretório de importação e relay continuam vinculados ao mesmo ID.
Slugs inválidos, reservados ou já usados são rejeitados sem alteração parcial.

### Isolamento das consultas e estado temporário

- `db.WithProfile` transporta o ID interno no contexto da requisição. O middleware
  resolve a URL antes de executar handlers e consultas; não há cookie global de
  seleção e parâmetros de payload não substituem esse ID.
- As consultas SQLite de leitura usam uma CTE local à instrução que filtra
  `main.listens` por `profile_id`. Isso aplica o mesmo escopo às subconsultas de
  contagem, rankings, primeira escuta, interesse, sequências e retrospectivas.
  Trabalhos internos do catálogo podem ler globalmente com contexto sem perfil;
  essa opção não é exposta nas rotas públicas de estatísticas.
- Buscas acrescentam verificação de participação no histórico. Leituras diretas
  de mídia e de seus aliases/associações verificam a relação com o perfil.
- `summary.GenerateSummary` aplica efetivamente o ID recebido. Importadores,
  exportação e leitura de playing now não usam mais o ID fixo `1`.
- Playing now usa `now-playing:<profileID>`. Uma faixa ainda sem scrobble pode
  aparecer como tocando nesse perfil, com métricas pessoais zeradas. Se a mídia
  foi excluída, o estado obsoleto é removido em vez de retornar erro 500.
- O cache global de estatísticas de 30 minutos foi removido: consultas retornam
  dados atuais, sem misturar perfis nem manter contagens antigas após exclusão.
  Perfis vazios retornam valores numéricos finitos, evitando divisão por zero.
- Sem período explícito, a API usa `all_time`. Resultados vazios de busca são
  listas vazias, não `null`.
- O agrupamento de envios simultâneos inclui perfil, tipo e horário. A associação
  de scrobbles ao catálogo compartilhado é serializada no processo para evitar
  criação duplicada quando perfis enviam a mesma música ao mesmo tempo.
- Merges de faixas eliminam conflitos somente dentro do mesmo perfil, preservando
  escutas de pessoas diferentes no mesmo instante. Mesclar uma faixa com ela
  própria é rejeitado.

### Importação e exportação

O layout automático de entrada agora exige destino pelo ID estável:

```text
<KOITO_CONFIG_DIR>/import/<profileID>/<arquivo>
<KOITO_CONFIG_DIR>/import_complete/<profileID>/<arquivo>
```

Exemplo: `import/2/maloja_export.json`. O ID está disponível em `GET /profiles`.
Renomear o slug não muda esse diretório. Arquivos antigos diretamente em `import/`
não recebem destino implícito; devem ser movidos para o diretório apropriado.
`KOITO_SKIP_IMPORT` continua controlando a importação automática ao iniciar.

Para iniciar uma importação explicitamente pela API administrativa:

```http
POST /apis/web/v1/profiles/evertonberger/import
Content-Type: application/json
Cookie: koito_session=...

{"filename":"maloja_export.json"}
```

O arquivo deve existir em `import/<ID desse perfil>/`. O backend reconhece os
formatos pelos nomes já usados pelos importadores. No passo 2 esse endpoint aceitava somente arquivos locais; o passo 4 acrescentou
upload multipart na mesma rota, mantendo o contrato JSON de arquivos locais. Nomes com caminho e arquivos ou
diretórios de perfil que sejam symlinks são rejeitados. Os importadores também
exigem contexto de perfil quando chamados diretamente; o nome de usuário de um
arquivo importado não escolhe o destinatário.

A exportação usa o perfil escolhido, não a conta administrativa. Nomes são
escapados corretamente no JSON. Corrigido o cursor de paginação composto por
horário/faixa, evitando pular registros em exportações com mais de uma página.
Rankings e exportações drenam e fecham os resultados antes de consultar artistas
ou aliases, evitando bloqueio com SQLite de conexão única.

### Relay e migração consolidada

O relay foi inicialmente implementado em `004_profile_relays.sql`. Após a
consolidação, seu conteúdo faz parte de `003_listening_profiles.sql`, na mesma
transação dos perfis. A migração única acrescenta:

- `profile_relays`: URL e token de relay vinculados ao ID do perfil.
- `legacy_relay_binding`: registra uma única vez a decisão de associar a
  configuração legada, impedindo reassociação após exclusão ou revogação.
- Trigger que impede transferir uma API key existente para outro perfil.

A configuração global legada de relay é importada somente quando há um único
perfil inequivocamente ligado à conta administrativa. Havendo ambiguidade, o
backend registra a decisão sem destinatário e exige configuração administrativa
explícita. O relay só encaminha eventos do perfil associado, após processamento
válido do envio. Renomear o slug não altera seu destino.

Configuração explícita, protegida por sessão administrativa:

```http
PUT /apis/web/v1/profiles/evertonberger/relay
Content-Type: application/json
Cookie: koito_session=...

{"url":"https://servidor.example/apis/listenbrainz/1","token":"TOKEN_EXTERNO"}
```

A URL é a base que receberá `/submit-listens`. `DELETE` na mesma rota remove a
configuração. Configurações por perfil persistidas são independentes do flag
legado de bootstrap; remover o relay não o recria no próximo reinício.

### Validação e próximos passos

Testes implementados para:

- Dois perfis sem contas próprias enviando simultaneamente a mesma música/horário.
- Duas chaves alimentando o mesmo perfil e consultas sem vazamento entre perfis.
- Métricas pessoais de mídia, busca, resumo, perfis vazios e playing now de faixa nova.
- Rejeição de chaves em operações administrativas e de mutações com origem externa.
- Alteração de nome/slug preservando ID, chaves, histórico e exportação válida.
- Revogação de chave de outro perfil, revogação legítima e preservação do histórico.
- Importação com destino explícito, arquivamento por ID e relay sem mistura de perfis.
- Proibição de transferir chaves, relay sem reassociação e merges com escutas simultâneas.

Validações executadas: suíte Go completa, testes HTTP de perfis com detector de
corridas e checagem TypeScript. No passo 2, a interface ainda não tinha seletor nem navegação por
slug; esses itens foram implementados no passo 3. Cadastro/exclusão de perfis e upload de avatar continuam no
passo 4; as contas e colunas legadas permanecem apenas para compatibilidade de
migração/bootstrapping até a conclusão da transição.


Verificação adicional em cópia do banco de exemplo: migração da versão 3 para 4,
72 reproduções preservadas, estatísticas correspondentes a cada perfil,
validação da API key legada e `PRAGMA foreign_key_check` sem pendências.
O banco original permaneceu na versão 3 com as mesmas 72 reproduções.
O teste de paginação exportou os 1.001 registros esperados, incluindo dois no
mesmo segundo na transição entre páginas.


## Registro da implementação — passo 3

### Navegação e apresentação

- `/` sempre apresenta os perfis disponíveis, inclusive quando há somente um.
  Não há redirecionamento automático baseado em último perfil, cookie ou sessão.
  Sem perfis, exibe um estado vazio e mantém acesso à administração.
- `/:slug` usa um layout de perfil com nome visível, slug e avatar no cabeçalho.
  As telas existentes de início, histórico, gráficos, artistas, álbuns, faixas e
  retrospectiva ficam aninhadas nesse layout, preservando tema e componentes.
- A barra lateral ganhou um seletor compacto com avatar/ícone e lista de nomes.
  No celular, o mesmo controle aparece na barra horizontal. A lista marca o
  perfil atual e oferece um link para a seleção na raiz.
- O avatar usa imagem do cache quando informada; na ausência ou falha, usa um
  ícone permitido ou as iniciais do nome. Ícones suportados: `music`, `headphones`,
  `disc`, `radio` e `user`. O upload e a escolha administrativa ficam no passo 4.
- `/admin` reutiliza o formulário e os controles de conta administrativa existentes,
  sem exigir perfil selecionado. Cadastro/exclusão de perfis ainda não foi adicionado.
- Os controles laterais usam botões/links com nomes acessíveis. O seletor usa diálogo
  com identificação, navegação por teclado, fechamento por Escape e retorno do foco.

Rotas de página disponíveis:

```text
/
/admin
/:slug
/:slug/listens
/:slug/rewind
/:slug/chart/top/artists
/:slug/chart/top/albums
/:slug/chart/top/tracks
/:slug/artist/:id
/:slug/album/:id
/:slug/track/:id
```

### Seleção, links e compatibilidade

- A URL determina o perfil. Abrir diretamente uma página, atualizar o navegador
  ou usar duas abas com pessoas diferentes mantém cada destino independente.
- Links de cards, tabelas, artistas relacionados, resultados de busca e páginas de
  mídia preservam o prefixo do perfil. A navegação de períodos, paginação,
  retrospectiva e retorno após merge também preserva o slug.
- Trocar de perfil preserva histórico/gráfico/retrospectiva e seus períodos quando
  aplicável. Filtros por IDs de artista/álbum/faixa e a página da paginação são
  removidos, pois podem não ter sentido para a outra pessoa.
- Partindo de uma mídia específica, a troca leva ao início do novo perfil, em vez
  de abrir automaticamente uma faixa que ele pode nunca ter ouvido.
- O período da tela inicial agora também aparece na query string. Preferências
  locais de período e atividade usam o ID estável do perfil e a seção, mantendo
  suas escolhas mesmo se o slug for renomeado.
- Links antigos sem slug são resolvidos somente se houver um perfil. Com vários,
  mostram a escolha na raiz e preservam o destino em `next`; ao escolher a pessoa,
  abrem a página pretendida. O destino aceito é limitado a rotas internas conhecidas.
- Perfil inexistente, slug antigo após renomeação e mídia ausente no histórico
  apresentam erro com caminho de volta à seleção; não caem em dados globais.

### API, cache e ciclo de vida

- `AppProvider` mantém `adminSession`, separado de `ProfileProvider`. O nome `user`
  permanece como alias de compatibilidade para os controles administrativos antigos.
- `createProfileApi(slug)` cria funções de API vinculadas ao perfil. Os callbacks
  capturam o destino; não leem `window.location` para decidir onde gravar.
- Consultas e ações pessoais usam `/apis/web/v1/profiles/{slug}/...`. Login,
  configuração, conta administrativa e mutações do catálogo compartilhado mantêm
  os endpoints globais apropriados, protegidos pelo backend.
- Loaders usam `params.slug` e o `AbortSignal` da requisição do React Router.
  Assim, acesso direto e navegação rápida não dependem da ordem em que um contexto
  global foi atualizado.
- Cada perfil tem seu próprio `QueryClient`. Todas as chaves de consultas pessoais
  têm prefixo `["profile", profile.id, ...]`, inclusive as invalidações de cache.
- A troca desmonta o layout anterior, fecha seus modais, aborta requisições em
  andamento, cancela consultas e descarta seu cache. O ciclo também funciona no
  StrictMode de desenvolvimento. Uma nova seleção não reutiliza respostas da anterior.
- Busca e sugestões descartam respostas antigas. A busca de merge roda somente
  com seu modal aberto; cancelamentos de navegação não viram resultados atrasados.
- A primeira atividade é consultada por perfil. Perfis vazios não geram uma falsa
  data inicial em 1970 nem um intervalo inválido de anos nos gráficos.
- Corrigida a mensagem duplicada do histórico vazio e acrescentado um estado vazio
  nos gráficos. O histórico recente também permite mostrar playing now sem escutas
  concluídas. Versões locais `dev` não são comparadas como versões semânticas válidas.

### Verificações realizadas

- `make check`: suíte Go, geração de tipos/checagem TypeScript e testes de interface.
- `make client.build`: build de produção e assets gzip.
- `make client.test` / `cd client && corepack yarn test`: testes com o runner nativo
  do Node 24 para URLs, destinos obrigatórios, troca de tela/período e callbacks
  vinculados ao perfil, incluindo propagação de cancelamento.
- Validação manual no Chrome em uma cópia descartável do banco com as 72 escutas
  de exemplo e um segundo perfil vazio: seleção na raiz, métricas separadas, troca
  lateral, ranking com período, acesso direto e links de artista/faixa/álbum.
- Duas abas mantiveram perfis diferentes. O seletor e o cabeçalho foram inspecionados
  em viewport de 400 px; Escape fechou o diálogo e devolveu o foco ao seletor.
- Link antigo `/track/19` com vários perfis apresentou a escolha e abriu o destino
  com slug. Perfil inexistente apresentou erro sem fallback.
- Na cópia descartável, a exclusão de todos os perfis confirmou o estado vazio da
  raiz e a continuidade do acesso ao formulário em `/admin`.

O banco original foi usado somente para criar a cópia de validação. Nenhum perfil
ou registro de teste foi adicionado ao banco em uso. As portas de teste foram
separadas das portas normais do projeto; o padrão continua API 4110 e Vite 5173.
O Vite aceita `KOITO_DEV_API_TARGET` para apontar explicitamente para uma API de
validação, sem alterar o destino padrão de desenvolvimento.

Na conclusão do passo 3, cadastro/edição/exclusão administrativos, foto/ícone,
gestão de chaves e upload de importação ainda estavam pendentes. Esses itens
foram implementados no passo 4, registrado abaixo. A validação final está
registrada no passo 5.


## Registro da implementação — passo 4

### Administração e cadastro

- `/admin` agora apresenta a gestão de perfis após o login. Visitantes veem o
  formulário de autenticação; os formulários administrativos e tokens não são
  carregados sem sessão. O item “Administration”, inicialmente na barra principal,
  foi movido para dentro de Settings no passo 5.
- A conta administrativa fica em uma seção recolhível separada. O seletor
  “Manage profile” permite cadastrar ou administrar uma pessoa; `?profile=<ID>`
  mantém a seleção administrativa pelo ID estável, inclusive após mudar o slug.
- Cadastro com nome, slug sugerido a partir do nome e editável, ícone e foto
  opcional. A sugestão remove acentos e normaliza separadores; o servidor rejeita
  slugs inválidos, reservados e duplicados, além de nomes vazios ou excessivamente longos.
- Perfil e primeira chave são criados na mesma transação. Falhar a gravação da
  chave desfaz a criação do perfil. Nenhuma conta/senha de ouvinte é criada.
- Após cadastrar, a tela seleciona o perfil e oferece a chave inicial e o endpoint
  de scrobbling para copiar. A foto é enviada em uma operação posterior; se ela
  falhar, a interface informa que perfil/chave já existem e permite repetir só o upload.
- Edição de nome e slug preserva ID, histórico e tokens. A tela usa o slug retornado;
  a URL anterior retorna 404, como definido no desenho. Operações em andamento
  bloqueiam os formulários e o seletor de destino para evitar trocas acidentais.
- Corrigido o formulário existente da conta administrativa: mudanças de nome não
  enviam senha vazia, mudanças de senha não enviam nome vazio, a confirmação deve
  coincidir e a senha deve ter ao menos oito caracteres. O estado de carregamento
  agora termina ao concluir a requisição.

### Avatar e imagens

- Escolha entre iniciais e os ícones permitidos: `music`, `headphones`, `disc`,
  `radio` e `user`. Upload de foto multipart com limite de 5 MiB para a requisição.
- O servidor verifica o conteúdo e processa a imagem pelo cache WebP existente,
  atribuindo um UUID próprio. Não aceita um ID de imagem arbitrário pelo formulário.
- Foto tem prioridade sobre ícone. Salvar sem arquivo substitui a foto atual pelo
  ícone escolhido ou pelas iniciais; “Use initials” remove foto e ícone.
- Substituição/remoção e exclusão do perfil removem o arquivo antigo somente se
  nenhuma outra associação o usa. A limpeza global de imagens agora considera
  `listening_profiles.avatar_image_id`, evitando apagar avatares ao reiniciar.
- As imagens do catálogo continuam compartilhadas e não são alteradas ao editar
  o avatar de uma pessoa.

### Gestão de chaves

- Cada perfil tem sua lista administrativa de chaves: consultar, revelar/ocultar,
  copiar, criar com rótulo, editar rótulo e revogar com confirmação.
- A tela mostra tanto `/apis/listenbrainz/1/submit-listens` quanto a base
  `/apis/listenbrainz/1` para players que pedem a URL do servidor, e explica o
  cabeçalho `Authorization: Token ...`. A origem acompanha o endereço da instância.
- Chaves ficam ocultas visualmente até revelar, e continuam disponíveis somente
  em respostas administrativas. Não entram em metadados públicos nem em URLs.
- Não existe transferência de chave. Revogar interrompe novos envios sem excluir
  o histórico; outra chave pode ser criada para o mesmo perfil.

### Exclusão e preservação dos dados

- “Review deletion” consulta a contagem atual e mostra nome e quantidade de
  reproduções afetadas. É necessário digitar exatamente o nome de apresentação.
- O servidor exige nome e contagem e verifica ambos na própria instrução de
  exclusão. Mudanças desde a revisão retornam 409 e exigem revisar novamente.
- A exclusão remove perfil, suas chaves, escutas, relay/ponte legados por cascata,
  avatar sem outras referências e playing now em memória. Não exclui a conta
  administrativa, artistas, álbuns, faixas ou reproduções dos demais perfis.
- É permitido excluir o último perfil. A lista pública passa a vazia, `/admin`
  continua funcionando e pode criar um novo perfil com chave inicial.
- Arquivos já arquivados em `import_complete/<ID>/` permanecem no sistema de
  arquivos; não são escutas ativas e não são reatribuídos a novos perfis.

### Importação com destino explícito

- A seção “Import history” mostra nome, slug e ID do destino escolhido em
  “Manage profile”. Exige arquivo, formato e confirmação de que o histórico é
  dessa pessoa. Não usa a conta administrativa nem nomes no arquivo como destino.
- Upload na mesma rota administrativa de importação, com `multipart/form-data`,
  campos `format` e `file`, e limite de 100 MiB para a requisição. Formatos:
  `koito`, `maloja`, `spotify` e `lastfm` em JSON; `listenbrainz` usa o ZIP de
  exportação com entradas `listens/*.jsonl`, conforme o importador existente.
- O servidor gera um nome único com UUID e sufixo do formato selecionado. O nome
  original não determina caminhos nem parser e não sobrescreve arquivos existentes.
  O destino continua `import/<profileID>/`; diretório de perfil symlink é rejeitado.
- Sucesso arquiva o upload em `import_complete/<profileID>/`. Falha remove o arquivo
  pendente para impedir repetição automática no próximo início da aplicação.
- Os importadores existentes gravam progressivamente: erro ou interrupção pode
  deixar parte das escutas já salva. A interface informa essa limitação. O upload
  é síncrono, sem fila persistente ou barra de progresso; arquivos acima do limite
  podem usar o fluxo de arquivos locais já documentado no passo 2.
- Validado o conteúdo obrigatório do export Koito antes das gravações, evitando
  panic quando um arquivo enviado contém uma reprodução sem artistas.
- O link de exportação usa explicitamente o perfil administrado e exige sessão.

### Contrato HTTP acrescentado

Prefixo `/apis/web/v1`; todas as rotas abaixo exigem sessão administrativa e
continuam protegidas pela validação de origem das mutações:

| Método e rota | Resultado |
| --- | --- |
| `POST /profiles` | Cria perfil e chave inicial em transação; retorna metadados do perfil |
| `GET /profiles/{slug}/management` | Metadados e `listen_count` para revisão administrativa |
| `DELETE /profiles/{slug}` | Exclui com JSON `display_name` e `listen_count` obrigatórios |
| `PUT /profiles/{slug}/avatar` | Substitui avatar via multipart `icon` e `image` opcional |
| `DELETE /profiles/{slug}/avatar` | Remove foto/ícone e retorna perfil com iniciais |
| `POST /profiles/{slug}/import` | Acrescentado upload multipart `format` + `file`; preservado JSON `filename` |

As rotas administrativas de edição e chaves implementadas no passo 2 foram
reutilizadas. Na execução original do passo 4, não foi necessária outra migração
e o esquema estava na versão 4. Posteriormente, ele foi consolidado na versão 3.

### Validação realizada e continuidade

- `make check`: suíte Go completa, geração/checagem TypeScript e quatro testes
  nativos do frontend. Incluído teste para não enviar credenciais vazias ao
  alterar apenas nome ou senha da administração.
- `make client.build`: compilação de produção e assets gzip.
- Novos testes HTTP em `engine/profile_management_test.go`, também executados com
  `-race`: autorização, cadastro com primeira chave, rollback quando a chave falha,
  duplicatas/slugs/ícones inválidos, confirmação com contagem desatualizada,
  preservação de outro perfil e catálogo, exclusão do último perfil e novo cadastro.
- Testes de avatar cobrem upload real de PNG, leitura WebP, rejeição de conteúdo
  inválido, preservação pela limpeza global e remoção somente após a última referência.
- Testes de upload cobrem destino explícito, arquivamento, nome com tentativa de
  caminho, formato inválido, rejeição de visitante/origem externa, arquivo Koito
  inválido sem panic e limpeza do arquivo pendente após falha.
- No Chrome, em cópia descartável do banco de exemplo: cadastro com slug sugerido
  e ícone, chave inicial disponível/oculta, edição de nome e slug mantendo ID,
  edição do rótulo da chave e revisão de exclusão com nome/contagem e botão bloqueado
  antes da confirmação. A apresentação preserva os componentes e o tema existentes.
- O banco em uso, suas 72 reproduções, perfis, credenciais administrativas e chaves
  foram comparados com o estado anterior e preservados. Os processos de validação
  usam portas 14110/15173 e são encerrados ao concluir; a API habitual permanece parada.

Na conclusão do passo 4, a validação completa ainda estava pendente. O passo 5
abaixo registra sua execução com a API real e interface compilada, distinguindo
a validação do protocolo de scrobbling da configuração de um player externo.


## Registro da implementação — passo 5

### Ajuste de navegação solicitado

- Removido o ícone/botão administrativo separado da barra principal em
  `client/app/components/sidebar/Sidebar.tsx`.
- “Administration” foi colocado no menu interno de Settings, junto às suas
  seções. Inicialmente era um link para `/admin`; o ajuste posterior abaixo
  o substituiu por uma aba com conteúdo no próprio diálogo. Continua disponível para
  visitantes, que recebem o formulário de login; não concede acesso administrativo.
- `SettingsModal.tsx` preserva o estilo, o tema e o comportamento responsivo.
  O link é separado semanticamente do grupo de abas, e o diálogo passa a ter
  o nome acessível “Settings”. Não há duplicação de sessão ou de perfil atual.
- Conferido no Chrome em desktop e em viewport de 400 px, incluindo o clique
  em Administration e o fechamento de Settings ao abrir a página administrativa.

### Validação reproduzível com a aplicação real

Acrescentado `scripts/validate_multiuser.py`, usando somente a biblioteca padrão
Python 3, e o alvo `make multiuser.check`. O teste compila API e frontend, inicia
processos reais em loopback com portas temporárias, faz requisições HTTP e
reinicia a API para verificar a persistência. Não usa o banco definido em `.env`:
o script remove as variáveis `KOITO_*` herdadas e fornece configuração isolada,
com importação automática e provedores externos desabilitados.

```sh
# Compila e verifica instalação nova e migração de uma base legada v2 gerada no teste.
make multiuser.check

# Opcional: repete também com uma cópia de uma base existente.
# O arquivo informado é aberto somente para leitura; nenhuma credencial é impressa.
python3 scripts/validate_multiuser.py --binary ./koito \
  --example-db /caminho/para/koito.db
```

Sem `--work-dir`, os bancos/logs ficam em um diretório temporário removido ao
terminar. `--work-dir /caminho/novo` conserva cópias descartáveis para inspeção,
com diretório privado e logs 0600; o caminho deve ser novo. Todos os processos
iniciados pelo script são encerrados, inclusive se uma verificação falhar.

Cenários executados nesta etapa:

1. **Instalação nova:** banco criado pelo bootstrap real da aplicação. Login
   inicial, perfil inicial e criação de novos perfis funcionam sem login de ouvinte.
2. **Migração v2 → v4:** base legada de teste com administrador, senha, chave e
   reprodução. O primeiro início gera backup e preserva todos os dados legados;
   a chave antiga continua válida e a senha original permite login.
3. **Banco de exemplo:** cópia SQLite consistente do backup local anterior aos
   perfis, com as 72 reproduções. Hashes dos dados legados, inclusive senhas,
   tokens, sessões e catálogo, conferidos antes/depois da migração. Somente após
   essa conferência, a cópia recebe credenciais temporárias para repetir a jornada.
   O arquivo de origem permanece aberto apenas para leitura e inalterado.

Nos três cenários, `PRAGMA integrity_check` retornou `ok`,
`PRAGMA foreign_key_check` não retornou pendências e não existem escutas ou
chaves sem perfil. O reinício preservou histórico e chave ainda ativa.

### Critérios de aceite e evidências

| Critério | Resultado e evidência |
| --- | --- |
| Mesma faixa no mesmo segundo por dois perfis | Dois envios HTTP simultâneos resultam em dois registros, um em cada perfil |
| Duas chaves para uma pessoa | Chave inicial e chave adicional alimentam o mesmo histórico; revogar uma mantém a outra ativa |
| Payload não escolhe o destinatário | IDs de outro perfil no payload não substituem o perfil resolvido pela chave |
| Histórico, rankings e métricas de mídia | Conferidos os totais pessoais em histórico, rankings de faixas/artistas/álbuns e respectivas páginas de mídia |
| Atividade e retrospectiva | Soma dos buckets e total de `summary` conferidos separadamente para cada pessoa |
| Playing now | Evento aparece somente no perfil da chave; não vira scrobble salvo e não persiste após reiniciar |
| Importação/exportação | Exportação de duas escutas importada explicitamente no segundo perfil; duplicata da mesma escuta não aumenta o total |
| Perfil explícito | Rota antiga de estatísticas com vários perfis retorna 409; slug antigo após edição e perfil excluído retornam 404 |
| Administração protegida | Visitantes/chaves não acessam gestão, exportação ou importação; mutação com sessão e origem externa retorna 403 |
| Metadados públicos | Resposta pública de perfis não contém as chaves utilizadas no teste |
| Revogação e exclusão | Chave revogada retorna 401; histórico permanece; excluir o segundo perfil não altera o primeiro nem os perfis originais |
| Migração e persistência | Dados legados preservados, backup criado e novo login/chaves/histórico válidos após reinício da API |
| Último perfil e catálogo | Suíte administrativa confirma exclusão do último perfil, manutenção do administrador e catálogo e novo cadastro |
| Avatar | Suíte HTTP confirma upload/serviço WebP, proteção contra limpeza de imagens em uso e remoção após a última referência |
| URL e cache na interface | Duas abas independentes, atualização da página e troca de ranking com período preservado, conferidas no Chrome |

As verificações de protocolo usam o endpoint ListenBrainz real da aplicação,
com `Authorization: Token ...` e payload de scrobbling, sem alterar a configuração
do Navidrome ou de outro servidor musical em uso. Os testes de relay usam um
servidor HTTP local, já coberto pela suíte de perfis.

### Nome e senha: ponto de atenção resolvido

- Acrescentado `TestAdminAccountChangesKeepListeningProfile`, que altera só o
  nome administrativo, efetua novo login com o nome novo e a senha anterior,
  altera só a senha, rejeita a senha antiga e efetua login com a nova.
- O teste confirma que essas mudanças não alteram nome/slug/ID do perfil público
  nem invalidam sua chave de ingestão. Em seguida, muda o nome e o slug do perfil
  `default` e verifica que o login administrativo continua independente.
- A jornada com API real repete as mudanças e valida o novo login após reiniciar
  o processo, nos três cenários de banco. O teste frontend existente confirma que
  campos não alterados são omitidos da requisição, evitando enviar senha/nome vazios.
- Não foi necessário alterar novamente a implementação de credenciais do passo 4:
  o comportamento corrigido foi reproduzido e confirmado. Para alterar a identidade
  pública, use “Profile details”; para alterar o login/senha, use “Administrator account”.

### Verificações executadas

- `make check`: suíte Go completa, geração/checagem TypeScript e quatro testes frontend.
- `make multiuser.check`: build de API e frontend de produção com gzip e jornadas
  completas de instalação nova e migração v2, incluindo respostas das páginas SPA
  por acesso direto e disponibilidade dos assets JavaScript.
- Jornada adicional com `--example-db` apontando para o backup local pré-perfis:
  mesmos critérios exercitados em uma cópia das 72 reproduções de exemplo.
- `go test -race ./engine ./internal/db/sqlite -run 'Test(Profile|AdminAccount)'`,
  com `KOITO_MIGRATION_TEST_DB` apontando para o backup: passou, incluindo migração,
  isolamento, autorização, cadastro/exclusão, avatar, importação, relay, merge,
  paginação de exportação e atualização das credenciais.
- Chrome, com frontend compilado: seleção na raiz, login administrativo, duas
  abas com perfis distintos, troca lateral no ranking mantendo `period=all_time`
  e ano, atualização da página e Settings → Administration em desktop e 400 px.
  As páginas exibiram os dados corretos de cada perfil, sem mistura de rankings.
- `git diff --check`: sem problemas de whitespace.

### Preservação e limites

O banco em uso permaneceu na versão 4, com 72 reproduções e um perfil. A impressão
digital de todas as tabelas de dados foi comparada antes/depois, preservando também
senhas, chaves, sessões e catálogo. Os bancos descartáveis e processos usados na
validação foram removidos/encerrados. A API principal na porta 4110 continua parada;
o Vite que já estava aberto na porta 5173 não foi reiniciado nem encerrado.

Esta etapa conclui a validação local dos critérios do MVP. Permanecem os limites
já descritos: perfis públicos, catálogo compartilhado e importação síncrona com
limite de upload e possibilidade de gravação parcial em caso de erro. A configuração
de players externos reais e integrações de imagens/serviços remotos não foi alterada
nem validada contra contas externas nesta execução.

Na conclusão original do passo 5, o esquema estava na versão 4; a consolidação
posterior abaixo passou a usar a versão 3. As colunas `user_id`, a ponte e os métodos legados
seguem presentes como compatibilidade técnica; a identidade de escuta operacional
é `profile_id`. A remoção física dessa compatibilidade prevista no modelo proposto
ainda exige uma migração própria e atualização dos consumidores legados; não foi
incluída como mudança de esquema nesta etapa de validação. Isso não altera os
resultados dos critérios funcionais acima.


## Ajuste de interface — administração dentro de Settings

- Administration passou a ser `TabsTrigger`/`TabsContent`, como Appearance e
  Account. Seus formulários aparecem na área de conteúdo de Settings; clicar
  nessa opção mantém o diálogo aberto, a página pública e seu período.
- Extraído `AdministrationPanel.tsx`, compartilhado pelo painel integrado e pelo
  acesso direto `/admin`. A gestão de perfis, avatares, chaves e importação usa
  os mesmos componentes e endpoints administrativos existentes.
- No painel integrado, a escolha do perfil administrado usa estado local, sem
  acrescentar/remover parâmetros da URL pública. Ao usar `/admin` diretamente,
  a seleção continua disponível em `?profile=<ID>`.
- Login e logout atualizam a sessão pelo `AppProvider`, sem recarregar a página
  ou fechar Settings. A aba Account mantém os controles de login e senha da
  conta administrativa, separados dos perfis públicos.
- Ao fechar Settings depois de visitar Administration, as consultas e os loaders
  da página pública são atualizados para refletir as alterações. O link explícito
  para visitar um perfil fecha o painel e abre a página selecionada.
- Validado com TypeScript, os quatro testes frontend e build de produção. No
  Chrome, em banco descartável: Administration no painel, login sem recarregar,
  seleção de perfil preservando a URL e período, troca para Account e logout
  mantendo Settings aberto. A apresentação também foi inspecionada em 400 px.
- Conferida também a versão de desenvolvimento em porta separada: aba interna,
  fechamento do diálogo e retorno do foco para Settings.
- Nenhuma alteração de banco ou permissão foi necessária para esse ajuste.


## Teste de atualização sobre instalação antiga — banco `.sqlexample` v2

Registro histórico anterior à consolidação: neste teste havia migrações separadas
3 e 4. O fluxo atual usa somente a migração 3, conforme o registro seguinte.

Executado com `.sqlexample/koito.db`, confirmado na versão de migração 2 antes
do teste. A fonte foi aberta somente para leitura e copiada com a API de backup
SQLite, incluindo WAL. O binário compilado do código atual foi iniciado sobre
essa cópia como em uma atualização de instalação; o teste não aplicou SQL de
migração diretamente, não redefiniu a senha e não gerou novas chaves.

**Resultado: migração automática v2 → v4 aprovada.**

| Dados | Antes | Depois |
| --- | ---: | ---: |
| Usuários | 1 | 1 |
| API keys | 1 | 1 |
| Sessões | 1 | 1 |
| Escutas | 72 | 72 |
| Artistas | 18 | 18 |
| Álbuns | 19 | 19 |
| Faixas do catálogo | 76 | 76 |

- Aplicadas automaticamente `003_listening_profiles.sql` e
  `004_profile_relays.sql`. Criado `/default`, com as 72 escutas existentes e
  67 faixas efetivamente ouvidas; a diferença para as 76 faixas de catálogo
  já existia na base e não representa perda durante a migração.
- Backup automático pré-migração gerado na versão 2, com permissão 0600 e hashes
  de dados iguais aos da origem. Dados legados comparados por tabela, incluindo
  hash da senha, token e sessão, sem expor suas credenciais no relatório.
- A API validou a chave antiga no perfil migrado. Leituras públicas de histórico,
  estatísticas, rankings, atividade, resumo e playing now responderam normalmente.
  A chave de ingestão não autorizou gestão administrativa nem apareceu nos perfis públicos.
- `PRAGMA integrity_check`: `ok`; `PRAGMA foreign_key_check`: sem pendências;
  nenhuma escuta ou chave ficou sem perfil. Conferida a PK de escutas
  `(profile_id, track_id, listened_at)`.
- Segundo início: versão 4 mantida, sem nova migração, novo backup, perfil duplicado
  ou alteração nos hashes dos dados persistidos.
- Arquivo fonte `.sqlexample/koito.db` e seu WAL conservaram os mesmos hashes
  antes/depois. A instalação em uso não foi modificada. A API de teste foi encerrada.
- A senha não foi usada em um login: sua preservação foi comprovada pela igualdade
  do hash armazenado. Não houve reset de senha nem gravação de scrobbles neste teste.

Artefatos conservados para inspeção:

- Banco migrado: `.local/upgrade-sqlexample-v2/koito.db`.
- Backup v2: `.local/upgrade-sqlexample-v2/koito.db.pre-profiles-2790378440.bak`.
- Relatórios: `.local/upgrade-sqlexample-v2/REPORT.md` e `report.json`.
- Log do início/migração e reinício: `.local/upgrade-sqlexample-v2/server.log`.

Acrescentado `scripts/validate_upgrade.py` para repetir exatamente esse teste sem
alterar a base fonte. O diretório de trabalho informado deve ser novo:

```sh
CGO_ENABLED=1 go build -o /tmp/koito-upgrade-sqlexample ./cmd/api
python3 scripts/validate_upgrade.py \
  --binary /tmp/koito-upgrade-sqlexample \
  --source .sqlexample/koito.db \
  --work-dir .local/upgrade-sqlexample-v2-retest
```

Para atualizar uma instalação real, a versão atual deve receber o mesmo
`KOITO_CONFIG_DIR` da antiga, conservando o banco, cache de imagens e demais
arquivos dessa instalação. A migração ocorre ao iniciar a API. Este teste validou
esse comportamento em cópia, sem substituir o banco da instalação em uso.


## Consolidação autorizada — migração única v2 → v3

A pedido do usuário, após restaurar o desenvolvimento para um banco na versão 2,
as alterações multiusuário foram reunidas em uma única migração 3. Essa decisão
substitui, neste ciclo de desenvolvimento, a organização experimental em duas
migrações registrada nas seções anteriores; não renumera o histórico de um banco
já migrado.

### Alterações realizadas

- `003_listening_profiles.sql` agora contém também `profile_relays`,
  `legacy_relay_binding` e `trg_api_key_profile_immutable`, anteriormente em
  `004_profile_relays.sql`. O arquivo 004 foi removido.
- Perfis, associação dos dados legados, reconstrução das escutas/chaves, índices,
  relay e proteção contra transferência de chave são aplicados na mesma transação
  Goose. Falhar qualquer trecho desfaz todo o avanço 2 → 3.
- As migrações 001 e 002 não foram alteradas. Instalações novas aplicam 001, 002 e
  003; bases na versão 2 aplicam somente 003, automaticamente ao iniciar a API.
- O backup automático anterior à migração continua preservando o banco v2.
  A proteção contra downgrade destrutivo também foi mantida.
- Testes SQLite que dependiam da versão 4 agora usam a versão 3. As verificações
  com API real comparam a versão final com a última migração disponível, detectando
  inclusive o uso acidental de um binário antigo nos testes.
- Acrescentado teste de falha tardia: um conflito na criação da tabela de relay
  deve reverter também a criação de perfis e a reconstrução de escutas/chaves.
  Após remover o conflito, uma nova tentativa precisa concluir na versão 3,
  incluindo relay funcional e proteção da chave contra transferência.

Bases experimentais já migradas pela organização anterior (antiga v3 ou v4)
devem ser restauradas de um backup v2 antes de usar esta consolidação, como foi
feito no desenvolvimento. Não alterar manualmente `goose_db_version` nem apenas
rebatizar um banco antigo v4 como v3; isso não reaplica o conteúdo da migração.
Os relatórios antigos v2 → v4 foram mantidos como histórico, não como evidência
da versão atual.


### Validação da consolidação

- Suíte Go completa passou, incluindo o teste de rollback tardio seguido de nova
  tentativa bem-sucedida. Também passaram os testes de migração, perfis, relay e
  credenciais com detector de corridas (`-race`).
- Jornada HTTP completa passou em instalação nova, base v2 gerada pelo teste e
  cópia do banco v2 de desenvolvimento: cadastro, chaves, envios simultâneos,
  isolamento, consultas, importação/exportação, revogação/exclusão e reinício.
  A versão final conferida nos três cenários é 3.
- Teste específico de atualização com o binário atual sobre cópia de
  `.local/koito/koito.db`: log contém somente `003_listening_profiles.sql` e
  registra migração concluída na versão 3. Preservados usuário, hash da senha,
  chave antiga, sessão, 72 escutas, 18 artistas, 19 álbuns e 76 faixas do catálogo.
- Backup automático v2 confirmado, sem violações de FKs ou dados sem perfil;
  segundo início sem mudança de dados, nova migração ou novo backup.
- Relatório e banco migrado conservados em `.local/upgrade-consolidated-v3/`.
  O banco real de desenvolvimento permaneceu em v2 com as 72 escutas, preparado
  para aplicar automaticamente a migração 3 na próxima inicialização da API.
- O executável local `./koito` foi recompilado com a migração consolidada.
  As APIs de teste foram encerradas; a API principal não foi iniciada.

O diretório antigo `.sqlexample` não estava mais disponível nesta execução;
a origem dos testes adicionais foi o banco v2 restaurado em desenvolvimento,
sempre aberto apenas para leitura. Nenhum arquivo de versão do banco foi
editado manualmente para simular a consolidação.


## Administração do relay por perfil — 1 de novembro de 2026

Settings → Administration → Manage profile agora inclui ListenBrainz forwarding:
checkbox para ativar, URL base, token externo em campo aberto, indicação de estado
salvo/alterações pendentes, Save forwarding e Test connection. A interface utiliza
os endpoints PUT e DELETE existentes por perfil. Salvar desativado remove a URL
e o token persistidos; para reativar depois é necessário preenchê-los novamente.
O histórico local é preservado.

Novo GET administrativo carrega URL/token salvos com `Cache-Control: no-store`.
Respostas públicas de perfis continuam sem credenciais de relay. O novo POST
`/relay/test`, também administrativo, consulta `/validate-token` na URL do
formulário com o token no Authorization, informa o usuário externo quando válido
e não salva credenciais nem envia reproduções. Aceita somente URL base HTTP(S)
sem credenciais embutidas, query ou fragmento, limita tempo a 10 segundos e o
tamanho da resposta, e não segue redirects. Erros externos são resumidos sem
retornar tokens ou respostas brutas. O protocolo segue a
[documentação ListenBrainz](https://listenbrainz.readthedocs.io/en/latest/users/api/core.html#get--1-validate-token).

Não requer migração de banco. As variáveis globais legadas mantêm a associação
única de bootstrap; desativar remove o relay persistido do perfil. Fila persistente
e alterações nas tentativas de entrega permanecem para uma etapa posterior.

Validação: suíte Go completa, incluindo autorização, leitura/remoção por perfil,
credenciais inválidas, resposta inválida, redirects e teste sem persistência;
typecheck, quatro testes do frontend e build de produção passaram. Os testes
usaram bancos descartáveis e servidores locais simulados, sem enviar dados
a serviços externos nem alterar a configuração real de relay.

## Numeração do TuneTrail

A versão atual usa `0.x`, a partir de TuneTrail 0.3. `make version.bump` avança o segundo número, inclusive de 0.9 para 0.10; 1.0 depende de decisão futura. O CI valida o incremento e aceita a migração inicial da numeração SUM. O Koito original usado como base permanece na versão 0.3.2, independente da numeração do TuneTrail. Tags e notas SUM existentes são registros históricos.

A validação histórica de SUM confirmou os assets em RC2 e exercitou RC2 → RC3 em uma cópia temporária.


## Checklist do PR: documentação e avisos

Atualizados os guias de uso do site para perfis, migração, chaves, importação e
relay. A API experimental Node dos testes foi substituída pelo compilador
TypeScript existente; corrigida uma tag JSON malformada de teste anterior e
incluído Go vet em `make check`. O [registro de validação](../../../docs-dev/PR_VALIDATION.md)
contém comandos, resultados e avisos antigos das ferramentas/dependências.

O site de documentação compilou (9 páginas), comparado com a base `a079fa6`
(8 páginas): ambos emitem o mesmo aviso antigo Node DEP0190. Testes/build da
aplicação e verificação do Dockerfile ficaram sem avisos. Os dois itens
correspondentes do checklist do PR estão marcados, com esses limites registrados
no relatório de validação.
