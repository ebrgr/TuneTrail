---
id: doc-019
title: Issue 038 — contrato e plano da configuração administrativa
type: specification
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - design
---
> Importado de [docs-dev/CONTRATO_CONFIGURACAO_038.md](../../../docs-dev/CONTRATO_CONFIGURACAO_038.md).

# Issue 038 — contrato e plano da configuração administrativa

Data: 09/11/2026. Planejamento retomado por solicitação do usuário.
[Issue 038](../../../docs-dev/issues/038-contrato-configuracao.md) ·
[Inventário completo](../../../docs-dev/ROADMAP_02_CONFIGURACAO_TOML.md) · [Roadmap](../../../docs-dev/ROADMAP.md).

Este documento define a primeira entrega das issues 039–042. A leitura TOML da
039–040 foram aprovadas e fechadas; a tela da 041 foi implementada, aprovada e fechada. A 038 permanece
aberta para acompanhamento e aceite do contrato.

## 1. Escopo confirmado

O usuário priorizou sete opções globais, pediu a avaliação de fuso e relay e
acrescentou Subsonic. Também determinou que **nenhum desses valores seja tratado
como segredo**. Todos os campos desta entrega serão visíveis e editáveis pelo
administrador autenticado: sem máscaras, botão de revelar ou protocolo especial
de manter/substituir/limpar. A senha de login da conta administrativa não integra
essa lista e continua no fluxo de autenticação existente.

Primeira entrega: dez campos globais em `conf.toml`. A gestão de relay já existente
continua no SQLite, por perfil de escuta. Não criar novas contas, privacidade de
perfis, sincronização de escutas Subsonic ou edição de todas as variáveis antigas.
As demais opções continuam externas, conforme o inventário de compatibilidade.

## 2. Campos globais

A coluna TOML define as opções implementadas na primeira entrega.

| Variável atual | Chave TOML | Padrão | Efeito e validação do TOML/UI |
| --- | --- | --- | --- |
| `KOITO_LOG_LEVEL` | `logging.level` | `info` | `debug`, `info`, `warn`, `error`, `fatal`; rejeitar outros valores. Logger é inicializado uma vez. |
| `KOITO_MUSICBRAINZ_URL` | `musicbrainz.url` | `https://musicbrainz.org` | Base HTTP(S), host obrigatório, sem userinfo, query ou fragmento. Remover barra final preservando basepath. Permitir servidor próprio/rede local. |
| `KOITO_MUSICBRAINZ_RATE_LIMIT` | `musicbrainz.rate_limit` | `1` | Inteiro positivo compatível com o consumidor Go. Endpoint oficial exige 1; validar junto com a URL normalizada. |
| `KOITO_DISABLE_DEEZER` | `images.disable_deezer` | `false` | Booleano; desativa a fonte de imagens Deezer, não uma integração de escutas. |
| `KOITO_DISABLE_COVER_ART_ARCHIVE` | `images.disable_cover_art_archive` | `false` | Booleano; desativa a fonte de capas CAA. |
| `KOITO_DISABLE_MUSICBRAINZ` | `musicbrainz.disabled` | `false` | Booleano; desativa o cliente de metadados MusicBrainz no engine, não apenas imagens. |
| `KOITO_LASTFM_API_KEY` | `images.lastfm_api_key` | vazio | Texto visível/editável. Vazio desativa o provedor de imagens Last.fm; não configura scrobbling para Last.fm. |
| `KOITO_SUBSONIC_URL` | `subsonic.url` | vazio | Base HTTP(S) do servidor Subsonic/Navidrome; mesma regra estrutural de URL, preservando basepath e aceitando rede local. Deve ser preenchida junto com params. |
| `KOITO_SUBSONIC_PARAMS` | `subsonic.params` | vazio | Texto visível/editável com parâmetros para fetch. Interpretar query URL-encoded: `u` e (`t`+`s` ou `p`), sem valores vazios, duplicatas ou mistura dos dois modos. Não permitir substituir parâmetros fixos do cliente (`f`, `v`, `c`, `query`, `id` e contagens). |
| `KOITO_FORCE_TZ` | `time.force_tz` | vazio | Fuso IANA aceito por `time.LoadLocation`, ou vazio para automático. Não alterar timestamps persistidos. |

URLs são para consulta aos provedores. Salvar configuração não faz acesso remoto.
O campo params Subsonic não será rotulado nem tratado como segredo na UI.
URL e params ambos vazios desativam Subsonic; par incompleto é erro no TOML/UI.
A validação nova é deliberadamente mais estrita que o parser legado de ambiente.

## 3. Fuso: usar FORCE_TZ na interface, manter TZ externo

