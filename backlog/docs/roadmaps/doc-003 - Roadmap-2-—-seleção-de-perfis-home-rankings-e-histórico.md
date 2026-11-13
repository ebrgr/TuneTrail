---
id: doc-003
title: Roadmap 2 — seleção de perfis, home, rankings e histórico
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_02_PAGINAS.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Roadmap 2 — seleção de perfis, home, rankings e histórico

Análise de 4 de novembro de 2026, sobre os arquivos locais, com HEAD de referência
`72e4be2` e a prévia Go do passo 1 ainda no diretório de trabalho.
**Status atual: implementado e validado em 04/11/2026. Issues 003, 004 e 005
fechadas e resolvidas.** [Entrega, testes e screenshots](../../evidencias/roadmap-02/README.md).
`KOITO_WEB_UI=go` habilita a migração; React continua padrão.

O texto abaixo preserva a análise inicial e suas propostas, anteriores à
implementação. Afirmações de trabalho pendente referem-se a esse levantamento,
não ao estado atual. O relatório acima registra as decisões finais.

O banco continua sendo **SQLite com esquema consolidado v3**. Não há necessidade
identificada de nova migração para renderizar estas páginas. Esta análise não
iniciou servidores, abriu bancos, alterou código funcional ou executou testes.
Os pontos abaixo foram encontrados por leitura do código; possíveis defeitos
ainda precisam de reprodução focada antes de uma correção.

## Conclusão e escopo

O backend já oferece as consultas por perfil necessárias. O passo 2 deve ligar
essas consultas à apresentação Go, substituir os dados fictícios nas páginas
reais e preservar a navegação durante a convivência com React.

| Página | URL final a preservar | Responsabilidade |
| --- | --- | --- |
| Seleção | `/` | Listar perfis públicos, inclusive zero ou um perfil, sem escolher automaticamente |
| Home | `/{slug}` | Estatísticas gerais, atividade, destaques por período e últimas escutas |
| Artistas mais ouvidos | `/{slug}/chart/top/artists` | Ranking e paginação |
| Álbuns mais ouvidos | `/{slug}/chart/top/albums` | Ranking e filtro opcional de artista |
| Faixas mais ouvidas | `/{slug}/chart/top/tracks` | Ranking e filtros opcionais de álbum/artista |
| Histórico | `/{slug}/listens` | Escutas por período, filtros e paginação |

Artistas, álbuns, faixas individuais e retrospectiva permanecem em React no passo
3; administração completa, importação, chaves, edição e mesclagem ficam no passo
4. Os links para essas áreas precisam continuar funcionais durante a transição.
Não retirar React, seu build ou suas dependências neste passo.

## Base existente e lacunas da prévia

`internal/webui/preview.go` reúne fixtures, modelos de apresentação, renderização,
assets e respostas falsas para comparar a SPA. `NewPreview` aceita somente os
perfis fictícios; `makePage` fabrica contagens, músicas, datas e URLs. Seus
parâmetros `state` e `theme` são controles de demonstração.

Os templates e o CSS são reaproveitáveis, mas `NewPreview` e `NewReference` não
devem ser registrados como handlers de produção. A referência recusa escrita e
simula JSON; não é uma camada de acesso ao banco. Separar fixtures e composição
de demonstração dos handlers reais antes de integrar o servidor.

Os modelos atuais também simplificam artistas para uma string. A apresentação
real deve transportar IDs, lista de artistas, `is_various_artists`, imagens
responsivas e `rank` do backend. Repetir as dez faixas fictícias para formar o
histórico, numerar posições pelo índice ou gerar tempos artificiais não serve
para a migração funcional.

## Consultas e regras a reutilizar

| Informação | Código atual / contrato Go |
| --- | --- |
| Perfis e resolução por slug | `db.ProfileStore.ListListeningProfiles`, `GetListeningProfile`; implementação em `internal/db/sqlite/profile_store.go` |
| Histórico | `db.ListenStore.GetListensPaginated(ctx, db.GetItemsOpts)` |
| Rankings | `GetTopArtistsPaginated`, `GetTopAlbumsPaginated`, `GetTopTracksPaginated` nas interfaces de `internal/db/db.go` |
| Estatísticas | Composição em `engine/handlers/stats.go`: contagens, tempo, dias ativos, sequências e razões com proteção contra divisão por zero |
| Atividade | `GetListenActivity`, `GetListenStreak` e preenchimento de dias em `engine/handlers/get_listen_activity.go` |
| Primeira escuta | `GetFirstListenUnix`; atualmente o SQL sem registros retorna erro envolvendo `sql.ErrNoRows` |
| Tocando agora | `engine/handlers/now_playing.go`: memória por `now-playing:<profileID>` e `GetTrack` com `AllowUnplayed` |
| Períodos, datas e parâmetros | `engine/handlers/handlers.go`, `internal/db/timeframe.go`, `period.go` |
| Imagens | `models.ImageList`, `catalog.BuildImageList` e rota existente `/image/{image_id}/{filename}` |

