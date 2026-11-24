---
id: doc-047
title: Auditoria de uso e repetição dos dados do banco (schema 23)
type: other
tags:
  - estudo
  - banco-de-dados
created_date: '2026-11-24T00:45:00Z'
---
# Auditoria de uso e repetição dos dados do banco (schema 23)

**Complemento de 24/11/2026.** O usuário concordou com a avaliação da Central de Saúde na v0.75 (schema 24). A [revisão de Health, Inbox, ingestão e Outbox](../../evidencias/db-auditoria/REVISAO-HEALTH.md) registra os fluxos, o ensaio corrigido de 300 mil recibos, as medições e a ordem de execução. Os números históricos abaixo continuam sendo os do schema 23.

Estudo de 23/11/2026, sobre a `main` na v0.74. Pedido do usuário: **saber e filtrar o que é útil e o que não é dentro do banco**, porque ele parece ter dados demais e repetidos. Complementa o doc-046 (modelo, schema, migrações e plano de atualização). Nada foi alterado no código nem no banco do usuário: o estudo mede e propõe.

## 0. Resposta curta

- **A suspeita está certa, e o excesso está num lugar só: o recibo.** Cada escuta que chega pela API custa **2.382 bytes** no banco, e **65% disso é a tabela `ingest_events`** (o recibo da submissão), contra 13% da própria escuta (`listens`). O Koito v2 gastava **cerca de 120 bytes por escuta** (catálogo incluído; 35 MB para 300 mil escutas).
- **Há repetição provada nos dados.** As 5.000 revisões de `ingest_revisions` são **idênticas** ao recibo; `resolved_at` e `closed_at` valem o mesmo em **5.000 de 5.000** recibos; o horário do recibo (texto de 30 bytes) repete o inteiro de `listens`; `listen_export_order` é **exatamente uma linha por escuta**.
- **Das 283 colunas, 207 ficam como estão.** Das demais, **29 saem** (sem uso ou constantes legadas), **27 se simplificam**, e **20 dependem de uma decisão sua**. Quatro tabelas inteiras saem e seis se fundem ou simplificam.
- **A economia estimada** vai de **−10% (sem mudar nada que o usuário veja)** a **−18% (moderada)** por escuta, e há mais, mas depende de decisões de produto (seção 5).
- **Você pode filtrar sozinho:** o arquivo `colunas.csv` tem uma linha por coluna, com a evidência e o veredito, e o script que o gera pode ser rodado de novo.

## 1. Como foi medido, e o que não foi

**Banco de teste.** Um banco novo, criado pelo binário da v0.74, recebeu **5.000 escutas pela API compatível com ListenBrainz**, como um scrobbler faria (formato do Navidrome: `media_player`, `submission_client`, número da faixa, duração, MBIDs em parte das escutas). O resolvedor em segundo plano aceitou as 5.000 (cerca de 32 por segundo): 5.000 escutas, 2.435 faixas, 327 artistas, 676 lançamentos, 11,4 MB. Os dados do usuário não foram usados; o banco de desenvolvimento dele só serviu de referência anterior.

**Quatro fontes de evidência por coluna.**

1. **Código** (medido): quantas consultas SQL do Go de produção a **leem** e quantas a **escrevem** (análise por consulta; testes e a lista de layout ficam de fora). 482 consultas analisadas.
2. **Banco** (medido): se um índice, gatilho, view, coluna gerada ou chave estrangeira depende dela.
3. **Dados** (medido): quantas linhas estão preenchidas, quantos valores distintos há, se é constante, e o tamanho médio.
4. **Repetição** (medido): colunas que valem o mesmo em todas as linhas, e consultas que comparam a coluna com outra tabela.

**O que esta medição não cobre. Leia antes de agir.**

- **«Vazia» não é «inútil».** O cliente de teste não envia `tags`, `duration_played`, número de disco, nem usa importação de arquivo, identificador externo ou os estados de falha (tentativas, estacionamento, reprocesso). Colunas que só esses caminhos preenchem aparecem vazias aqui e **continuam necessárias**. O veredito delas é «manter».
- **«Constante» também não é «legado».** Num fluxo sem falha, `state` é sempre `accepted`. Só as constantes que o **código já proíbe variar** (modo, variante do resolvedor, política de entrega) foram marcadas como legado.
- **Os índices do recibo servem a páginas** (saúde, Inbox, fila de revisão). O estudo diz quanto custam, **não** se as páginas dependem deles; isso exige ler o plano de consulta antes de remover.
- **O banco de produção terá outro perfil.** Num Koito v2, as tabelas de recebimento começam vazias, e o custo por escuta nova é o da seção 2.

