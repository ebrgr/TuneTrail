---
id: ISSUE-055
title: 055 — Aproveitar tags e datas de lançamento das músicas
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-24T19:54:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-13
dependencies:
  - ISSUE-079
  - ISSUE-214
  - ISSUE-217
  - ISSUE-218
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
type: feature
ordinal: 55000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/055-metadados-tags-lancamento.md`; reescrita em 24/11/2026 sobre o schema 24. O texto anterior falava em SQLite v3 e está no histórico do Git.

**Roadmap:** [metadados musicais](<../docs/roadmaps/doc-001 - ROADMAP.md>) · **Desenho:** doc-048 · **Estudos:** `backlog/evidencias/issue-055/ESTUDO.md` e `ESTUDO-COMPLEMENTAR.md`

## Objetivo

Aproveitar gêneros, tags, ano e datas de lançamento de gravações, lançamentos e artistas, vindos do que o cliente envia e de fontes consultadas: Navidrome, MusicBrainz, ListenBrainz e, opcionalmente, Last.fm, Deezer e Apple. Esta issue é o guarda-chuva; o trabalho está nas issues abaixo.

## Situação confirmada em 24/11/2026

- A tabela `metadata_values` (migração 024, ISSUE-174) já aceita os tipos `mbid`, `genre`, `tag`, `recording_year`, `release_date` e `artist_origin_country`, por perfil, com fonte, instância e data. Hoje só `mbid` é gravado e nada lê os demais tipos.
- O snapshot da escuta não copia `additional_info.tags` e não tem ano nem data. O adaptador do Navidrome 0.64.2 não envia gênero, tag, ano, data, ISRC nem BPM.
- O MusicBrainz só é consultado por MBID; a busca por nome é nova.

## Decomposição

ISSUE-214 subsistema e cadeias (antes da 195) · 215 estudo da medição · 216 MusicBrainz por nome e ISRC · 217 MusicBrainz por MBID · 218 Navidrome com metadados · 219 captura do que o cliente envia · 220 configuração das fontes · 221, 222 e 223 estudos de Last.fm, Deezer e Apple · 212 interface e gráficos · 213 monitoramento e histórico.

## Decisões do usuário

Registradas em doc-048: separar tags de gêneros, incluir artistas, preservar conflitos por perfil, comerciais desligados por padrão, sem scraping.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Gêneros, tags, ano e datas de gravação, lançamento e artista gravados em metadata_values, por perfil, com fonte, instância e data (ISSUE-217 e 218)
- [ ] #2 Tag popular distinta de gênero; datas com precisão parcial; edição distinta do primeiro lançamento do grupo
- [ ] #3 Ausência ou erro de uma fonte nunca apaga dado anterior, e a edição manual prevalece
- [ ] #4 Migração, exportação e reimportação v2 cobertas, com isolamento entre perfis testado (ISSUE-214)
- [ ] #5 Documentação e CHANGELOG atualizados; interface na ISSUE-212; aceite do usuário
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:39:00Z

---

Plano TuneTrail 1.0 (doc-032): ROADMAP item 7, Fase 4 · Insights. A migração de metadados deve ser coordenada com a v4 da ISSUE-079/080.

---

author: claude
created: 2026-11-23T00:31:00Z

---

Atualização (22/11/2026): a descrição acima está defasada. Ela fala em 'SQLite consolidado v3' e no bootstrap limitado à versão 3; hoje o schema está na versão 21 (v0.65) com o modelo de gravações (recordings), ocorrências (occurrences) e lançamentos (releases), recibos de ingestão (ingest_events) e escutas com event_id e origin. Antes de implementar, refazer o plano sobre esse modelo: onde ficam tags (gravação, lançamento, artista?), gênero separado de tag livre, ano original x data do lançamento x data da gravação com precisão parcial, procedência por campo, política de dados conflitantes no catálogo compartilhado entre perfis, e o que entra no Snapshot do recibo (additional_info.tags, ano e data, MBIDs do grupo de lançamento) e na exportação e reimportação (ISSUE-183). Ordem: por decisão do usuário, esta issue vem ANTES da consolidação das migrações (ISSUE-195), que só roda depois dela e da limpeza do código legado (ISSUE-194), para que o schema final já inclua estes metadados. Como produção nunca viu as migrações 003 a 021, a migração desta issue pode ser aditiva e depois dobrada no arquivo único da 195. Relacionadas que também mexem em metadados ou procedência: 174, 175 (MBID declarado com fonte e data), 190 e 097 (duração) e 193 (normalização).

---

author: claude
created: 2026-11-23T00:38:00Z

---

Decisão do usuário (22/11/2026): esta issue está no mínimo que altera o banco antes do primeiro upgrade de produção e antecede a consolidação das migrações (ISSUE-195). Migração aditiva agora; depois dobrada no arquivo único da 195.

---

author: claude
created: 2026-11-23T01:19:00Z

---

Novo estudo em backlog/evidencias/issue-055/ESTUDO.md (22/11/2026): cadeia de fontes (escuta, Navidrome, ListenBrainz, MusicBrainz, opcionais Last.fm/Apple/Deezer), enriquecimento assíncrono com cache negativo e disjuntor, modelo aditivo sobre o schema 21, seis cortes e seis decisões do usuário. Não encontrei anúncio de privatização da API do Last.fm; pedida a fonte. Scraping de HTML não recomendado.

---

author: codex
created: 2026-11-24T11:30:00Z

---

Investigação de 24/11: Navidrome 0.64.2 envia nomes, duração, posição e MBIDs disponíveis, mas não gênero, ano nem ID Subsonic; API testada com amostras sanitizadas e busca com caso sem correspondência. Evidências e correção sobre Last.fm em backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md.

---

author: codex
created: 2026-11-24T11:42:00Z

---

Decisões do usuário em 24/11/2026: fontes primárias OpenSubsonic configurado e/ou ListenBrainz/MusicBrainz; fallback opcional Last.fm, Apple e Deezer quando faltarem dados, mediante configuração e respeitando termos; sem scraping. Tags populares separadas dos gêneros; incluir artistas. Interface e gráficos em tarefa posterior. Preservar valores conflitantes de bibliotecas por perfil. Enriquecer histórico gradualmente, com limites configuráveis de requisições e informações na central de monitoramento integrada à central de saúde. Subsistema compartilhado começa pelas ISSUE-174 e 175.

---

author: codex
created: 2026-11-24T11:51:00Z

---

Decisão do usuário em 24/11/2026: consultar provedores para obter MBID de música e artista apenas quando o identificador correspondente não vier nos metadados da escuta do Navidrome; tratar cada entidade independentemente, sem substituir identificadores recebidos. Consultas para gênero, ano e outros dados podem ocorrer mesmo quando o MBID já estiver presente.

---

author: claude
created: 2026-11-24T19:54:00Z

---

24/11/2026: issue reescrita sobre o schema 24 e dividida nas ISSUE-214 a 223, conforme as decisões do usuário registradas em doc-048. O modelo de catálogo compartilhado do ESTUDO.md fica substituído por metadata_values por perfil. A ISSUE-195 passa a depender da 214, que carrega a migração; esta issue segue como guarda-chuva e não bloqueia a consolidação.

---
<!-- COMMENTS:END -->
