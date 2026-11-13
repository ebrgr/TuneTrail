---
id: doc-007
title: Roadmap 4 — administração, importação, chaves, edição, deleção e mesclagem
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_04_ADMINISTRACAO.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Roadmap 4 — administração, importação, chaves, edição, deleção e mesclagem

Estudo de 5 de novembro de 2026 sobre a árvore de trabalho atual, incluindo as
alterações locais das etapas anteriores. **Issues 010–015 implementadas e integradas
no modo Go opcional; testes, revisão Advisor e 55 capturas registrados. Aceite
do usuário concedido em 05/11/2026; issues 010–015 fechadas e resolvidas.** Issues 001–009 permanecem encerradas pelo aceite do usuário.
A revisão do [Advisor](../../../.agents/advisor.md), no modelo Sol com esforço alto, integra
este estudo. As [evidências](../../evidencias/roadmap-04/README.md) distinguem leitura,
reprodução e verificações futuras.

A base vigente é **SQLite com esquema consolidado v3**. A migração da apresentação
não exige, por si só, uma migration. Não alterar as migrações 001–003 nem abrir o
banco em operação para executar testes destrutivos. O serviço operacional permanece parado; somente processos descartáveis foram usados nos testes.

## Estado após a integração da 015

[Entrega e evidências](../../evidencias/roadmap-04/ISSUE_015.md). Administração e
gestão agora são documentos Go quando `KOITO_WEB_UI=go`, com actions POST
protegidas. React continua padrão e fallback. O restante deste arquivo preserva
o estudo e seus critérios; pendências descritas no futuro devem ser lidas junto
ao relatório final, incluindo os limites conhecidos. O usuário aceitou e encerrou as issues 010–015 em 05/11/2026.

## Conclusão e escopo

A maior parte das operações existe no backend, mas parte das regras está dentro
de handlers HTTP. É necessário extrair serviços compartilhados para que HTML e
JSON executem as mesmas regras, sem chamadas HTTP internas e sem adaptar um
formulário para invocar um handler JSON como serviço.

Além da apresentação, há dois bloqueadores funcionais reproduzidos: mesclagem de
artista consigo mesmo apaga dados e mesclagem de álbuns bloqueia com uma conexão
SQLite. Corrigi-los explicitamente antes de migrar as ações destrutivas. O aceite
da etapa 3 não significa que todos os casos adversariais do backend foram testados.

Esta etapa abrange conta/sessão administrativa; gestão dos perfis e avatares;
chaves de ingestão e relay; importação/exportação; aliases, MBIDs, artistas e
compilações; imagens; adição/exclusão de escuta; exclusão e mesclagem de catálogo.
A política de autorização continua sendo administrador único versus perfis públicos.
Contas individuais de ouvintes, importação remota periódica, fila durável de relay,
importação em segundo plano e eliminação de React são trabalhos separados.

## Inventário do que existe

| Fluxo | Interface atual | Backend e semântica |
| --- | --- | --- |
| Entrar/sair e alterar conta | `LoginForm`, `Account`, `/admin`, Settings → Account | `handlers/auth.go`, `sqlite/user.go`, `sessions.go`: sessão administrativa; username/senha não renomeiam perfis |
| Perfis | `AdminProfiles`: `/admin?profile={id}` e painel embutido | Criar gera perfil e chave inicial atomicamente; editar nome/slug conserva ID e história; URL anterior vira 404 |
| Avatar | Foto, ícone ou iniciais | `ProfileAvatarHandler`: multipart até 5 MiB; nova foto tem prioridade; salvar sem foto usa ícone/iniciais; limpeza confere referências |
| Chaves | `AdminProfiles` e `ApiKeysModal` | Listar, revelar/copiar, criar, renomear rótulo e revogar; chave não pode mudar de perfil e não autentica administração |
| Relay | `ProfileRelayEditor` | Configurar/remover URL e token por perfil; testar rascunho via `validate-token`, sem salvar e sem enviar escuta |
| Importação | `AdminProfiles`, confirmação de destino | POST multipart por perfil, **síncrono**, até 100 MiB na requisição; Koito/Spotify/LastFM/Maloja JSON e ListenBrainz ZIP |
| Exportação | Link do perfil e `ExportModal` | GET administrativo por perfil, arquivo Koito JSON v1 transmitido em páginas de 1000 registros |
| Edição de mídia | `EditModal` em `/{slug}/manage/{kind}/{id}` | Aliases de artista/álbum/faixa; definir principal; MBID; vários artistas em álbum; relações de artistas conforme operações abaixo |
| Artistas de uma faixa | `ArtistManager` | Associar artistas existentes, marcar/desmarcar principal e remover associação de faixa |
| Artistas de um álbum | Mesmo componente | Marcar/desmarcar principal; a UI não oferece adição/remoção de artista de álbum |
| Imagem | `ImageReplaceModal` e `ImageDropHandler` | URL ou upload; sugestão Cover Art Archive para álbum com MBID; imagem da faixa é a do álbum, com ID explícito desde a 009 |
| Escuta manual/exclusão | `AddListenModal`, `ListensTable`, `static/live.js` | POST/DELETE administrativos com perfil explícito; exclusão identifica perfil + faixa + timestamp |
| Mesclagem | `MergeModal` | Pesquisa, direção reversível e troca opcional de capa para artista/álbum; catálogo global e deduplicação pessoal das escutas de faixa |
| Exclusão de mídia | `DeleteModal` | Artista/álbum/faixa globais, com cascatas/limpeza; não equivale a remover somente uma escuta |
| Exclusão de perfil | `AdminProfiles` | Nome + contagem conferidos no DELETE atômico; remove perfil, escutas, chaves e relay; preserva catálogo e administrador |
| Limpeza global | API `DELETE /apis/web/v1/data` | Apaga escutas/faixas/álbuns/artistas de todos os perfis; não há tela correspondente encontrada no cliente |