## 2. Por que o banco parece ter dados demais

| Tabela | Linhas | Tabela (KB) | Índices (KB) | Bytes por escuta | % do total |
| --- | ---: | ---: | ---: | ---: | ---: |
| `ingest_events` | 5000 | 4672 | 2920 | 1.555 | 65.3% |
| `listens` | 5000 | 412 | 1092 | 308 | 12.9% |
| `ingest_revisions` | 5000 | 696 | 252 | 194 | 8.1% |
| `listen_export_order` | 5000 | 224 | 236 | 94 | 4.0% |
| `track_aliases` | 2435 | 80 | 108 | 39 | 1.6% |
| `occurrences` | 2435 | 56 | 116 | 35 | 1.5% |
| `artist_tracks` | 2435 | 36 | 100 | 28 | 1.2% |
| `tracks` | 2435 | 40 | 52 | 19 | 0.8% |
| `recordings` | 2435 | 64 | 24 | 18 | 0.8% |

**Cada escuta custa cerca de 20 vezes mais do que no Koito v2.** No v2, o banco inteiro (catálogo incluído) gastava cerca de 120 bytes por escuta (35 MB para 300 mil escutas). Hoje são 2.382 bytes por escuta no banco de teste, e só as quatro tabelas que descrevem uma escuta nova (recibo, revisão, escuta e ordem de exportação) somam **2.151 bytes**. Projeção linear, **a medir em escala**: 300 mil escutas pela API ocupariam cerca de **645 MB**, contra 35 MB no v2.

O recibo, sozinho, tem **63 colunas e 20 índices** (18 criados por SQL e 2 automáticos), que custam 957 bytes de tabela e 598 bytes de índices por escuta. Os quatro índices maiores (`idx_ingest_source_receipt`, `idx_ingest_health_decisions`, `idx_ingest_inbox_example` e `idx_ingest_health_received`) somam **328 bytes por escuta** e existem para as páginas de saúde.

Dos 750 bytes de conteúdo de cada recibo (a tabela ocupa mais por causa do cabeçalho de cada linha e das páginas), o `original_snapshot` (o conteúdo submetido, em JSON) ocupa 277 e o `snapshot_digest` 64. O resto são identificadores de 36 caracteres (`receipt_id`, `event_id`, `batch_id`), o horário em texto (30) e as colunas de decisão (`reason_codes` 29, `evidence` 22, `rule_version` 13).

## 3. O que é repetido (provado nos dados)

| # | Repetição | Prova (5.000 escutas) | Custo por escuta |
| --- | --- | --- | --- |
| R1 | `ingest_revisions` repete a decisão do recibo (`reason_codes`, `evidence`, `rule_version`) e **nenhuma consulta a lê**. | 5.000 de 5.000 revisões idênticas ao recibo; 0 leituras no Go. | **194 B** (8%) |
| R2 | `ingest_events.listened_at` é texto RFC3339 de 30 bytes e repete `listens.listened_at`, que é inteiro. | 5.000 de 5.000 iguais depois de converter. | 26 B a mais que um inteiro |
| R3 | `closed_at` repete `resolved_at`. | 5.000 de 5.000 iguais nos recibos aceitos. | ~4 B |
| R4 | `listen_export_order` guarda uma linha por escuta, repetindo `event_id` (texto de 36 bytes) com dois índices. | 5.000 linhas para 5.000 escutas. | **94 B** (4%) |
| R5 | O resultado padrão do caminho normal (`reason_codes`, `evidence`, `rule_version`) é igual em todos os recibos. | 1 valor distinto nas 5.000 linhas. | 64 B |
| R6 | Constantes legadas `mode`, `resolver_variant` e `delivery_policy`. | 1 valor distinto cada; os gatilhos de admissão já proíbem outro. | 23 B |
| R7 | `listens.user_id` repete o dono já guardado em `profile_id`. | 0 de 5.000 preenchidas; só tem valor no histórico migrado do v2. | índice `idx_listens_user_id` |
| R8 | `tracks`, `occurrences` e `recordings` guardam a mesma música três vezes, com o mesmo ID. | 2.435 = 2.435 = 2.435; `tracks.musicbrainz_id` vazio, porque o MBID mora em `recordings`. | 227 B por música nas quatro tabelas (parte é necessária); o desperdício está nas chaves repetidas e nos 11 gatilhos de ponte |
| R9 | O `original_snapshot` repete artista, título, lançamento e MBIDs que também estão no catálogo. | 277 B por recibo, 37% dos dados do recibo. | 277 B (decisão de produto) |