`TZ` configura o fuso local do processo/container e pode influenciar logs e outros
consumidores de `time.Local`. Não é uma opção do carregador `cfg` e não deve ser
alterada por `os.Setenv` depois de o serviço iniciar.

`KOITO_FORCE_TZ` já é consumido pela aplicação e prevalece sobre preferências da
requisição. Por isso, a tela oferecerá **Fuso da aplicação**, com escolha IANA ou
modo automático, persistidos em `time.force_tz`.

- Forçado: usar o fuso global em consultas/apresentações que já consomem esse helper;
  avisar que a preferência do navegador deixa de prevalecer nesses caminhos.
- Automático (`force_tz = ""`): manter query `tz` válida > cookie `tz` válido >
  fuso do processo. Query inválida pode cair no cookie válido, como hoje.
- Origem da opção forçada: ambiente > `_FILE` > TOML > padrão vazio.
- `TZ` permanece na implantação, inclusive `KOITO_TIMEZONE` no Compose. A UI pode
  informar o fuso local observado, mas não altera Compose, ambiente ou relógio do SO.
- Isto não cria fuso persistido por usuário/perfil nem modifica a semântica dos
  calendários excepcionais ou médias registrada na issue 018.

Exemplo: processo em UTC e `force_tz = "America/Sao_Paulo"` usam São Paulo nos
consumidores do fuso forçado; isso não muda automaticamente o fuso dos logs.

## 4. Relay: já existe no banco por perfil

Verificado em `003_listening_profiles.sql`, `profile_store.go`, `admin/relay.go`,
handlers de relay, template administrativo e ingestão ListenBrainz:

| Entrada legada | Situação atual | Plano |
| --- | --- | --- |
| `KOITO_ENABLE_LBZ_RELAY` | Habilita leitura das variáveis legadas na inicialização. Não há coluna `enabled` por perfil. | Manter compatibilidade; o envio usa a associação do perfil no banco. |
| `KOITO_LBZ_RELAY_URL` | Destino persistido em `profile_relays.url`, vinculado por `profile_id`. | Continuar edição administrativa do perfil; não duplicar no TOML global. |
| `KOITO_LBZ_RELAY_TOKEN` | Valor persistido em `profile_relays.token`, vinculado por `profile_id`. | Continuar no banco, visível na administração conforme pedido; sem novo mecanismo de ocultação. |

A tabela `profile_relays(profile_id, url, token)` pertence ao esquema consolidado
v3. Uma linha com URL/token habilita o relay; remover a associação o desativa.
Não existe toggle que preserve dados ao desativar: isso exigiria desenho separado.
Os valores são texto no banco; a UI atual mostra o token e a API admin
o retorna. Esta análise não leu o banco operacional nem valores reais.

`legacy_relay_binding` registra uma decisão única de associação das variáveis
antigas. O bootstrap só escolhe automaticamente quando há exatamente um perfil e
mapeamento legado de administrador. Com vários perfis, exige seleção explícita;
remoção/revogação não deve reativar ou reassociar o destino no próximo reinício.
`ON CONFLICT DO NOTHING` preserva configuração por perfil já existente.

“Por usuário” significa aqui **perfil de escuta**, não conta proprietária com
permissão de autogestão. Quem altera é o administrador global autenticado. A
configuração do relay é aplicada às novas leituras sem depender do restart das
opções globais; envio já iniciado pode concluir. O relay continua best-effort.

## 5. Arquivo, precedência e compatibilidade

- Caminho: `<KOITO_CONFIG_DIR>/conf.toml`, ao lado de `tunetrail.db`. Descobrir o
  diretório externamente antes de ler o TOML, inclusive pelo mecanismo `_FILE`.
  Não colocar `config_dir` dentro do próprio arquivo nem mover banco pela UI.
- Configuração global em TOML; relay em SQLite. Sem migração ou alteração das
  migrations 001–003. Conta master e senha continuam na gestão de conta existente.
- Ordem por campo: variável direta **não vazia** > `_FILE` configurado e legível >
  campo TOML presente > padrão. Ambiente direto vazio mantém o fallback atual.
- `_FILE` é um mecanismo de leitura de arquivo. Seu conteúdo recebe trim; se o
  arquivo configurado estiver vazio, esse vazio é a fonte selecionada, sem cair
  silenciosamente no TOML. Erro de leitura não usa fallback silencioso.
- Campo TOML ausente herda padrão/fontes superiores; `false` não é ausência.
  Texto vazio limpa os campos cujo domínio admite vazio (Last.fm, fuso e par
  Subsonic). URL MusicBrainz e log level não aceitam texto vazio.
- Sem TOML: preservar defaults e interpretação do ambiente atual (log desconhecido
  vira info, rate não numérico cai no padrão, booleano só reconhece true). Não
  copiar automaticamente variáveis para arquivo nem criar TOML no primeiro boot.