Não há controle de duração no editor atual: `UpdateTrackHandler` aceita MBID;
`UpdateTrackOpts` interno possui outros campos, mas isso não constitui edição
oferecida pela UI. Adicionar edição de duração seria funcionalidade nova. A UI
atual de MBID não oferece um contrato explícito para limpar o valor; definir
representação e testes antes de anunciar essa capacidade.

A limpeza global continuará acessível pela API existente. Uma nova tela de
purga não é necessária para paridade e não está incluída neste plano.

## Escopos e autorização

| Tipo de dado/ação | Escopo que deve aparecer na tela e na confirmação |
| --- | --- |
| Perfil, avatar, chaves, relay, import/export, escuta | Perfil resolvido na rota; trocar perfil muda o destino das ações seguintes |
| Nome/aliases/MBID/capa/relações de mídia | Catálogo compartilhado; alterações podem aparecer em todos os perfis |
| Mesclar/excluir mídia | Global, inclusive impacto em escutas de outras pessoas; calcular efeito das relações/cascatas reais |
| Conta/sessões | Administrador; independente do perfil selecionado e recuperável mesmo com zero perfis |

Preservar `RequireAdmin` e a separação de `IngestionKey`: visitante, chave de
player e cookie inválido não autorizam mutações. Resolver perfil sem cookie de
"perfil corrente". Formulário aberto em uma aba deve continuar apontando para
seu perfil, mesmo após selecionar outro em outra aba. Revalidar identidade e
existência no servidor; campos ocultos são dados não confiáveis.

Em edição de catálogo, o perfil é contexto de navegação, não dono do item. Leitura
pessoal continua exigindo escutas relacionadas; a busca pública permanece pessoal.
Se for necessária busca global para selecionar destino de merge/artista existente,
criá-la apenas atrás da sessão administrativa. A busca atual não prova que um item
sem escutas naquele perfil inexiste no catálogo.

## Arquitetura e contratos HTML propostos

- Extrair serviços pequenos por operação para um pacote como `internal/admin/`,
  com interfaces de store mínimas; manter a transação na persistência e reaproveitar
  `catalog`, `importer`, `export` e cache de imagens. Não copiar SQL para templates.
- Serviços e adaptadores JSON/HTML compartilham validação de IDs, autorização,
  resultados e erros. Rotas HTML ficam explicitamente atrás dos mesmos controles;
  não confundir `webui.IsPage` (classificação de documento) com autenticação.
- Novas páginas: `/admin` conserva o bookmark `?profile={id}`; páginas por seção
  podem usar query. Preservar `/{slug}/manage/{artist|album|track}/{id}`. Reservar
  novas actions POST explícitas em `/admin/...` e nessas rotas de gestão, sem
  habilitar mutação em GET nem um override genérico de métodos.
- Formulários funcionam sem JavaScript: GET para leitura/busca, POST para ações,
  multipart para arquivo, resposta 303 após sucesso e mensagem de resultado sem
  segredos. JS pequeno melhora copiar/revelar, drag-and-drop e feedback; não é
  requisito para confirmar uma exclusão ou escolher a direção de mesclagem.
