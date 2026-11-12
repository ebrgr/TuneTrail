---
id: doc-005
title: Roadmap 3 — artistas, álbuns, faixas e retrospectiva
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_03_MIDIA_RETROSPECTIVA.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Roadmap 3 — artistas, álbuns, faixas e retrospectiva

Estudo de 4 de novembro de 2026, baseado nos arquivos locais sobre HEAD `72e4be2`,
incluindo a implementação ainda não commitada do roadmap 2.
**Status atual: núcleos 006–008 e integração 009 implementados; comparação visual e testes
registrados na 009; issues 006–009 encerradas por aceite do usuário em 05/11/2026.**
[Implementação da 006](../../evidencias/roadmap-03/ISSUE_006.md) ·
[Implementação da 007](../../evidencias/roadmap-03/ISSUE_007.md) ·
[Implementação da 008](../../evidencias/roadmap-03/ISSUE_008.md) ·
[Integração e comparações da 009](../../evidencias/roadmap-03/ISSUE_009.md).

O restante deste documento preserva o estudo inicial. Suas referências a trabalho
não implementado descrevem a situação do levantamento, anterior ao marco acima.

A base vigente é **SQLite com esquema consolidado v3**. Não foi identificada
necessidade de migration para esta etapa. O serviço iniciado a pedido do usuário
com `KOITO_WEB_UI=go` não foi reiniciado nem reconfigurado neste estudo.

## Conclusão e limites

Os dados já estão disponíveis no backend. A migração precisa acrescentar modelos,
templates e composição de leitura, substituir o gráfico Recharts por SVG e
estender o roteamento Go/React. A retrospectiva já possui um serviço Go utilizável,
`summary.GenerateSummary`; não é preciso reescrever seu cálculo na apresentação.

O ponto que exige mais cuidado é a administração embutida nas páginas de mídia.
Trocar somente seu HTML removeria o acesso a editar, mesclar, excluir, trocar capa
e adicionar escutas. Recomenda-se uma rota React transitória específica para essas
ações, mantendo as páginas públicas em Go. A implementação completa dos editores
permanece na etapa 4. Não remover React, seu build ou dependências nesta etapa.

Também foram encontrados contratos que precisam de testes/correções antes da
ativação: médias da retrospectiva, erros ignorados nos detalhes de mídia e
validação/limites dos intervalos do gráfico. Não copiar esses problemas para os
novos templates.

| Página pública | URL preservada | Conteúdo principal |
| --- | --- | --- |
| Artista | `/{slug}/artist/{id}` | Cabeçalho, oito faixas, onze escutas, interesse, atividade e álbuns associados |
| Álbum | `/{slug}/album/{id}` | Cabeçalho/artista ou Various Artists, oito faixas, onze escutas, interesse e atividade |
| Faixa | `/{slug}/track/{id}` | Cabeçalho/artistas/álbum, onze escutas, interesse e atividade |
| Retrospectiva | `/{slug}/rewind` | Seletor mensal/anual, três top 5, indicadores e opção de mostrar minutos |

## O que a interface atual efetivamente faz

### Mídia

`client/app/routes/MediaItems/{Artist,Album,Track}.tsx` e `MediaLayout.tsx` montam:

- Capa responsiva, tipo, título, rank geral **do perfil**, reproduções, tempo
  ouvido, primeira escuta e link MusicBrainz quando disponível. Esses números
  são all_time; a página não tem um seletor global de período funcional.
- Top tracks de artista/álbum: oito itens, all_time, com links para o ranking
  completo filtrado. É um ranking por reproduções, não uma lista de faixas na
  ordem do disco. O modelo atual não oferece número/ordem de faixa para esse uso.
- Últimas onze escutas: all_time, com o filtro da entidade e links para o histórico.
  `showNowPlaying` só inclui a faixa em execução quando corresponde ao artista,
  álbum ou faixa atual, além de pertencer ao perfil.
- Interesse: dezesseis divisões temporais por padrão. O gráfico cobre da primeira
  escuta da entidade no perfil até **agora**, não apenas até a última escuta.