- TOML presente: schema 1 tipado e apenas as dez chaves acima. Sintaxe inválida,
  versão não suportada ou chave desconhecida causam erro claro, sem reescrita.
  A validação estrita dos campos explícitos do arquivo também ocorre se houver
  override no ambiente. Documentar mudanças de compatibilidade ao ativar TOML.
- Validar duas composições antes de salvar: TOML com defaults (caso overrides sejam
  removidos) e configuração efetiva com ambiente/arquivos externos. Com TOML
  ativo, o par Subsonic efetivo e a combinação MusicBrainz também devem ser válidos.
- A tela mostra valor salvo, valor ativo e fonte/override. Campos controlados pelo ambiente ficam desabilitados na interface, com o nome
  da variável direta ou `_FILE`; a gravação preserva o valor TOML existente.

Exemplo de arquivo aceito pelo leitor implementado na 039:

```toml
schema_version = 1

[logging]
level = "info"

[musicbrainz]
url = "https://musicbrainz.org"
rate_limit = 1
disabled = false

[images]
disable_deezer = false
disable_cover_art_archive = false
lastfm_api_key = ""

[subsonic]
url = ""
params = ""

[time]
force_tz = ""
```

Para Subsonic, por exemplo, URL `http://navidrome:4533` e params fictícios
`u=preview&t=exemplo&s=exemplo` são informações para fetch, não uma integração de
histórico. Os exemplos não representam credenciais de uma instalação real.

## 6. Persistência e aplicação

Todos os dez campos globais serão aplicados **após reinício externo** na primeira
entrega: logger, filas e clientes já são construídos na inicialização. Salvar não
reinicia o serviço nem promete recarga parcial. O estado “pendente de reinício”
compara a composição efetiva salva com a fotografia ativa do último startup.

A escrita usa um serviço próprio (implementado na 040): validação, revisão do documento, lock entre
requisições, temporário no mesmo diretório, escrita/sync e substituição atômica;
sincronizar o diretório quando suportado. Definir permissões compatíveis com o
usuário do serviço e preservar atributos aplicáveis do arquivo existente.

- Conflito de revisão retorna 409 sem descartar alterações do outro editor.
- Antes do rename, falha mantém arquivo/revisão anteriores e limpa temporário.
- Após rename, se o sync do diretório falhar, reler o arquivo e informar estado
  de durabilidade incerto; não declarar que o arquivo anterior foi preservado.
- Não descartar comentários silenciosamente. A biblioteca/estratégia escolhida na
  039/040 deve preservar o texto fora das chaves editadas; se isso não for possível,
  recusar a escrita sem alterar o arquivo. Chaves desconhecidas bloqueiam edição.
- Comparar a revisão dentro do lock e imediatamente antes da publicação. Editor
  externo deve cooperar com o lock ou editar com o serviço parado: não prometer
  compare-and-swap universal contra qualquer processo externo.
- O leitor de configuração não exige escrita. O servidor atual ainda exige diretório
  de dados gravável para SQLite; a UI futura deve informar gravação indisponível.

## 7. Administração e teste Subsonic

Proposta de rotas HTML próprias: `GET /admin/settings`,
`POST /admin/settings/save` e `POST /admin/settings/subsonic/test`.
Integrar ao dispatcher Go e à navegação administrativa na 041.

Usar login/senha e sessão administrativa já existentes, autorização no servidor,
origem/CSRF, `no-store`, limite de corpo e proteção atual contra tentativas.
Não exigir confirmação adicional de senha por salvamento. Visitante e chave de
player/ingestão não podem ler ou escrever. Os valores solicitados ficam visíveis
na área admin; não ampliar o endpoint público de configuração nem despejar a
configuração em logs ou respostas públicas. Não criar parâmetro para escolher
caminhos `_FILE` pela interface.

Subsonic exige um teste explícito do rascunho: POST autenticado, sem salvar ou
alterar clientes ativos. Avisar que o teste usa o formulário e que overrides do
ambiente podem prevalecer na configuração efetiva. Utilizar helper compartilhado
com o ping do startup: timeout de 10 segundos, contexto, limite de resposta de
8 MiB, sem redirecionamentos, HTTP 2xx, JSON válido e
`subsonic-response.status == "ok"`. Retornar diagnóstico de conexão/status sem
copiar o corpo completo ou URL composta para logs. Os campos continuam visíveis
no formulário, conforme a orientação do usuário.

Hoje o startup usa `http.Get` sem timeout explícito e pode encerrar em falha.
A 039 limitou esse ping antes da edição na 041, preservando o erro
operacional de inicialização e orientando correção offline. Alterar arquivo não
faz requisição automática; indisponibilidade remota não bloqueia salvar um
rascunho estruturalmente válido. O teste remoto também não sincroniza escutas.