- Preservar `/apis/web/v1/...` e seus métodos/formatos: revisão/confirmação em duas
  fases é requisito das novas actions HTML; as APIs atuais não recebem
  obrigatoriamente um plano prévio ou token CSRF novo. Elas compartilham as
  invariantes, autorização, validações e transações corrigidas. Ajustes de contrato
  necessários aos bloqueadores devem ter regressão e changelog, não mudanças
  silenciosas só na nova UI.
- Usar `html/template`, limites de corpo antes do parsing, validação de tipo,
  nomes válidos, IDs positivos em int32 e timestamps explícitos. Valores repetidos e
  parâmetros incompatíveis são rejeitados; não truncar overflow por cast.
- Separar 400/422 de validação, 401 sem sessão, 403 sem permissão/CSRF, 404 ausente,
  409 confirmação desatualizada e 500 infraestrutura, com texto público genérico.
  Um erro de banco não deve virar ausência nem sucesso com zero linhas afetadas.
- `Cache-Control: no-store` em HTML administrativo, downloads pessoais e respostas
  com chaves/token. Nunca incluir credenciais em query, redirect, flash, logs ou
  screenshots. Ocultar no HTML é apresentação, não criptografia de segredos.
- GET de login pode preservar retorno interno validado; após POST, permitir somente
  destinos conhecidos da administração/gestão do mesmo aplicativo. O `safeNext`
  atual cobre rotas públicas legadas; não o ampliar indiscriminadamente nem aceitar
  `//host`, URL absoluta, escapes ambíguos ou retorno para action de escrita.

### Sessão e proteção de formulários

Hoje o login usa bcrypt, cookie HttpOnly e SameSite Strict; Secure depende de TLS
ou do cabeçalho de proxy. `SameOrigin` confere Origin quando presente e bloqueia
`Sec-Fetch-Site: cross-site`. Essas proteções devem ser preservadas. Não foi
reproduzido ataque CSRF e não se afirma que a proteção atual seja explorável.

Para as novas actions HTML, especificar token CSRF vinculado à sessão, comparação
constante e rejeição antes de executar trabalho ou gravar arquivo. No login usar
proteção pré-autenticação/nonce próprio, pois ainda não existe sessão admin;
aplicar também a logout e às forms multipart. Para uploads, não chamar
`ParseMultipartForm` antes de validar o token, pois ele pode gravar temporários.
Usar leitura multipart sequencial com token limitado validado antes da parte de
arquivo (rejeitar arquivo recebido antes do token), ou mecanismo equivalente que
funcione sem JavaScript e não grave upload antes da validação. Origin/Fetch Metadata continuam
como defesa adicional. Não aceitar um token arbitrário apenas por vir em hidden
input, nem reaproveitar a chave de ingestão como token CSRF.

Definir antes da implementação prazo fixo versus deslizante. Recomendação: manter
as opções 24h/30d com cookie e banco coerentes e revogar sessões na alteração de
senha, exigindo novo login. Isso muda comportamento e precisa de aceite/testes
na issue 010. Não alterar primitivas criptográficas ou confiar em cabeçalhos de
proxy enviados por origem não confiável sem definir a implantação suportada.

### Importação e exportação

Manter o contrato **síncrono** na primeira migração. Não existe job persistente,
progresso consultável ou botão administrativo de cancelamento no backend atual.
Usar contexto da requisição e cancelamento cooperativo nas consultas, loops,
requests e esperas. Fechar a aba pode cancelar parte do trabalho; não garante
interrupção instantânea nem rollback de registros já gravados.

Relatar resultado parcial ao cliente somente enquanto conectado; após desconexão,
o contexto pode encerrar o processamento e não existe relatório recuperável por
job. Logs operacionais podem registrar contagens seguras, sem arquivo/credenciais.

O formulário deve mostrar nome, slug e ID do destino e exigir confirmação para
o arquivo selecionado. Identidade dentro do arquivo não muda o perfil. Não repetir
POST automaticamente após timeout, 401 ou erro parcial. Manter upload com nome
gerado pelo servidor, permissões restritas e limpeza correta. Diferenciar os
arquivos arquivados em `import_complete/` no sucesso dos temporários removidos no
erro; o fluxo HTTP não deve deixar um arquivo pendente para reimportação no boot.