Os handlers HTML devem chamar serviços/consultas Go diretamente. Não consultar
a própria API por HTTP nem executar handlers JSON com `httptest.ResponseRecorder`.
Extrair somente a lógica compartilhada necessária: composição de estatísticas,
normalização de parâmetros/fuso, atividade e tocando agora. Manter os handlers
HTTP JSON finos, preservando seus contratos existentes.

### Home: o seletor de período não controla todos os blocos

Em `client/app/routes/Home.tsx` e nos componentes atuais:

- Estatísticas consultam sempre `all_time`.
- Atividade pede 36 × 7 dias, independentemente do período dos rankings.
- Top artists e Top albums pedem cinco itens; Top tracks pede dez.
- Últimas escutas pedem 28 itens em `all_time`, com tocando agora separado.
- O período padrão da home é `week`; a preferência local pode substituí-lo quando
  a URL não traz `period`. Uma URL explícita tem prioridade.

Preservar essa separação. Selecionar Month não deve filtrar silenciosamente o
histórico recente ou as estatísticas gerais. A grade deve usar as escutas reais,
incluir dias sem atividade e respeitar o fuso efetivo. O começo da semana da grade
depende hoje do locale do navegador; isso é distinto da semana ISO de um filtro.

O código de `LastPlayed.tsx` não configura polling periódico. Há consultas React
Query, inclusive com seus comportamentos de recarga; não assumir atualização
contínua já existente. Primeira entrega: dados atuais em cada navegação/recarga.
Se for acrescentado polling, ele deve ser pequeno, por perfil, cancelável, sem
cache compartilhado e documentado como mudança de comportamento.

## Identidade, erros e isolamento

Resolver `slug → ListeningProfile → ID` uma vez por requisição HTML. Antes de
qualquer consulta pessoal, aplicar `db.WithProfile(ctx, profile.ID)`. Slug inválido,
renomeado ou excluído retorna 404; falha interna de consulta retorna erro 500,
sem converter todos os erros em “perfil inexistente”.

Isso é obrigatório porque `internal/db/sqlite/scoped_reads.go` retorna a consulta
global quando o contexto tem ID zero. A resolução do perfil é uma barreira de
correção, não apenas um recurso do cabeçalho. Não usar `WithoutProfile`, ID fixo
ou cookie de “perfil atual” nas páginas públicas. Duas abas devem ser independentes.

`middleware.ProfileScope` já faz a resolução para a API, mas seus erros são JSON.
Extrair/reutilizar a resolução sem acoplar a resposta HTML a esse formato. Perfis
públicos não exigem sessão; chave de player nunca concede administração. O modelo
`ListeningProfile` não contém tokens e deve continuar assim nos templates.

Proposta de erros: parâmetros inválidos → 400 com mensagem legível; perfil ausente
→ 404; falha de infraestrutura → 500 sem detalhes SQL. Perfil existente sem
escutas → 200 com estado vazio. Falhar uma estatística não pode produzir um zero
falso: diferenciar conteúdo ausente de bloco indisponível e preservar os blocos
que puderem ser carregados com segurança.

## Parâmetros e paginação

Definir um modelo normalizado único para a página e seus links:

- `page`: base **1**, incluindo ausência do parâmetro; anterior desabilitado em 1.
- `limit`: padrão 100 nas listagens, valores positivos até 500; limites da home
  são explícitos. Validar overflow e evitar alocações com valores negativos.
- `period`: `day`, `week`, `month`, `year`, `all_time`; rejeitar valores inválidos
  na interface HTML, sem mudar silenciosamente a tolerância da API existente.
- `year`, `month`, `week`, `from`, `to`: representar um modo temporal coerente;
  não gerar combinações contraditórias nos formulários. Usar o mesmo intervalo
  efetivo para consulta e rótulo da tela.
- IDs de artista/álbum/faixa: inteiros positivos dentro do tipo de ID do backend.
  Não transformar um filtro não suportado numa indicação visual de filtro aplicado.

Filtros hoje reconhecidos: artistas não usam IDs de mídia; álbuns usam artista;
faixas dão precedência a álbum sobre artista; histórico dá precedência a faixa,
depois álbum, depois artista. Preservar esse contrato inicialmente ou rejeitar
combinações ambíguas no HTML com mensagem explícita. Não prometer interseção de
filtros quando as consultas usam precedência.

Links anterior/próximo mantêm perfil, filtros, período e limite. Trocar período ou
filtro volta à página 1. Trocar perfil mantém a tela geral e o período, mas remove
IDs de mídia e `page`, seguindo `client/api/profile-paths.ts`. Links de cards e
“See more” também precisam conter o slug.

