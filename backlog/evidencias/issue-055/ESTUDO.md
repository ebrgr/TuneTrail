# ISSUE-055 — Novo estudo: enriquecer metadados quando uma escuta chega

Estudo de 22/11/2026, refeito sobre o schema 21 (v0.65). A versão anterior da 055 falava em "SQLite v3". Pedido do usuário: ao chegar uma escuta nova, tentar primeiro o servidor Navidrome do usuário, depois o ListenBrainz e, se falhar, Last.fm, Apple ou outra fonte; avaliar o fim anunciado da API do Last.fm e a alternativa de extrair o HTML. Nada foi implementado.

## 1. O que foi confirmado e o que não foi

| Afirmação | Situação |
| --- | --- |
| OpenSubsonic expõe `genres` (lista de objetos com `name`), `musicBrainzId` e `originalReleaseDate` (extensão, tipo data) | Confirmado na [documentação do tipo Child](https://opensubsonic.netlify.app/docs/responses/child). O `musicBrainzId` de uma música pode ser o ID da gravação, conforme o servidor. A data original aparece em estruturas de álbum; conferir no JSON real do Navidrome. Não achei campo de "tags" livres em `getSong`: gênero é o que existe. |
| ListenBrainz resolve metadados por MBID de gravação com `inc` = `artist`, `tag`, `release` | Confirmado na [documentação do endpoint](https://listenbrainz.readthedocs.io/en/latest/users/api/metadata.html). Não confirmei que `release` devolve a data de lançamento. O `lookup` por nome de artista e faixa exige token, segundo o [blog do MetaBrainz](https://blog.musicbrainz.org/author/ruaok/), e uma discussão da comunidade relata que o `lookup` pode apontar para outra versão do lançamento. |
| MusicBrainz tem tags, gêneros e datas de lançamento com precisão parcial, via `inc=tags+genres+release-groups` | Conhecimento do serviço; ver o [guia da API](https://musicbrainz.org/doc/MusicBrainz_API). Exige User-Agent e cerca de 1 requisição por segundo. A licença dos dados suplementares (tags e notas) difere da dos dados centrais: **verificar antes de guardar e exibir**. |
| A API do Last.fm ficará privada a partir de dezembro | **Não encontrei nenhum anúncio.** A [introdução oficial da API](https://www.last.fm/api/intro) segue publicando a API, com uso comercial sob contato com a empresa. Peço a fonte (link ou e-mail) antes de planejar em torno disso. Convém desenhar de modo que a perda do Last.fm seja só desligar um provedor. |
| Apple e Deezer têm busca pública com gênero e data | Conhecimento geral, **não verificado nesta rodada**: a busca iTunes não pede chave e devolve gênero principal e data; a API pública do Deezer devolve gênero e data do álbum. Os dois têm limites de taxa e termos que restringem armazenar e redistribuir os dados. Verificar os termos antes de adotar. |

## 2. Cadeia de fontes recomendada

Princípio: fonte mais próxima do usuário e mais confiável primeiro; fontes abertas e identificáveis por MBID antes das que dependem de busca por texto; serviços comerciais por último e opcionais.

1. **Valores enviados na própria escuta** (`additional_info`: tags, ano, data, MBIDs). Primeira fonte, sem rede. Hoje o TuneTrail já recebe `tags` e não as guarda.
2. **Servidor Navidrome do próprio usuário** (API Subsonic já configurada para as ISSUE-174 e 088): `getSong` ou `getAlbum` pelo ID da faixa. Gênero, ano, data original e MBIDs. É o dado que o usuário curou. Sem custo de terceiros e sem enviar dados para fora.
3. **ListenBrainz** por MBID de gravação (`/1/metadata/recording`, `inc=tag release`): tags e lançamento. Só quando existir MBID; busca por nome só com token e com verificação de ambiguidade.
4. **MusicBrainz** por MBID (gravação, lançamento, grupo de lançamento): datas com precisão parcial e gêneros. É a referência para ano e data. Fila com 1 req/s.
5. **Opcionais, desligados por padrão:** Last.fm (`track.getTopTags`, chave gratuita e uso não comercial), Apple (busca iTunes) e Deezer. Servem de plano B para tags de gosto popular quando as abertas não têm nada.
6. **Extração de HTML (scraping): não recomendo.** Viola os termos da maioria dos serviços, quebra com qualquer mudança de página, é bloqueada (o MetaBrainz relata medidas contra coletores) e traz risco jurídico para um projeto que será distribuído. Se o Last.fm fechar a API, a resposta é tirar o provedor, não raspar o site.

Regra por campo: tags livres e gênero vêm de várias fontes e se somam com procedência; ano e data seguem a precedência 1 > 2 > 4 > 3 e nunca são sobrescritos por fonte de ordem pior nem por edição manual.

## 3. Quando e como buscar (nunca no caminho do ACK)

- A escuta é aceita e respondida como hoje. O enriquecimento é **assíncrono**, em fila própria, disparado quando o resolvedor liga a escuta a uma gravação ou lançamento **sem metadado fresco**. Escutas repetidas da mesma faixa não geram nova busca.
- Uma busca por entidade (gravação, lançamento), deduplicada por MBID. Resultado negativo (nada encontrado, erro, provedor desligado) vira **cache negativo** com prazo e tentativa futura, sem retry infinito e sem pendência (mesma regra da 174 e 175).
- Disjuntor por provedor: timeout curto, limite de taxa, pausa após erros e retomada, como no relay. Falha de provedor nunca atrasa nem impede o registro de escutas e nunca apaga dado existente.
- Privacidade: só vai para fora o necessário (MBID ou título e artista). Nada do perfil, da chave ou do histórico. O Navidrome consultado é o do próprio usuário. Cada provedor liga e desliga na configuração do servidor, com os externos desligados por padrão.
- Importações de arquivo (milhares de escutas) usam a mesma fila com prioridade baixa e teto por minuto, para não saturar MusicBrainz e Navidrome.

## 4. Modelo de dados proposto (aditivo sobre o schema 21)

O catálogo é compartilhado entre perfis. Tags e datas são dado de catálogo, sem dado pessoal; se algum dia houver tag pessoal, vai em tabela por perfil.

- `releases`: colunas `release_date` (texto `YYYY`, `YYYY-MM` ou `YYYY-MM-DD`), `release_date_precision`, `original_year`, `date_source`, `date_fetched_at`. Distinguir lançamento específico de ano original do grupo de lançamento.
- `recordings`: `first_release_year` e a mesma procedência, se a gravação tiver data própria.
- `tags` (`id`, `name`, `normalized`) e `recording_tags` (`recording_id`, `tag_id`, `source`, `count` ou peso, `fetched_at`, `kind`: genre ou tag). Mesma estrutura opcional para `release_tags` e `artist_tags` (decidir se artista entra).
- `metadata_fetch_log` (`entity_kind`, `entity_id`, `provider`, `status`, `fetched_at`, `next_retry_at`): cache negativo e auditoria.
- `metadata_overrides`: edição manual do usuário, que sempre vence.
- Índices em `tag_id` e em `(entity, provider)`. O tamanho e o custo das consultas de ranking por tag ou gênero entram na medição.
- Snapshot do recibo: campos de tags, gêneros, ano e data declarados pela fonte, com a mesma procedência (como `mbid_source` hoje). Exportação e reimportação (ISSUE-183) incluem tags e datas.

Esta migração entra na janela mínima antes da consolidação (ISSUE-195).

## 5. Ligações com outras issues

174 e 175 (MBID ausente) usam a mesma cadeia Navidrome → ListenBrainz. Recomendo **um só subsistema de enriquecimento** com interface de provedor, fila, disjuntor e cache negativo, usado pelas 174, 175 e 055, em vez de três implementações. A 193 (normalização) atua antes da associação; a 190 e a 097 (duração) são outro metadado do mesmo subsistema. A 121 e a 145 não são afetadas.

## 6. Cortes sugeridos

1. Contrato dos provedores e da fila de enriquecimento, com cache negativo, disjuntor e configuração (sem dados novos ainda), reaproveitando o que a 174 precisar.
2. Guardar o que já chega na escuta (tags, ano e data do `additional_info`) com procedência; migração aditiva.
3. Provedor Navidrome (gênero, ano, data original).
4. Provedores ListenBrainz e MusicBrainz por MBID.
5. Provedores opcionais (Last.fm, Apple, Deezer), um por vez, cada um com revisão de termos.
6. Apresentação: página da faixa e do álbum com gêneros e datas, filtros e rankings por gênero e ano, respeitando procedência.

Reviewer em cada corte; Advisor (effort high) no corte 1, por tocar fila, concorrência e chamadas externas.

## 7. Decisões que preciso do usuário

1. Fonte da informação sobre o Last.fm em dezembro (link). Com ou sem ela, Last.fm, Apple e Deezer entram só como opcionais desligados?
2. Extração de HTML fica fora (recomendação)? Sim ou não.
3. Tags de gosto popular (Last.fm) misturadas com gêneros abertos, ou separadas e exibidas à parte?
4. Artistas também recebem gêneros e tags, ou só gravações e lançamentos?
5. Gênero e tag aparecem na interface no primeiro corte ou só depois de os dados existirem (corte 6)?
6. Um único subsistema de enriquecimento para 174, 175 e 055 (recomendado)?