Os cinco parsers continuam reutilizados; preservar janelas de importação e
compatibilidade de formatos. O teto de 100 MiB é do corpo HTTP, incluindo envelope
multipart, e não limita sozinho bytes descompactados, quantidade de membros,
linhas ou alocação de JSON. Definir/testar orçamentos por formato, throttling
cancelável e resultado com importados/duplicados/ignorados/erros. Contagem de
entrada processada não é necessariamente quantidade de novas escutas gravadas.

Export continua administrativo e pessoal, com a paginação de timestamp + faixa
existente e download direto, sem carregar todo o arquivo no navegador. Especificar
snapshot ou limite de consistência durante ingestão concorrente. Propagar erros
de escrita/cancelamento; depois de iniciar o arquivo, não anexar JSON de erro ao
conteúdo parcial nem anunciar integridade/completude sem terminar a geração.

### Edição, imagens e escutas

Aliases e relações devem preservar ao menos as invariantes do backend: não apagar
alias principal como se fosse secundário, não fabricar entidades ao selecionar
uma relação e não copiar metadados de outra mídia por ID extraído da URL.
Validar MBID/compilação e manter o estado de formulário após erro de campo.

Usar identidade de imagem explícita: artista → artista; álbum → álbum; faixa →
álbum real. Foto de perfil e capa de catálogo têm associações diferentes. Um
arquivo novo pode existir antes de o banco confirmar a troca: planejar compensação
se falhar e remover imagem anterior somente após commit e verificação global de
referências. Nunca tratar erro de limpeza posterior como rollback de banco.

Download de imagem e teste/envio de relay precisam de política explícita de
rede: contexto, timeout, limite de bytes e redirects. O Koito suporta destinos
self-hosted; não bloquear silenciosamente toda rede privada. Definir política
configurável/suportada e aplicá-la em DNS/redirects tanto no teste quanto no envio.
Não há reprodução de SSRF neste estudo. A credencial externa fica somente no
formulário administrativo protegido e não entra na confirmação pública.

Escuta manual deve mostrar perfil/faixa e interpretar datetime-local no fuso
selecionado, incluindo ambiguidades de DST. Exclusão deve confirmar aquele
registro e afetar somente perfil + faixa + instante. Rever `live.js` para que
JS e formulário compartilhem o serviço; oferecer alternativa sem JS e retorno
que preserve filtros/página válidos. Invalidar tocando agora quando necessário.

### Deleções e mesclagens

Há transações nos merges, mas isso não basta: validar origem/destino positivos,
distintos e existentes dentro da transação; fazer leituras pelo mesmo `tx` e
propagar erros de Scan/Rows. Corrigir a espera de `MergeAlbums` com leitura
transacional; mover a leitura para antes de BeginTx não garante consistência.

Nas novas actions HTML, a confirmação deve nomear origem, destino e qual ID sobrevive; direção reversa
altera esses dados. Exibir impacto global, perfis afetados e efeito esperado em
escutas/aliases/relações/imagens. Recalcular/revalidar o plano ao gravar e devolver
409 se mudou desde a revisão. Contar somente o perfil da URL é insuficiente.
Esse contrato de revisão em duas fases pertence ao HTML. Para a API JSON atual,
preservar payloads e aplicar as invariantes transacionais; exigir plano prévio
ou novo CSRF também no JSON seria mudança funcional própria, coordenada com
React e clientes, e não faz parte da promessa de paridade transparente.

Preservar a deduplicação da faixa somente quando perfil e timestamp coincidem.
Escutas de pessoas distintas no mesmo segundo permanecem. A contagem total pode
cair por duplicatas pessoais; não prometer conservar sempre a soma. Definir e
testar quais aliases/artistas/MBIDs/capa vencem; não afirmar que os merges atuais
já combinam todos os metadados da origem.

Exclusão de álbum propaga para faixas/escutas; exclusão de artista depende das
relações/cascatas reais. A confirmação não pode generalizar ambas. Perfil excluído
preserva catálogo; mídia excluída pode afetar todos os perfis. Testar atomicidade,
falhas injetadas e ingestão concorrente em SQLite descartável.

Após merge, tentar página pessoal do sobrevivente apenas se houver relação com
o perfil; caso contrário ir para home/histórico do perfil. Após excluir a mídia,
não usar `history.back()` como destino. Limpeza de arquivos e memkv segue commit,
sem apresentar reversão inexistente. Não acrescentar undo, auditoria persistente
ou job transacional sem escopo e persistência próprios.

## Achados e prioridade