- Atividade: 36 semanas alinhadas ao começo da semana do locale; em até 640 px
  esconde as dez semanas mais antigas. Intensidade satura em 20 escutas/dia para
  artista, 10 para álbum e 5 para faixa; a home usa 100. Zero, futuro e erro são
  estados distintos. O atributo React `configurable` é passado, mas não é usado
  pelo componente atual: não há seletor configurável a reproduzir.
- Artista: até 99 álbuns via `GetTopAlbumsPaginated`, filtro por associação em
  `artist_releases`. São álbuns ouvidos pelo perfil e associados ao artista;
  não é uma discografia completa. A contagem do álbum inclui suas escutas no
  perfil, inclusive faixas de outros artistas de uma coletânea. Preservar esse
  significado e o texto “Albums featuring …”. Acrescentar acesso à listagem
  completa quando houver mais de 99, sem truncamento silencioso.
- Álbum: a SPA mostra somente o artista principal, ou Various Artists. A faixa
  mostra todos os artistas e o álbum. O modelo Go deve manter a lista completa
  sem colapsar nomes em uma única string; decidir a exibição com os metadados.

Aliases são usados na edição e o artista altera sua lista em memória, mas não há
um painel público de aliases renderizado nessas páginas. `MediaItemNote` é
importado em alguns arquivos, sem uso: não constitui funcionalidade existente.

### Retrospectiva

`RewindPage.tsx` usa `getRewindParams` em `client/app/utils/utils.ts`:

- Sem parâmetros: janeiro abre o ano anterior; nos demais meses abre o mês anterior
  do ano corrente. O `month=0` de janeiro é uma convenção para anual.
- `year` isolado seleciona anual; `year` + `month` seleciona mensal; mês isolado
  infere o ano mais recente segundo a regra atual do cliente. `week`, `from`, `to`
  são aceitos pelo endpoint genérico de summary, mas não fazem parte do seletor
  desta página. Não apresentar suporte HTML a eles sem definir a experiência.
- O seletor oferece períodos concluídos, limitado pela primeira atividade. URLs
  explícitas podem hoje pedir o período corrente, mesmo sem botão correspondente.
- O card exige pelo menos um artista, um álbum e uma faixa. Ausência de qualquer
  grupo produz “Not enough data”; não é o mesmo que erro de consulta.
- São cinco itens por grupo, ordenados por reproduções. “Show time listened?”
  acrescenta minutos dos itens, sem mudar a ordenação nem os totais do período.
- Indicadores visíveis: minutos, plays, plays/dia, únicos e novos de cada tipo.
  `avg_minutes_listened_per_day` existe na API, mas não é mostrado no card atual.
  “Novos” significa primeira escuta **do perfil** no intervalo, não criação global
  do item ou novidade no catálogo.
- Os títulos dos top items atuais não são links. Torná-los navegáveis pode ser
  uma melhoria pequena, mas deve ser registrada, não descrita como paridade já existente.

## Serviços e consultas a reutilizar

| Necessidade | Fonte local |
| --- | --- |
| Resolver slug e estabelecer escopo | `internal/webui/handlers.go`, `db.WithProfile`, `ProfileStore` |
| Detalhes e estatísticas all_time | `GetArtist`, `GetAlbum`, `GetTrack` em `internal/db/sqlite/` |
| Relações e imagens | `models.Artist/Album/Track`, `models.ImageList`, `catalog.BuildImageList` |
| Rankings filtrados e histórico | `GetTopTracksPaginated`, `GetTopAlbumsPaginated`, `GetListensPaginated` |
| Atividade e streak | `GetListenActivity`, `GetListenStreak`, `internal/listening/activity.go` |
| Interesse | `GetInterest(ctx, db.GetInterestOpts)` em `internal/db/sqlite/interest.go`; tipos em `internal/db/types.go` |
| Tocando agora | `internal/listening/now_playing.go` |
| Retrospectiva | `internal/summary/summary.go`, chamado pela API em `engine/handlers/get_summary.go` |
| Fuso e intervalos | `internal/webquery/`, `internal/db/timeframe.go` |
| Layout, tema e controles existentes | `internal/webui/templates/`, `static/`, `live_models.go` |

