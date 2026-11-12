# Estudo — configuração em TOML e administração pelo usuário master

Estudo inicial: 06/11/2026; planejamento retomado em 09/11/2026. Este é o item 1 do roadmap atual; não se confunde com a antiga etapa 2 de migração de páginas. A leitura inicial de `conf.toml` foi implementada na 039; edição pela UI foi implementada e aprovada. O banco continua sendo SQLite, esquema consolidado v3; a proposta não exige migração do banco.

## Contrato atual da primeira entrega — 09/11/2026

O usuário retomou a 038 e priorizou dez campos globais: logging, MusicBrainz,
fontes Deezer/CAA/Last.fm, Subsonic URL/params e fuso. Todos os valores citados
serão visíveis/editáveis pelo administrador autenticado, sem tratamento especial
de segredo, conforme orientação explícita. A senha da conta de login é separada.

O [contrato da issue 038](CONTRATO_CONFIGURACAO_038.md) é a referência para schema,
precedência, compatibilidade, persistência, tela e matriz de validação das 039–042.
A interface usará FORCE_TZ; TZ permanece externo. Relay já é armazenado por perfil
no SQLite v3 e não será duplicado no TOML. Subsonic fornece fetch de dados/imagens,
não sincronização de escutas. Leitura da 039, persistência da 040 e interface da 041 implementadas; aceite integrado permanece na 042.

## Situação atual e fontes

`cmd/api/main.go` fornece `os.Getenv` ao engine e `internal/cfg` resolve as variáveis e `_FILE`. Para cada opção efetivamente consultada, primeiro é usado `KOITO_NOME` quando não vazio; caso contrário, `KOITO_NOME_FILE` aponta para um arquivo cujo conteúdo é lido e tem espaços nas extremidades removidos. Um arquivo indicado que não pode ser lido encerra o processo; o valor direto prevalece quando ambos estão preenchidos. A aplicação não carrega `.env` por conta própria: ferramentas de desenvolvimento podem fornecê-lo ao processo. Não foram lidos arquivos `.env` reais nem segredos.

`KOITO_CONFIG_DIR` determina o diretório de `tunetrail.db`, com migração do legado `koito.db` (issue 046); seu padrão é `/etc/koito`. A configuração é carregada no início (`engine/engine.go`, `internal/cfg/cfg.go`), e clientes, servidor e templates também são inicializados com esses valores. Há um setter específico para login gate usado em testes: isso não constitui recarga geral de configuração.

## Inventário das variáveis do aplicativo

Padrões abaixo são os do código, não necessariamente os valores fornecidos pelo Compose ou `.env.example`. A senha de bootstrap da conta é separada dos campos comuns priorizados; não usar valores operacionais reais em exemplos. `_FILE` é um mecanismo adicional para cada opção efetivamente lida; no relay os campos só são lidos quando o relay legado está habilitado.