| Prioridade | Achado | Evidência e ação |
| --- | --- | --- |
| Bloqueador | Merge de artista consigo mesmo remove dados | **Reproduzido**: `MergeArtists(1,1,false)` retornou nil e reduziu 4 escutas a 2 em catálogo ouvido por dois perfis; `artist.go:582`; rejeitar identidade igual antes de mutar e revalidar dentro do tx |
| Bloqueador | Merge de álbum espera outra conexão após abrir tx | **Reproduzido**: `MergeAlbums(1,2,false)` atingiu deadline de 250 ms em `NewInMemory`; 4 escutas preservadas; `album.go:572–579`, helper `sqlite.go:150`; tornar consultas tx-aware |
| Correção | Merge de faixa ignora erros ao obter álbuns; erros SQL podem ir à resposta | Leitura: `track.go:588–589`, `handlers/merge.go`; distinguir validação/ausência/infra e testar rollback |
| Contrato | Sessão é renovada no banco por 30d sem alinhar cookie; senha não revoga sessões | Leitura: `middleware/authenticate.go`, `handlers/auth.go`, `sqlite/user.go`; definir contrato único na 010 |
| Correção | Chaves admin são retornadas sem no-store | Leitura: `handlers/apikeys.go`; proteger cache no JSON e HTML; relay já usa no-store |
| Limite | Import síncrono, JSON em memória, ZIP sem orçamento expandido e Sleep sem cancelamento | Leitura: `engine/profile_import.go`, `internal/importer/`; definir recursos/cancelamento e falha parcial na 012 |
| Correção | Export ignora algumas escritas e tenta erro JSON após começar arquivo | Leitura: `internal/export/export.go`, `handlers/export.go`; tratar falha de stream e teste de writer com erro |
| Correção | Teste e envio de relay têm políticas diferentes de redirects e leitura | Leitura: `profile_relay.go` bloqueia redirect/limita corpo; `lbz_submit_listen.go` usa cliente e leitura distintos; alinhar sem prometer entrega durável |
| Correção | Troca de capa não confere referência da imagem antiga; requests remotos sem limites coerentes | Leitura: `replace_image.go`, `images/imagesrc.go`, `imagecache/imagecache.go`; comparar com `cleanupProfileImage` e testar compensação/referências |
| Paridade | Alguns formulários React encerram loading antes da Promise; exclusões de mídia têm confirmação simples | Leitura: `AliasManager`, `MergeModal`, `DeleteModal`; implementar estado coerente e confirmação global explícita |

As reproduções não são testes HTTP, de UI ou de segurança. O programa, resultado
e procedimento estão em [comparisons/roadmap-04](../../evidencias/roadmap-04/README.md).
Os demais achados são leitura de código, sem exploração demonstrada.

## Organização e sequência

```text
internal/admin/                 # serviços compartilhados de conta e operações
internal/webui/admin*.go        # páginas, formulários e composição protegidos
internal/webui/manage*.go       # gestão das mídias existentes
internal/webui/templates/       # admin, keys, import, edit e confirm
internal/webui/static/          # melhorias pequenas de formulário/upload
engine/handlers/               # adaptadores JSON sobre os mesmos serviços
internal/db/sqlite/             # operações transacionais e impacto
scripts/validate_admin_webui.py # jornadas futuras sobre bases descartáveis
backlog/evidencias/roadmap-04/ # evidências e pares visuais futuros
```

Árvore proposta, sem criar abstrações vazias. Não introduzir dependência circular:
`internal/admin` não importa `engine/handlers`; rever a dependência atual do parser
ListenBrainz em tipos de `handlers` ao extrair serviços de importação.

1. [010 — contratos e bloqueadores de escrita](<../../tasks/issue-010 - 010-—-Estabelecer-contratos-administrativos-e-corrigir-bloqueadores-de-escrita.md>):
   identidade/sessão/CSRF, serviços compartilhados, erros e correções funcionais
   explícitas antes das novas telas.
2. [011 — administração, perfis e chaves](<../../tasks/issue-011 - 011-—-Migrar-conta-perfis-avatares-chaves-e-relay-para-Go.md>):
   conta, CRUD de perfis, avatar, chaves e configuração/teste de relay.
3. [012 — importação/exportação](<../../tasks/issue-012 - 012-—-Migrar-importação-e-exportação-por-perfil-para-Go.md>): formulários,
   parsers reutilizados, cancelamento/limites, resultado parcial e download.