Handlers HTML devem chamar serviços Go diretamente, com cancelamento da requisição;
não consultar a própria API por HTTP nem executar handlers JSON para obter dados.
`ReadStore` atual já incorpora as interfaces de leitura necessárias por composição,
embora elas também declarem escritas. Preferir interfaces menores nos novos
compositores; não introduzir dependência direta do renderizador no SQLite.

`summary.GenerateSummary` exige ID positivo, reaplica `db.WithProfile` e monta três
top 5. Para cada um dos até quinze itens, reconsulta tempo e contagem do período,
além dos indicadores gerais. Compartilhar esse serviço com a API; otimizar somente
se houver medição que justifique. Não paralelizar leituras indiscriminadamente;
os testes em memória usam uma conexão. Não introduzir cache global de summary,
HTML ou gráfico que misture perfis.

## Identidade, dados ausentes e erros

Resolver `slug → ID` antes de cada leitura pessoal. `scopedRead` restringe as
referências a `listens`; `requireRelated` impede abrir mídia sem escutas naquele
perfil. Um item existir no catálogo global não o torna uma página daquele perfil.
Não usar `AllowUnplayed` na página de faixa: é uma exceção reservada ao tocando agora.

Os metadados de catálogo, incluindo relações, imagens e aliases, são compartilhados;
contagens, rank, primeira escuta, interesse, atividade e “novos” são pessoais.
Um álbum associado pode listar artistas globais de seus metadados sem que isso
represente escutas desses artistas pelo perfil. Não substituir relações de catálogo
por uma suposta propriedade exclusiva de um usuário.

Contrato recomendado para o HTML:

- ID não inteiro, zero, negativo ou overflow → 400; IDs positivos cabem em int32.
- Slug ausente/renomeado/excluído ou mídia não relacionada → 404.
- Falha real de banco/serviço no conteúdo essencial → 500, sem detalhes SQL.
- Falha isolada em atividade/interesse/histórico → bloco indisponível, preservando
  os blocos carregados corretamente; não fabricar zeros.
- Perfil vazio ou período sem dados de rewind → 200 e mensagem explícita.
- Álbum de uma faixa indisponível por falha de infraestrutura deve ser distinguido
  de metadado ausente; não fabricar título ou converter erro em “não encontrado”.

Datas devem usar o fuso efetivo e `<time datetime>` quando aplicável. Primeira
escuta zero não pode virar 1970. Cabeçalho all_time não deve aparentar responder
à query de período: rejeitar filtros contraditórios/não suportados na rota de
mídia; o ID da rota determina o filtro dos blocos, não um `track_id` arbitrário.

Troca de perfil em página de mídia leva à home do destino e remove IDs/paginação,
como já fazem Go e React. Na retrospectiva, preserva ano/mês e permite o estado
vazio no destino; não carrega a retrospectiva do perfil anterior por fallback.

## Gráficos com HTML e SVG

Extrair a composição da grade hoje embutida em `internal/webui/home.go` para um
helper com filtro, limiar de intensidade, fuso, início da semana e relógio. Reusar
nas páginas de mídia e na home; cobrir esse reaproveitamento com regressões.
Não aplicar um filtro de artista ao intervalo/contagem de outra entidade.

Para interesse, gerar SVG no Go com `viewBox`, curva/área e gradiente usando as
variáveis do tema. Reproduzir a curva e o acabamento descritos na análise inicial;
se a interpolação mudar, registrar a diferença nos pares visuais. Os 16 valores
continuam sendo contagens das divisões do backend, sem fabricar datas ou séries.

Prever zero, um único instante, contagens iguais e falha de serviço. Evitar divisão
por zero ou coordenadas NaN/infinito. Oferecer descrição e tabela/lista de valores
acessível; cor e hover não podem ser a única forma de ler os dados. Preferir dados
inline e SVG estático inicialmente, sem nova biblioteca de gráficos ou nova API.