| Variável | Padrão e finalidade atual | Tratamento proposto |
| --- | --- | --- |
| `KOITO_CONFIG_DIR` | `/etc/koito`; diretório do banco, importação e cache | Bootstrap externo; descobrir antes de ler TOML. Mudá-lo também muda o banco usado: não mover dados automaticamente. |
| `KOITO_WEB_UI` | `go`; outro valor causa erro | Compatibilidade; não oferecer React. |
| `KOITO_BIND_ADDR` | vazio; endereço de escuta | Rede; reinício. Vazio permite bind em interfaces não especificadas. |
| `KOITO_LISTEN_PORT` | `4110` se ausente/inválida | Rede; reinício; validar faixa na nova interface. |
| `KOITO_ENABLE_STRUCTURED_LOGGING` | `false`; logging estruturado | Logging; reinício. |
| `KOITO_LOG_LEVEL` | `info`; `debug`, `warn`, `error`, `fatal` aceitos | Logging; reinício. Valor desconhecido resulta em `info`. |
| `KOITO_MUSICBRAINZ_URL` | `https://musicbrainz.org` | Serviço externo; validar URL. |
| `KOITO_MUSICBRAINZ_RATE_LIMIT` | `1`; inteiro | Serviço externo; a URL padrão exige valor 1. |
| `KOITO_DEFAULT_USERNAME` | `admin`; criação inicial da conta | Apenas bootstrap; não altera usuário existente. Gerenciar conta corrente na tela própria. |
| `KOITO_DEFAULT_PASSWORD` | `changeme`; criação inicial da conta; **segredo** | Apenas bootstrap; não altera senha existente. Não copiar automaticamente para TOML; alteração da conta corrente usa o fluxo existente. |
| `KOITO_DEFAULT_THEME` | vazio na configuração; interface usa `yuu` se vazio/inválido | Aparência; validar temas disponíveis. |
| `KOITO_DISABLE_DEEZER` | `false`; desativa fonte | Serviços de imagem. |
| `KOITO_DISABLE_COVER_ART_ARCHIVE` | `false`; desativa fonte | Serviços de imagem. |
| `KOITO_DISABLE_MUSICBRAINZ` | `false`; desativa cliente de metadados MusicBrainz | Metadados do catálogo; reinício. |
| `KOITO_SUBSONIC_URL` | vazio; endereço do servidor | Fetch Subsonic; campo comum visível, reinício. |
| `KOITO_SUBSONIC_PARAMS` | vazio; parâmetros de consulta/autenticação | Fetch Subsonic; campo comum visível/editável, validado junto da URL. |
| `KOITO_LASTFM_API_KEY` | vazio; chave Last.fm | Provedor de imagens; campo comum visível/editável. |
| `KOITO_SKIP_IMPORT` | `false`; pula importação de inicialização | Importação; reinício. |
| `KOITO_CORS_ALLOWED_ORIGINS` | vazio; origens separadas por vírgula | Segurança HTTP; validar cada origem. |
| `KOITO_DISABLE_RATE_LIMIT` | `false`; desativa limitação | Segurança HTTP; reinício. |
| `KOITO_THROTTLE_IMPORTS_MS` | `0`; intervalo em milissegundos | Importação; validar inteiro não negativo. |
| `KOITO_IMPORT_BEFORE_UNIX` | sem limite; segundos Unix positivos | Importação; validar intervalo. |
| `KOITO_IMPORT_AFTER_UNIX` | sem limite; segundos Unix positivos | Importação; validar intervalo. |
| `KOITO_FETCH_IMAGES_DURING_IMPORT` | `false`; busca imagens durante importação | Importação. |
| `KOITO_ARTIST_SEPARATORS_REGEX` | `\s+·\s+`; padrões separados por `;;` | Importação/catálogo; compilar e validar antes de salvar. |
| `KOITO_LOGIN_GATE` | `false`; opção lida pelo carregador | Fora da primeira entrega; não prometer privacidade por perfil sem validação dos consumidores. |
| `KOITO_FORCE_TZ` | vazio; fuso IANA opcional | Datas e estatísticas; validar existência do fuso. |
| `KOITO_ENABLE_LBZ_RELAY` | `false`; habilita leitura do relay legado | Migração inicial para configuração por perfil no SQLite, não substitui a gestão existente por perfil. |
| `KOITO_LBZ_RELAY_URL` | vazio; URL do relay legado | Bootstrap legado; URL só é lida quando habilitado. |
| `KOITO_LBZ_RELAY_TOKEN` | vazio; token legado | Bootstrap legado; gestão administrativa visível por perfil no banco, sem sobrescrita automática. |

A leitura booleana atual considera verdadeiro apenas `true`, sem distinguir maiúsculas. Valores numéricos inválidos frequentemente retornam ao padrão ou a zero. O código atual só habilita Subsonic se URL e parâmetros estiverem ambos preenchidos; a verificação seguinte de configuração parcial é inalcançável, portanto um valor isolado hoje desabilita a integração silenciosamente. Uma validação mais rigorosa precisa ser planejada como alteração de comportamento, sem impedir inesperadamente a inicialização de instalações existentes.