**Falsos positivos que descartei.** `occurrence_id` e `recording_id` valem o mesmo em todos os recibos do banco novo, mas só porque os dois contadores nasceram juntos: num Koito v2 migrado os números diferem. **Não são** repetição. `api_keys.user_id` bate com o dono do perfil, mas é atribuição histórica barata (1 linha por chave).

## 4. O que é útil e o que não é

### 4.1 Visão geral das 283 colunas

| Veredito | Colunas | Significado |
| --- | ---: | --- |
| Manter | 207 | Chaves, colunas ativas, geradas que sustentam índice, e as que dependem do cliente ou da importação. |
| Simplificar | 27 | Existem, mas repetem outra ou guardam sempre o valor padrão. |
| Decidir | 20 | Só recebem escrita, ou só aparecem em constantes do Go. Precisam de uma resposta sua. |
| Remover | 29 | Sem nenhuma referência, constante legada, ou de tabela sem escritor. |

Por classe de evidência: chave (96), ativa (73), ativa, constante (36), ativa, vazia (27), só escrita (16), revisar (14), lida, nunca escrita (8), morta (8), gerada (4), só no banco (1).

### 4.2 Tabelas com algo a filtrar

| Tabela | Colunas | Mantém | Simplifica | Decidir | Remove | Por quê |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| `ingest_events` | 63 | 49 | 5 | 1 | 8 | 63 colunas; 8 sem uso ou constantes legadas, 5 repetidas ou constantes do caminho normal. |
| `ingest_revisions` | 9 | 0 | 9 | 0 | 0 | Repete 100% o recibo e ninguém lê. |
| `ingest_event_details` | 8 | 0 | 0 | 0 | 8 | Nenhum escritor no código de produção. |
| `integrations` | 9 | 2 | 0 | 6 | 1 | Colunas sem uso, possivelmente reservadas às ISSUE-086 a 091. |
| `outbox_details` | 6 | 0 | 0 | 0 | 6 | Nenhum escritor no código de produção. |
| `ingest_observations` | 10 | 6 | 0 | 4 | 0 | Duas colunas só de escrita. |
| `tracks` | 4 | 0 | 4 | 0 | 0 | Ponte legada, repetida em `occurrences` e `recordings`. |
| `profile_relays` | 3 | 0 | 3 | 0 | 0 | Relação 1:1 com `relay_destinations` mantida por 5 gatilhos. |
| `event_origins` | 2 | 0 | 2 | 0 | 0 | Nove valores fixos: vira `CHECK`. |
| `ingest_discards` | 8 | 6 | 0 | 2 | 0 | Duas colunas só de escrita, de auditoria por desenho. |
| `ingest_source_summary` | 6 | 4 | 0 | 2 | 0 | Duas colunas citadas só por constantes do Go. |
| `legacy_relay_binding` | 2 | 0 | 0 | 0 | 2 | Marca o bootstrap de relay por variáveis de ambiente (decisão sua). |
| `listen_export_order` | 2 | 0 | 2 | 0 | 0 | Uma linha por escuta repetindo `event_id`. |
| `listens` | 9 | 7 | 0 | 0 | 2 | `review_distinct` morta e `user_id` repete o dono do perfil. |
| `outbox` | 18 | 16 | 0 | 2 | 0 | Duas colunas só de escrita. |
| `profile_ingest_review` | 2 | 0 | 2 | 0 | 0 | Uma coluna de `listening_profiles` basta. |
| `relay_delivery_gate` | 2 | 0 | 0 | 0 | 2 | Vale `durable` em todo banco. |
| `sessions` | 5 | 3 | 0 | 2 | 0 | Duas colunas só de escrita no Go. |
| `relay_sent_archive` | 3 | 2 | 0 | 1 | 0 | Contagem que ninguém lê. |

### 4.3 `ingest_events`: as 63 colunas, uma a uma (só o que não é «manter»)