O método SQLite lê todos os timestamps em memória, mesmo para 16 divisões; o
custo cresce com as escutas da entidade. Registrar esse limite, medir antes de
reescrever a agregação e manter cancelamento. Não expor `buckets` livremente no
HTML: usar 16 na primeira entrega. Validar o endpoint público existente, que hoje
não impede valores negativos ou enormes.

## Retrospectiva: contrato temporal proposto

Criar um normalizador próprio, por exemplo `webquery.Rewind`, em vez de simplesmente
chamar `HTML(home=false)`: este assume all_time nas listagens e não representa o
padrão especial da retrospectiva. Usar o fuso efetivo para escolher “agora”, ano,
mês e título, inclusive perto da meia-noite/virada de janeiro.

Recomendação inicial para implementação:

1. Preservar as URLs de ano/mês e o padrão último mês concluído/ano anterior em
   janeiro. Aceitar `month=0` como compatibilidade anual, normalizando a URL sem mês.
   Validar ano 1–9999 e mês 1–12; rejeitar NaN/overflow/combinações incompatíveis.
2. Manter o seletor em períodos concluídos. Aceitar URL explícita do período atual
   com indicação “em andamento”, sem inventar média de dias transcorridos: a média
   continua referida ao período de calendário completo. Rejeitar períodos futuros
   no HTML com 400 legível. A API genérica de summary conserva seus outros filtros.
3. Calcular uma vez os limites efetivos usados na consulta, no título e no rótulo.
   Manter os limites inclusivos em segundos do backend; não somar um segundo em
   todas as consultas globalmente para corrigir apenas o denominador das médias.
4. Corrigir a média de calendários completos para dias civis inclusivos no fuso
   escolhido. Janeiro de 2026 tem denominador 31; fevereiro, 28; fevereiro de 2024,
   29. Não dividir horas por 24 para contar dias em regiões com DST. Casos genéricos
   de semana/intervalo da API também precisam de contrato/teste antes de alterar
   seus resultados; não reduzir o endpoint às opções do formulário.
5. Usar uma única fonte de seleção, a URL, inclusive voltar/avançar. Mensal/anual
   pode ser formulário GET ou links, sem estado duplicado divergente. A preferência
   “mostrar tempo” pode usar CSS/JS pequeno, com alternativa GET (`show_time=1`)
   para funcionar sem JavaScript; não deve recalcular ou reordenar o summary.

São decisões propostas para a implementação, não comportamento já entregue.
A correção de médias altera resultados da API e deve ser documentada no changelog,
com testes comparando API e HTML. Não manter duas fórmulas para o mesmo summary.

## Pontos encontrados e grau de evidência

| Observação | Evidência e ação proposta |
| --- | --- |
| `GenerateSummary`: `int(t2.Sub(t1).Hours()/24)` com fim inclusivo a 23:59:59 | Reprodução isolada com o helper Go real: janeiro/2026 dá 30 em vez de 31; fevereiro dá 27 em vez de 28. Corrigir e testar na 008. |
| `GetInterest`: distribuição usa `(totalSeconds+1)`, mas rótulos usam divisão inteira de duração | Fórmulas reproduzidas: início 100, fim 110, 3 divisões; timestamp 110 entra na última, cujo fim publicado é 109. Definir fronteiras consistentes antes de usá-las em descrição/tooltip. |
| `GetInterest` só rejeita `Buckets == 0` | Leitura: negativo chega a `make` com tamanho negativo; valor enorme causa alocação sem teto. Validar API e composição; testar sem provocar alocação perigosa. |
| Interesse com timestamp inicial no futuro | Leitura: duração/índice podem ficar negativos ou o denominador zerar. Reproduzir com dados controlados e definir tratamento de timestamps futuros. Não confirmado em servidor. |
| `GetArtist`, `getAlbumByID`, `getTrackByID` ignoram `Scan` de contagem, duração e rank | Leitura: uma falha pode virar valor zero. Propagar erros; só ausência esperada pode produzir estado vazio. |
| Handlers JSON de detalhes devolvem 404 para qualquer erro | Leitura: falha de infraestrutura confundida com ausência. HTML deve distinguir; correção JSON requer teste de compatibilidade, sem expor SQL. |
| Parser `ParseIDParam` aceita zero/negativo | Leitura: valida tipo, sem exigir positivo. Aplicar validação explícita no contrato HTML e cobrir 0, -1 e overflow. |
| Seletor de rewind mantém estado local além da URL | Leitura: `useState(loadedYear/loadedMonth)` não sincroniza depois da carga; risco em voltar/avançar. Testar no navegador; usar URL na nova página. |
| `new Date(year, month+1)` para disponibilidade | Leitura: datas locais do navegador podem divergir do fuso forçado; testar primeira atividade no último dia e virada do mês. |
| Link MusicBrainz deriva `artist/album/track` do nome visual | Leitura: o MBID de álbum corresponde a release; o de faixa é recording. Mapear os tipos explicitamente, sem copiar `/album/` e `/track/` para os novos links externos. |
| Upload arrastado usa último segmento da URL como ID | Leitura: em faixa o tipo enviado é album, mas o ID extraído é o da faixa. Antes de reutilizar esse fluxo, passar tipo/ID explicitamente; testar IDs de faixa e álbum diferentes. |
| `TestGenerateSummary` tem corpo comentado | Leitura: esse nome de teste não valida os cálculos. Há testes de escopo no backend, mas é necessária cobertura de valores/datas na 008. |