Renderizar o `rank` calculado por `RANK()`: empates podem produzir `1, 1, 3`.
Os rankings já têm ordenação secundária por ID. Para histórico, verificar um
desempate explícito por faixa quando escutas têm o mesmo segundo. Offset continua
sendo suficiente para esta entrega, mas inserções concorrentes podem deslocar
páginas; não apresentar uma garantia de snapshot que o código não oferece.

## Datas e fusos

Preservar a precedência atual: `KOITO_FORCE_TZ` → query `tz` válida → cookie `tz`
válido → fallback do servidor. A API de atividade trata o nome `Local` como UTC;
unificar/documentar essa diferença ao extrair o contexto temporal. Não copiar o
offset fixo de São Paulo usado na prévia para o código real.

A SPA grava o cookie no início, antes das consultas. Com HTML pronto, a primeira
requisição pode chegar sem esse cookie. Proposta: script pequeno grava o fuso e
faz no máximo uma recarga inicial quando necessário; sem JS, renderizar com o
fallback do servidor. Um fuso forçado não deve ser substituído pelo navegador.

Usar `TimeframeToTimeRange` como referência para limites de consulta e rótulos,
incluindo precedência de intervalo explícito, Unix, calendário e período. Hoje
os períodos móveis usam `time.Now()`/`AddDate`, enquanto datas de calendário usam
o fuso fornecido. Não trocar um “último dia” móvel por “hoje desde meia-noite”
apenas para combinar com o texto da SPA. Eventual mudança dessas regras é separada.

Usar `<time datetime="...">` para instantes e texto relativo coerente. Para a
primeira escuta, tratar ausência explicitamente, sem ano 1970 ou erro geral da
página; não esconder falhas reais de banco como se fossem perfil vazio.

## Pontos encontrados que não devem ser copiados sem revisão

| Ponto observado por leitura | Consequência / ação na migração |
| --- | --- |
| `ChartLayout.updateParams(params)` sombreia o `params` retornado por `useParams`; usa `params.slug` do objeto de alterações | Pode construir URL sem perfil; usar slug separado e obrigatório, com teste de mudança de período/filtro |
| Loaders usam `page=0` como ausência; API normaliza para 1; botão inferior de Prev no histórico desabilita apenas em zero | Não transportar a ambiguidade; padronizar base 1 |
| Sem query de período, ChartLayout destaca `day`, enquanto a API assume `all_time` | Proposta: listagens sem filtro usam e mostram `all_time`; links/URLs explícitos conservam o filtro |
| Rótulos do ChartLayout calculam semana a partir de 1º de janeiro; backend usa ISO, a partir de 4 de janeiro | Usar limites compartilhados. Exemplo: semana ISO 1 de 2026 começa em 29/12/2025 |
| Rótulos de day/week da SPA não representam exatamente as janelas móveis do backend | Exibir o intervalo realmente consultado |
| Ranking obtém total por janela SQL somente nas linhas retornadas | Uma página além do fim pode informar total zero; mostrar “página sem itens”, sem concluir que o perfil não tem histórico |
| Histórico ordena apenas por timestamp e ignora erros dos `Scan` das consultas de contagem | Validar empate/paginação e propagação de erros; ajustes pontuais, se necessários, sem migration |
| Estado de exclusão no histórico usa apenas `time`; tabela renderizada usa `data.items`, não o estado atualizado | Identificar escuta por perfil + faixa + segundo e recarregar dados após sucesso |
| `TestListenActivityOptsToTimes` está com as verificações comentadas | Acrescentar testes efetivos de datas; não considerar o nome desse teste como cobertura |

Não foram reproduzidas falhas no navegador ou no banco nesta análise. A issue de
contratos abaixo organiza a confirmação dos pontos antes de correções funcionais.

## Convivência com React e administração

`engine/routes.go` termina no fallback da SPA para `index.html`. Registrar as seis
rotas HTML específicas antes dele e manter `/apis`, `/image`, assets, `/admin`,
páginas de mídia e rewind nas implementações atuais. Resolver caminhos legados
sem slug no servidor: um perfil → destino explícito; vários/zero → seleção com
`next`. Permitir somente destinos internos conhecidos, como já faz `Profiles.tsx`.

Um detalhe indispensável: `Link`/`useNavigate` da SPA podem renderizar Home ou
rankings React sem consultar o servidor. Apenas adicionar handlers Go não garante
a migração quando o visitante vem de uma página de artista ainda em React.

Proposta de ativação: opção de implantação `KOITO_WEB_UI=react|go`, inicialmente
com padrão `react` — nome e valores são uma proposta, ainda não implementados.
Em `go`, somente estas seis páginas mudam de renderizador. Compartilhar o modo
com a SPA e implementar uma ponte para navegação completa nas rotas já migradas,
cobrindo também voltar/avançar e navegação programática. Não alterar o modo por
um cookie global de perfil. Testar que a ponte não entra em loop.