| Veredito | Coluna | Bytes médios | Motivo |
| --- | --- | ---: | --- |
| remover | `uncertainty_ms` | — | Nenhuma referência no Go de produção nem em índice, gatilho ou view. |
| remover | `client_name` | — | Nenhuma referência no Go de produção nem em índice, gatilho ou view. |
| remover | `client_version` | — | Nenhuma referência no Go de produção nem em índice, gatilho ou view. |
| remover | `normalized_snapshot` | — | Nenhuma referência no Go de produção nem em índice, gatilho ou view. |
| remover | `normalizer_version` | — | Nenhuma referência no Go de produção nem em índice, gatilho ou view. |
| remover | `mode` | 6.0 | Constante legada: padrão da época da sombra; todo recibo entra com um único valor. |
| remover | `resolver_variant` | 10.0 | Constante legada: padrão da época da sombra; todo recibo entra com um único valor. |
| remover | `delivery_policy` | 7.0 | Constante legada: padrão da época da sombra; todo recibo entra com um único valor. |
| simplificar | `listened_at` | 30.0 | Texto RFC3339 de 30 B que repete listens.listened_at (inteiro): guardar como inteiro. |
| simplificar | `reason_codes` | 29.0 | Resultado padrão do caminho normal (idêntico em todos os recibos): gravar só quando fugir do padrão. |
| simplificar | `evidence` | 22.0 | Resultado padrão do caminho normal (idêntico em todos os recibos): gravar só quando fugir do padrão. |
| simplificar | `rule_version` | 13.0 | Resultado padrão do caminho normal (idêntico em todos os recibos): gravar só quando fugir do padrão. |
| simplificar | `closed_at` | — | Igual a resolved_at em todos os recibos aceitos: uma coluna de tempo basta (conferir os demais estados). |
| revisar | `batch_id` | 36.0 | Só escrita (36 B por recibo): expor no recibo ou remover. |

As outras 49 colunas ficam: chaves, estados de tentativa, colunas geradas que sustentam a Inbox e a saúde, e as que dependem de importação de arquivo ou de identificador externo (`file_digest`, `file_format`, `external_event_id`, `client_idempotency_key` e afins), que estão vazias aqui só porque o teste usou a API.

### 4.4 As 20 colunas que dependem de uma decisão sua

| Coluna | Evidência | Pergunta |
| --- | --- | --- |
| `ingest_discards.actor_id` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `ingest_discards.origin_state` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `ingest_events.batch_id` | 0 leituras e 2 escritas no Go; 5000 linhas preenchidas | Só escrita (36 B por recibo): expor no recibo ou remover. |
| `ingest_observations.schema_version` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `ingest_observations.received_at` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `ingest_observations.closed_at` | 0 leituras e 0 escritas no Go; 0 linhas preenchidas | Referenciada só por constantes de Go; conferir à mão. |
| `ingest_observations.protected` | 0 leituras e 0 escritas no Go; 0 linhas preenchidas | Referenciada só por constantes de Go; conferir à mão. |
| `ingest_source_summary.received_at` | 0 leituras e 0 escritas no Go; 1 linhas preenchidas | Referenciada só por constantes de Go; conferir à mão. |
| `ingest_source_summary.receipt_id` | 0 leituras e 0 escritas no Go; 1 linhas preenchidas | Referenciada só por constantes de Go; conferir à mão. |
| `integrations.source_type` | 0 leituras e 2 escritas no Go; 1 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `integrations.display_name` | 0 leituras e 0 escritas no Go; 1 linhas preenchidas | Referenciada só por constantes de Go; conferir à mão. |
| `integrations.token_ref` | 0 leituras e 0 escritas no Go; 0 linhas preenchidas | Sem uso hoje; pode estar reservada para as ISSUE-086 a 091. |
| `integrations.revoked_at` | 0 leituras e 0 escritas no Go; 0 linhas preenchidas | Sem uso hoje; pode estar reservada para as ISSUE-086 a 091. |
| `integrations.created_at` | 0 leituras e 2 escritas no Go; 1 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `integrations.diagnostic_enabled` | 0 leituras e 0 escritas no Go; 1 linhas preenchidas | Sem uso hoje; pode estar reservada para as ISSUE-086 a 091. |
| `outbox.operation` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `outbox.delivery_key` | 0 leituras e 1 escritas no Go; 0 linhas preenchidas | Recebe dados e nenhuma consulta lê. |
| `relay_sent_archive.sent_count` | 0 leituras e 0 escritas no Go; 0 linhas preenchidas | Só escrita por gatilho; ninguém lê. Mostrar a contagem ou remover. |
| `sessions.created_at` | 0 leituras e 2 escritas no Go; 0 linhas preenchidas | Só escrita no Go; confirmar se alguma tela a usa. |
| `sessions.persistent` | 0 leituras e 2 escritas no Go; 0 linhas preenchidas | Só escrita no Go; confirmar se alguma tela a usa. |