## 8. Sequência das issues

| Issue | Entrega planejada |
| --- | --- |
| 038 | Este contrato: inventário real, dez campos, fuso, relay, compatibilidade e critérios. Sem mudança runtime. |
| 039 | Leitor tipado/origem dos valores, validações, mesma descoberta no reset-password, carregamento com erro estável e helper limitado do ping Subsonic. Preservar execução sem TOML. |
| 040 | Persistência dos dez campos implementada e aprovada. [Evidências](../../../docs-dev/comparisons/configuracao/ISSUE_040.md). |
| 041 | Tela Go autenticada implementada, aprovada e fechada. [Evidências e capturas](../../../docs-dev/comparisons/configuracao/ISSUE_041.md). |
| 042 | Matriz, reinício, recuperação, Docker/Compose, Delve, capturas e docs validados; aguardando aceite. [Evidências e limites](../../../docs-dev/comparisons/configuracao/ISSUE_042.md). |

O uso atual de `sync.Once` em `cfg.Load` guarda a inicialização, mas não persiste
seu erro em variável global. A 039 passou a preservar esse erro em chamadas
posteriores; isso não é hot reload.
Subsonic, fuso e os dez campos novos não passam a ser preferências por perfil.

## 9. Matriz mínima de validação das próximas entregas

| Grupo | Casos / resultado esperado |
| --- | --- |
| Dez campos | Defaults, TOML presente/ausente, env, `_FILE`, vazio/false explícito e origem; arquivo `_FILE` ausente ou vazio; nenhum banco real. |
| Estrutura | TOML inválido, tipo errado, schema/chaves desconhecidos, arquivo grande, permissões e comentários; erro não sobrescreve o arquivo. Definir limites de bytes na implementação. |
| MusicBrainz | URL oficial e variante normalizada com rate 1; rate 0/negativo/fracionário, URL inválida e oficial+rate diferente de 1 rejeitados; endpoint próprio; arquivo inválido não escondido por override. |
| Imagens / logs | Cada booleano e Last.fm vazio/preenchido; nível válido/inválido; valores ativos só mudam após restart. |
| Subsonic | Par vazio/válido/incompleto, basepath, query malformada, duplicatas e parâmetros reservados; ping ok/timeout/redirect/JSON inválido/status de erro com servidor local fictício. |
| Fuso | ForceTZ versus query/cookie/TZ, IANA inválido, modo automático, diferença entre fuso de logs e aplicação; timestamps SQLite inalterados. |
| Relay existente | Dois perfis com destinos distintos; remover e reiniciar não reativa; env legado não sobrescreve associação existente; sem duplicação no TOML. |
| Administração | Login válido, visitante/chave negados, CSRF/origem, leitura pública sem os campos administrativos; valores visíveis ao administrador; nenhuma nova confirmação de senha. |
| Gravação | Duas abas, revisão obsoleta, edição externa cooperativa, falhas pré/pósrename, readonly e reinício; distinguir salvo/ativo/override. |
| Recuperação | Instalação sem arquivo, arquivo válido/inválido, correção offline, reset-password no mesmo diretório/banco; banco v3 e relay preservados. |

## 10. Evidências e revisão do planejamento

Conferência estática de cfg/getters, main/readEnvOrFile, engine, logger,
MusicBrainz, cliente Subsonic, webquery/TimezoneWithForce, schema e serviços de
relay, além dos formulários atuais. Não foram lidos `.env` reais, banco operacional
ou valores reais dos campos.

Executados com `CGO_ENABLED=0`, `GOPROXY=off`, cache temporário e `-count=1`:
`TestWebUIConfig`, `TestHTMLContracts`,
`TestTimezoneFallbackKeepsProcessLocalAndUsesValidCookieAfterInvalidQuery` e
`TestProfileKeysCannotMoveAndRelayDoesNotReassign`. Os três pacotes envolvidos
passaram (`internal/cfg`, `internal/webquery`, `internal/db/sqlite`).
São verificações do comportamento existente, não testes de TOML ou tela novos.

Advisor/Sol, esforço baixo, revisou a proposta antes da edição documental:
**aprovado com ressalvas**, incorporadas no contrato: validar as duas composições,
readonly, falhas antes/depois da publicação, concorrência cooperativa, par Subsonic
e teste limitado compartilhado. Confirmou aprovação de campos visíveis conforme
pedido e sessão admin sem reautenticação adicional. Não executou testes nem
alterou arquivos. Em seguida, a implementação da 039 foi revisada e aprovada
após correções. A 040 foi fechada e a 041 aprovada e fechada; a 042 foi implementada e aguarda aceite.
