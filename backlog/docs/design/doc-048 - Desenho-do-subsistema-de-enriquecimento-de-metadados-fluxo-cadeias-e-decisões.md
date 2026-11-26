---
id: doc-048
title: >-
  Desenho do subsistema de enriquecimento de metadados (fluxo, cadeias e
  decisões)
type: other
created_date: '2026-11-24T19:54:00Z'
updated_date: '2026-11-24T19:54:00Z'
tags:
  - design
  - enriquecimento
---
# Desenho do subsistema de enriquecimento de metadados

Registro de 24/11/2026 do fluxo e das decisões do usuário para achar MBIDs e completar metadados. Substitui, no que divergir, o modelo de catálogo compartilhado do `ESTUDO.md` da ISSUE-055: o esquema vigente (migração 024, ISSUE-174) guarda observações **por perfil** em `metadata_values`.

## 1. Decisões do usuário (24/11/2026)

- **Fluxo:** escuta recebida, banco já salvo, MBID ausente, Navidrome se configurado (salva MBIDs e metadados), MusicBrainz se ausente (busca MBIDs e metadados), e por último ListenBrainz e os serviços comerciais (Last.fm, Deezer, Apple). Serviços locais e abertos antes dos proprietários.
- **MusicBrainz busca MBID por nome** (peça nova, ISSUE-216), condicionado à medição da ISSUE-215.
- **Duas cadeias:** uma de identidade (MBID) e uma de metadados.
- **Fontes comerciais desligadas por padrão**, ligadas pelo servidor; faltam os estudos de cada uma (ISSUE-221, 222 e 223). Sem scraping.
- O relay continua só para o ListenBrainz; o MusicBrainz serve a identidade e a metadados, nunca a envio.
- Tags populares separadas de gêneros; metadados de artistas incluídos; valores conflitantes de bibliotecas preservados por perfil.
- Interface e gráficos na ISSUE-212; enriquecimento gradual do histórico e monitoramento na ISSUE-213.
- A migração antecede a consolidação das migrações (ISSUE-195); ela é carregada pela ISSUE-214.

## 2. Fluxo e cadeias

| | **Cadeia A: identidade (MBID)** | **Cadeia B: metadados** (gênero, tag, ano, data) |
| --- | --- | --- |
| Quando roda | MBID ausente | Campo ausente, já com MBID (da escuta ou da cadeia A) |
| 1 | Banco já salvo | Banco já salvo |
| 2 | Navidrome (`search3`, ISSUE-174) | Navidrome (`getSong`, `getAlbum`, ISSUE-218) |
| 3 | MusicBrainz por nome e por ISRC (ISSUE-216) | MusicBrainz por MBID (ISSUE-217) |
| 4 | ListenBrainz `/1/metadata/lookup/`, exige token do relay (ISSUE-175); Last.fm se devolver MBID | ListenBrainz `/1/metadata/recording/`, sem token; depois Last.fm, Deezer e Apple, opcionais |
| Para quando | Primeiro candidato verificado | Cada campo preenchido, campo a campo |

Regras: a ordem é configuração (ISSUE-214 e 220), e não código; resultado ambíguo não é aceito e segue para a próxima fonte; se todas falharem o campo fica em branco, sem pendência e com cache negativo; nunca sobrescreve MBID recebido nem edição manual; todo valor grava fonte, instância e data; a rede nunca roda no caminho do ACK nem sob o lock de escrita.

## 3. Evidências de 24/11/2026

Chamadas reais com uma faixa pública, sem dados do usuário nem tokens.

| Chamada | Resultado |
| --- | --- |
| ListenBrainz `GET /1/metadata/lookup/` sem token | HTTP 401, «You need to provide an Authorization header.» |
| O mesmo com token inválido | HTTP 401, «Invalid authorization token.» |
| ListenBrainz `GET /1/metadata/recording/` por MBID, sem token | HTTP 200, com dados do artista |
| MusicBrainz `recording?query=recording:"Alive" AND artist:"Pearl Jam"`, sem chave | HTTP 200, 911 resultados; os três primeiros com `score` 100 e durações de 373, 342 e 331 s |

Conclusões: a busca por nome no ListenBrainz exige a conta (token do relay do perfil), e por isso perfis sem relay dependem do Navidrome e do MusicBrainz; o `score` do MusicBrainz não distingue versões, então a duração, os créditos e o álbum precisam desempatar. Amostra do Navidrome do usuário: `getSong` trouxe `genres` em 78 de 100 músicas, `year` em 98, `musicBrainzId` em 41, `isrc` em 26 e `bpm` em 14; o adaptador de escuta dele não envia gênero, tag, ano, data nem ISRC.