## 5. O filtro proposto e quanto economiza

Estimativas por escuta, sobre os 2.382 bytes medidos. São **estimativas de cálculo**, não medições após a mudança.

| Nível | O que muda | Economia por escuta | Resultado |
| --- | --- | ---: | ---: |
| **1. Sem mudar nada que o usuário veja** | Parar de gravar a revisão automática do recibo aceito (R1); remover as 3 colunas legadas (R6) e as 5 mortas; horário do recibo como inteiro (R2); fundir `closed_at` e `resolved_at` (R3). | **−247 B** (−10%) | 2.135 B |
| **2. Moderado** | Nível 1, mais: gravar o resultado padrão só quando fugir dele (R5); trocar `listen_export_order` por uma coluna de sequência (R4, líquido de 8 B); remover os 3 índices redundantes de `listens` (aprox. 33 B a 300 mil escutas). | **−430 B** (−18%) | 1.952 B |
| **3. A decidir (não somado)** | Rever os quatro índices maiores do recibo (até **328 B**, exige ler os planos das páginas de saúde); compactar o `original_snapshot` e o digest de recibos antigos já aceitos (até **341 B**, muda o que a página «por que esta escuta está aqui» consegue explicar). | até −669 B | depende |

**Por que o nível 3 não entra na soma.** Os dois itens trocam custo por funcionalidade: o primeiro pode deixar uma página de saúde lenta; o segundo apaga a prova de como a escuta chegou. São decisões de produto, e não de limpeza.

**Ordem sugerida.** O nível 1 é seguro e cabe na ISSUE-195 (a consolidação das migrações). O nível 2 pede ajuste em Go e planos de consulta antes e depois. O nível 3 só com a sua decisão.

## 6. Como filtrar e refazer

Os arquivos ficam em `backlog/evidencias/db-auditoria/`:

- **`colunas.csv`**: uma linha por coluna, com `classe`, `leituras_go`, `escritas_go`, `usada_pelo_banco`, `linhas_nao_nulas`, `valores_distintos`, `tamanho_medio_bytes`, `repete_outra_coluna`, `veredito` e `motivo`. Abre em qualquer planilha: filtre por `veredito` ou por `tabela`.
- **`audit_db_usage.py`**: mede um banco (somente leitura) e escreve o JSON de evidências.
- **`classify_columns.py`**: transforma o JSON no CSV. As decisões manuais e seus motivos estão no topo do arquivo, para você contestar uma por uma.

```
python3 audit_db_usage.py  BANCO.db  RAIZ_DO_REPOSITORIO  evidencias.json
python3 classify_columns.py BANCO.db evidencias.json colunas.csv
```

Rode sobre uma **cópia** do banco em uso para ver os dados reais. Num banco com poucos recibos, as colunas de recebimento aparecerão vazias, como na seção 1.

## 7. Decisões que preciso do usuário

1. **Nível 1** (seção 5): aprovar como parte da ISSUE-195?
2. **`ingest_revisions`:** parar de gravar a revisão automática de recibo aceito, mantendo só as decisões de usuário e de fusões? (R1, 194 B por escuta.)
3. **Recibo aceito:** por quanto tempo guardar o conteúdo original (`original_snapshot` e digest)? Sempre, ou compactar depois de N dias? Isso é o maior item do nível 3.
4. **Índices do recibo:** autoriza medir os planos das páginas de saúde para ver quais dos quatro maiores podem sair?
5. **As 20 colunas da seção 4.4:** em especial `integrations.token_ref`, `revoked_at` e `diagnostic_enabled`, que podem estar reservadas às ISSUE-086 a 091.
6. **Tabelas só de escrita:** `relay_sent_archive` (mostrar a contagem diária ou remover) e as colunas só de escrita de `sessions`, `outbox`, `ingest_observations`.