4. [013 — edição, imagens e escutas](<../../tasks/issue-013 - 013-—-Migrar-edição-de-mídia-imagens-e-ações-de-escutas-para-Go.md>): aliases,
   MBIDs, relações, capa/upload, escuta manual e exclusão pessoal.
5. [014 — deleção e mesclagem globais](<../../tasks/issue-014 - 014-—-Migrar-confirmações-de-deleção-e-mesclagem-para-Go.md>): plano de
   impacto, confirmação/revalidação, transações, limpeza e retorno.
6. [015 — integração e aceite](<../../tasks/issue-015 - 015-—-Integrar-administração-Go-e-registrar-jornadas-e-aceite-visual.md>): ponte,
   jornadas, regressão e capturas; ativar famílias completas após aceite.

As issues são arquivos Markdown locais, sem publicação. As entregas 010–014
foram compostas na [015](../../evidencias/roadmap-04/ISSUE_015.md), com jornadas e
comparação visual registradas. Todas foram encerradas por aceite do usuário em 05/11/2026; a administração
está disponível no modo Go opcional. O checklist abaixo preserva os critérios
do estudo e deve ser confrontado com a cobertura e limites do relatório final.

## Aceite e comparação planejados

Usar bases v3 descartáveis com catálogo compartilhado por dois perfis, um vazio,
mesmo timestamp em pessoas distintas, duplicatas pessoais, relações múltiplas,
imagem compartilhada, nomes longos/Unicode, última conta/perfil e dados alterados
entre revisão e submissão. Sessões e chaves reais do usuário não entram nos testes.

Exigir visitante/chave/cookie expirado/não-admin, token CSRF ausente/inválido,
Origin/Fetch Metadata, login/logout e senha alterada; sigilo de chave/token em
cache/URL/log; conta independente de seleção de perfil; duas abas e bookmark após
renomear/excluir perfil. Testes de permissões e isolamento pertencem a cada issue,
mesmo com a revisão transversal do roadmap 5 ainda futura.

Cobrir import dos cinco formatos, ZIP/JSON inválido e grande, falha intermediária,
repetição do mesmo arquivo, duplicatas, cancelamento e reinício; export >1000,
mesmo segundo, nome escapado, erro de escrita e isolamento. Simular rede externa
localmente para imagem/relay; não testar tokens pessoais nem enviar escutas reais.

Edição e destruição: ID inválido/ausente, alias principal, relações, MBID/compilação,
falha de imagem/commit/limpeza, escuta em fuso/DST, merge reverso, self merge,
existência e plano alterados, rollback e contagem final de cada perfil. Completar
os casos de arrastar arquivo, teclado e edição de todos os tipos que a 009 deixou
registrados como não executados; o encerramento aceito da 009 é preservado.

| Pares React/Go futuros | Estados |
| --- | --- |
| Login e conta | Anônimo, erro, sucesso, sessão expirada, retorno interno |
| Administração/perfis | Zero perfis, criar, renomear, foto/ícone/iniciais e erro |
| Chaves/relay | Lista vazia, revelar/copiar, revogar, teste/salvar/remover; segredos fictícios ocultados |
| Import/export | Destino confirmado, formato, execução, falha parcial e download |
| Edição/capas | Artista/álbum/faixa, alias/relações/MBID, URL/upload e erro |
| Deleção/merge | Impacto global, direção reversa, cancelamento, confirmação desatualizada e retorno |

Capturar desktop/celular, Yuu/Pearl, abaixo da dobra, foco/teclado e sem JavaScript,
com manifesto de perfil, rota, modo, fuso, viewport e região. Comparar usando cópias
equivalentes, sem credenciais reais nas imagens. Não há capturas da etapa 4 neste
estudo: as da etapa 3 são histórico, não aceite de administração Go.

Go tests, cliente, build e validação multiusuário precisam passar antes de ativar
cada família no modo Go. Atualizar dispatcher, ponte React e manifesto de rotas
juntos, incluindo actions/métodos. A API, ListenBrainz, imagens e assets preservam
suas rotas. React continua disponível como fallback, com build/dependências; sua
remoção e ganho de desempenho exigem as etapas 5/6 e medição separada.

## Orientação de execução atualizada

O usuário determinou que conferência de layout e comparações visuais da etapa 4
ocorram **somente ao final da issue 015**. A 010 entrega infraestrutura e regressões
funcionais; as próximas issues executam testes de dados, permissões e escrita sem
antecipar screenshots ou inspeção visual. As descrições do estudo acima preservam
os problemas reproduzidos antes das correções; o relatório da 010 registra o depois.