## 4. Modelo de dados

`metadata_values` (migração 024) é a fonte única das observações: um dos três vínculos (artista, gravação ou lançamento), `kind` (`mbid`, `genre`, `tag`, `recording_year`, `release_date`, `artist_origin_country`), valor, instância, fonte e data, por perfil. Hoje só `mbid` é gravado. Mudam nesta frente `enrichment_configuration` (vira o registro de provedores) e `enrichment_jobs` (uma linha por recibo e provedor); a produção ainda não viu a 024, então o ajuste entra no arquivo único da ISSUE-195.

## 5. Issues

| Issue | Assunto | Antes da 195 |
| --- | --- | --- |
| 214 | Interface única, registro de provedores e duas cadeias | Sim |
| 215 | Estudo: MusicBrainz por nome × ListenBrainz (regra de aceitação e ordem) | Não bloqueia |
| 216 | MusicBrainz por nome e ISRC (identidade) | Não |
| 217 | MusicBrainz por MBID (gêneros, tags, datas) | Não |
| 218 | Navidrome com gêneros, ano, datas, ISRC e BPM | Não |
| 219 | Capturar tags, ano e data enviados pelo cliente | Não |
| 220 | Card «Fontes de metadados» e conf.toml | Não |
| 221, 222, 223 | Estudos: Last.fm, Deezer, Apple | Não |
| 212 | Interface e gráficos por gênero e ano | Não |
| 213 | Monitoramento e enriquecimento do histórico | Não |
| 055 | Guarda-chuva: tags e datas das músicas | Não (a 195 passa a depender da 214) |

## 6. Implementação da ISSUE-214 (26/11/2026)

Migração **025** (`enrichment_providers` e `enrichment_provider_runs`, no lugar de `enrichment_configuration`), pacote `internal/enrichment` (registro, cadeias, descritores) e executor em `internal/db/sqlite/enrichment.go`.

- **Registro.** Cada fonte é uma linha: ligada, disponível, prioridade, cadeias e elegibilidade. O código declara o que a fonte é (descritor); o banco guarda só as escolhas do dono (ligada e prioridade). Ao iniciar, o motor insere as linhas que faltam, então **uma fonte nova não precisa de migração**.
- **Cadeias.** O executor percorre a cadeia de identidade e depois a de metadados na ordem armazenada, com uma gravação protegida por fence por provedor (mesma verificação de dono, versão de resolução e impressão digital do catálogo de antes, mais a renovação do lease entre etapas). A ordem é lida a cada recibo, então mudar prioridade ou desligar uma fonte vale sem reiniciar.
- **Parada por campo.** Cada fonte decide pelo pedido o que ainda está aberto e responde «ausente» sem rede quando nada falta. `Request.Prior` entrega as declarações que outras fontes já gravaram. Resposta ambígua nunca é gravada e a cadeia segue.
- **Recuperação.** Só se grava a resposta que mudou algo (encontrada, ambígua ou falha). Medido em 5.000 escutas, uma linha por recibo e provedor custaria cerca de 260 bytes por escuta (11% dos 2.382 B), quase toda de «nada a acrescentar». Um job recuperado pula as respostas gravadas e pergunta de novo às demais, limitado a duas tentativas por recibo.
- **Orçamento do MusicBrainz.** O cliente expõe a própria fila como `Budget`; o resolvedor e qualquer provedor futuro usam a mesma taxa de 1 requisição por segundo.
- **Compatibilidade.** `ConfigureEnrichment`, `ResolveNextEnrichment` e a variante com fallback continuam como adaptadores finos, e os testes das ISSUE-174 e 175 passam sem alteração de comportamento.

**Fica para as issues seguintes:** gravar gêneros, tags e datas (217, 218); cache negativo persistente entre escutas, que só importa quando o MusicBrainz por nome existir (216); a tela de configuração (220), para a qual `SetEnrichmentProvider` e `EnrichmentProviders` já existem; e o primeiro consumidor do orçamento do MusicBrainz (216). A elegibilidade `profile-relay` hoje é específica do ListenBrainz.

**Para a ISSUE-195:** a 025 entra no arquivo único junto com a 024. A trava de bancos de preview do `bootstrap.go` (versões 24 a 28) precisa ser reescrita para o esquema final.