Os dois primeiros cálculos estão registrados em
[analysis-checks.json](../../evidencias/roadmap-03/analysis-checks.json). Executou-se um
programa isolado em `.local/roadmap-03-study/`: chamou `TimeframeToTimeRange`,
reproduziu as fórmulas, sem abrir banco ou fazer HTTP. Os demais pontos acima
são achados de leitura; não foram apresentados como falhas reproduzidas no navegador.

## Administração transitória e ativação

Somente `/admin` não substitui o editor de mídia: a administração atual gerencia
principalmente perfis e contas, enquanto os modais ficam em `MediaLayout`.

Proposta concreta: aliases React `/{slug}/manage/artist/{id}`, `.../album/{id}` e
`.../track/{id}`, dentro do `ProfileLayout`, reutilizando componentes e loaders
existentes com IDs de rota explícitos. Essas rotas ainda **não existem**. Exibir
“Gerenciar este item” para sessão administrativa na página Go e fornecer link de
retorno para a página pública. Sem sessão, oferecer login em `/admin` e retorno
interno validado; chave de player não dá acesso às mutações. Não embutir editores
falsos nem transferir a identidade do perfil para cookie global.

Esses aliases precisam de teste de recarga, resolução de profile, troca de perfil,
merge nas duas direções, exclusão e retorno após editar/imagem/add-listen. Ajustar
extração frágil de ID no drag-and-drop antes de reaproveitar o componente. Mutação
de catálogo continua global e admin-only; adição/exclusão de escuta continua por
perfil. Reutilizar a confirmação e as proteções de sessão/origem existentes.
Não implantar novas mutações por GET.

Estender juntos `internal/webui/routing.go`, `client/api/web-ui.ts` e
`test_assets/web-ui-routes.json`: quatro páginas públicas passam a Go; aliases
manage, `/admin`, API, imagens e assets permanecem fora da ponte. Classificar
formatos de mídia conhecidos antes de validar o ID, para ID inválido receber 400
HTML em vez de cair acidentalmente na SPA. Preservar redirecionamentos legados,
`safeNext`, modo padrão React e opção `KOITO_WEB_UI=go`. A rota pública não deve
ser ativada parcialmente com blocos/gráficos/ações ainda inacessíveis.

## Organização e sequência de issues locais

```text
internal/webui/
  media.go, media_models.go             # composição dos três tipos
  activity.go, interest.go              # grade compartilhada e SVG
  rewind.go, rewind_models.go           # normalização aplicada e apresentação
  templates/media.html, rewind.html
internal/webquery/rewind.go             # contrato de ano/mês da página
internal/summary/                       # serviço existente + testes reais
client/app/routes/                     # aliases React transitórios de gestão
scripts/validate_webui.py               # ampliar cenários descartáveis
backlog/evidencias/roadmap-03/        # evidências, manifesto e pares futuros
```