### Legado e escopo externo

| Nome | Situação comprovada |
| --- | --- |
| `KOITO_DATABASE_URL` | Legado PostgreSQL: se preenchido, a inicialização falha. Não reativar PostgreSQL. |
| `KOITO_SQLITE_ENABLED` | Opção obsoleta; SQLite já é o banco, sem escolha de backend. |
| `KOITO_ALLOWED_HOSTS` | Lida como legado; o engine avisa que está obsoleta, sem torná-la controle de segurança novo. |
| `KOITO_ENABLE_FULL_IMAGE_CACHE` | Constante declarada, mas não consultada pelo carregador atual; não anunciar como funcional. |
| `KOITO_BASE_URL` | Apenas declaração comentada; não é opção ativa. |
| `TZ` | Variável de ambiente do runtime Go para fuso local; é diferente de `KOITO_FORCE_TZ` e não passa pelo carregador Koito. Não prometer que salvar TOML altera o ambiente do processo. |
| `KOITO_IMAGE_TAG`, `KOITO_HOST_BIND`, `KOITO_HOST_PORT`, `KOITO_DATA_DIR`, `KOITO_ADMIN_USERNAME`, `KOITO_ADMIN_PASSWORD`, `KOITO_TIMEZONE` | Interpolação do Compose, não opções lidas diretamente por `cfg`; podem definir imagem, portas, volume e variáveis encaminhadas ao contêiner. Não são editáveis pelo processo como substitutos do Compose. |
| `KOITO_VERSION` e variáveis de teste/build | Usadas na construção ou em testes; não são configurações comuns do servidor em execução. |

Outras variáveis do sistema ou de bibliotecas podem influenciar a execução, por exemplo proxy e certificados; não se deve interpretar esta lista como inventário exaustivo do runtime Go. Confirmar nomes e uso no manifesto de distribuição antes de modificar documentação de implantação.

## Plano atual e sequência de execução

O inventário acima preserva todas as opções externas conhecidas, mas a primeira
entrega será limitada ao [contrato da 038](CONTRATO_CONFIGURACAO_038.md).

1. **038 — contrato:** dez campos, fuso, relay existente e regras; planejamento
   registrado e revisado, issue em andamento.
2. **039 — leitura implementada e aprovada:** TOML tipado, origem, duas composições validadas, mesmo
   diretório no reset-password e ping Subsonic limitado; sem escrita automática.
3. **040 — gravação implementada e aprovada:** revisão, comentários, publicação atômica, conflitos e falhas.
4. **041 — UI implementada e aprovada:** valores comuns visíveis no admin autenticado, estado ativo/salvo,
   override, reinício e teste explícito Subsonic; relay continua na tela do perfil.
5. **042 — implementada, aguardando aceite:** matriz, integração local e Docker/Compose, Delve, capturas e documentação; [evidências e limites](comparisons/configuracao/ISSUE_042.md).

Diretório de dados, credenciais de bootstrap, rede/Compose e opções não priorizadas
continuam externos. Não há realocação de banco ou novo modelo de usuários.
Alterações globais exigirão reinício externo; a tela não faz hot reload.

As propostas iniciais de mascaramento dos campos citados e confirmação adicional
de senha foram substituídas pela orientação atual do usuário: campos comuns e
sessão administrativa existente. A descoberta/precedência, concorrência e demais
critérios detalhados estão no contrato para evitar versões divergentes do plano.

## Revisão do Advisor

O estudo de 06/11 recebeu reprovação com ressalvas para implementação imediata;
tratava de um escopo maior e ainda indefinido. Em 09/11 o Advisor/Sol revisou o
plano delimitado da 038 antes das edições: aprovado com ressalvas incorporadas
sobre duas composições, readonly, revisão/concorrência cooperativa, falhas antes e
depois do rename e teste Subsonic compartilhado. Confirmou campos visíveis e
sessão admin sem reautenticação extra. Não executou os testes dirigidos do agente
principal. Esta revisão é do planejamento, não aceite de implementação futura.