Settings deve continuar oferecendo Aparência e um acesso real à administração
React em `/admin` até o passo 4. Busca deve manter acesso ao fluxo atual ou ser
implementada como uma adaptação pequena do endpoint público existente; um diálogo
de demonstração não é aceitável como substituto no modo funcional.

Para exclusão de escutas, reutilizar o endpoint administrativo por perfil
existente, com sessão e proteção de origem, sem adicionar mutações por GET.
Integrar o controle existente com JS pequeno ou manter acesso explícito à tela
React equivalente durante a transição. A escolha deve ser fechada antes de ativar
o modo Go; não perder silenciosamente uma ação existente de administrador.

Essas integrações não significam migrar toda a administração agora. Caso o modo
misto ainda não preserve esses caminhos, continuar com ativação experimental e
não mudar o padrão da implantação.

## Organização proposta

```text
internal/webui/
  renderer.go, models.go, urls.go       # apresentação compartilhada
  handlers.go                         # HTML real, recebendo serviços
  preview.go, fixtures.go, reference.go # demonstração isolada
  templates/                          # seleção, home, ranking, histórico, erros
  static/                             # estilos e JS local
internal/listening/                   # composição de leituras sem HTTP
internal/webquery/                    # parâmetros/fuso compartilháveis
engine/routes.go                     # integração pequena com o servidor atual
backlog/evidencias/roadmap-02/       # evidências após implementação
```

Os nomes são uma proposta de organização. Preferir interfaces pequenas de leitura
a acoplar o renderizador à implementação SQLite ou a copiar SQL. Processar as
consultas inicialmente em sequência: o SQLite está configurado com uma conexão,
e paralelizar toda a home não garante ganho. Propagar cancelamento da requisição.
Não criar um cache global de página. Para HTML com estado administrativo usar
`no-store`; para leituras públicas, começar sem cache de resposta entre perfis.

## Sequência e issues locais

1. [003 — páginas com dados reais](<../../tasks/issue-003 - 003-—-Migrar-seleção-home-rankings-e-histórico-para-HTML-Go.md>): separar
   apresentação/fixtures, resolver perfil e integrar as seis páginas com leituras.
2. [004 — contratos de navegação e datas](<../../tasks/issue-004 - 004-—-Confirmar-contratos-de-paginação-filtros-e-datas.md>): confirmar
   pontos encontrados, normalizar paginação/filtros e testar limites temporais.
3. [005 — convivência Go/React](<../../tasks/issue-005 - 005-—-Preservar-navegação-e-ações-na-convivência-Go-React.md>): roteamento,
   ponte, links legados, Settings/busca e ações existentes.
4. Validar o conjunto com banco descartável e registrar screenshots/limites antes
   de ativar a opção Go numa instalação em uso.

As issues são Markdown locais, conforme orientação do usuário; não devem ser
publicadas neste estágio. Podem ser trabalhadas em fatias, mas 003 só fica pronta
para ativação quando 004 e 005 estiverem atendidas.

## Aceite e evidências a produzir

- Banco descartável com dois perfis e conteúdos distintos, um vazio, nomes longos,
  álbuns coletivos, empates e escutas da mesma faixa/segundo em perfis diferentes.
- Testes HTTP de acesso direto, recarga, slug renomeado/excluído, 404 real versus
  erro de banco, HTML escapado e nenhum segredo em páginas públicas.
- Comparar valores HTML e API para o mesmo perfil/filtro/fuso. Reutilizar os testes
  existentes de isolamento, sem considerar a referência JSON fictícia evidência
  da integração real.
- Paginação com mais de uma página, fronteiras com mesmo segundo, `page=0`,
  negativos/overflow, limites, página além do fim e alteração de período/filtro.
- Datas: dia móvel versus calendário, virada do ano ISO, meses de tamanhos
  diferentes, mudança de fuso e horário de verão em região que o utiliza.
- Navegação Go → mídia React → home Go; seletor de perfil em ambas as tecnologias;
  duas abas, voltar/avançar, links antigos e recusa de `next` externo.
- Visitante e chave de player continuam sem poderes administrativos; sessão
  válida mantém acesso às ações existentes. Requisições de origem externa falham.
- Capturas React/Go sobre cópias equivalentes do mesmo SQLite v3, com tema,
  viewport e fuso iguais. A comparação com fixtures do passo 1 continua útil para
  CSS, mas não comprova os números das páginas reais.
- Registrar em `backlog/evidencias/roadmap-02/` os pares de seleção, home,
  rankings e histórico; incluir página 2, filtros, perfil vazio, desktop e celular.

Esta análise entrega o mapa de integração e as issues propostas. Não há novas
screenshots ou resultados de testes funcionais do passo 2 nesta etapa.