Nomes propostos; não criar abstrações vazias nem duplicar SQL para atender à árvore.

1. [006 — páginas de mídia](<../../tasks/iss-006 - 006-—-Migrar-páginas-públicas-de-artistas-álbuns-e-faixas-para-Go.md>): composição, dados,
   cabeçalho, listas, links e erros. Não ativar as rotas de produção isoladamente.
2. [007 — gráficos de mídia](<../../tasks/iss-007 - 007-—-Migrar-atividade-e-interesse-das-páginas-de-mídia-para-HTML-SVG.md>): grade comum,
   interesse SVG, contratos de divisões e casos degenerados.
3. [008 — retrospectiva](<../../tasks/iss-008 - 008-—-Migrar-retrospectiva-mensal-anual-e-validar-seus-cálculos.md>): normalização, serviço
   compartilhado, médias, seletores e layout mensal/anual.
4. [009 — convivência, gestão e aceite](<../../tasks/iss-009 - 009-—-Integrar-mídia-rewind-Go-preservar-gestão-React-e-registrar-aceite.md>):
   aliases React, dispatcher, jornadas e capturas antes de ativar o conjunto.

Issues são Markdown locais, conforme orientação do usuário, sem publicação.
Nenhuma delas está implementada ou encerrada por este estudo.

## Aceite e comparação visual a executar

Usar bancos v3 descartáveis: dois perfis com estatísticas diferentes do mesmo
catálogo, um terceiro vazio, nomes longos/Unicode, múltiplos artistas, compilação,
capas ausentes, empates, só uma escuta, datas em meses/anos diferentes e itens novos
para um perfil mas antigos para outro. Não usar o banco que está em operação.

Cobrir dados essenciais e blocos opcionais, ID/slug inválido, perfil renomeado ou
excluído, item sem relação no destino, escaping, primeira escuta, rank empatado,
tocando agora relacionado/não relacionado e nenhum segredo em HTML público.
Resumo API/HTML deve coincidir em cada período/fuso; a correção das médias merece
regressão numérica explícita. Datas incluem janeiro, bissexto, DST, fuso forçado,
mês inferido, mês 0 legado, período atual, futuro e limites inclusivos.

Validar jornadas Go → artista → álbum → faixa → histórico/ranking filtrado →
retrospectiva, troca de perfil e duas abas; em seguida Go → gestão React → Go,
login, recarga, voltar/avançar e negativas de autorização/origem. Testes mutáveis
somente em bancos descartáveis. Comparar também com `KOITO_WEB_UI=react` para
confirmar que a opção anterior continua disponível.

| Par React/Go planejado | Estados e regiões |
| --- | --- |
| Artista | Cabeçalho, top 8, histórico 11, interesse/atividade, álbuns e mais de 99 |
| Álbum | Artista principal e compilação, top 8, histórico e gráficos |
| Faixa | Múltiplos artistas, álbum, execução atual compatível/incompatível |
| Rewind mensal | Período concluído, top 5 e minutos ligados/desligados |
| Rewind anual | Janeiro/ano anterior, seletor, títulos longos e ano vazio |
| Vazios/erros | Perfil vazio, período sem dados, imagem ausente, bloco indisponível |
| Gestão transitória | Link Go → editor React e retorno após ação de teste |

Capturar desktop e celular sobre cópias equivalentes do mesmo banco, com tema,
fuso, viewport e região de rolagem registrados em manifesto. Incluir Yuu e um
tema claro, regiões abaixo da dobra e teclado/foco. Não prometer ganho de
performance ou paridade pixel a pixel sem medição/comparação.

O [registro de evidências desta etapa](../../evidencias/roadmap-03/README.md) contém
apenas os cálculos do estudo e o plano de capturas. Ainda não há HTML Go de mídia
ou rewind para gerar pares novos. A captura React de artista do passo 2 continua
sendo referência histórica, não evidência de implementação da etapa 3.

Este estudo alterou somente documentação e evidências de análise. Não houve
execução da suíte do produto, alteração de código funcional, nova migration,
publicação de issues, commit/push ou mudança da configuração de execução.
