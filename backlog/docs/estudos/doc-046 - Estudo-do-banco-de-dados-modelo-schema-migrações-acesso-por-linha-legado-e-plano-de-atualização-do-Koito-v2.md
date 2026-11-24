---
id: doc-046
title: >-
  Estudo do banco de dados: modelo, schema, migrações, acesso por linha, legado
  e plano de atualização do Koito v2
type: other
created_date: '2026-11-23T21:42:00Z'
updated_date: '2026-11-23T21:42:00Z'
tags:
  - estudo
  - banco-de-dados
---
# Estudo do banco de dados do TuneTrail: modelo, schema, migrações, acesso por linha, legado e plano de atualização do Koito v2

Estudo de 23/11/2026, sobre a `main` na v0.74 (`216ccb0`), **schema 23**. Fontes: as 23 migrações em `db/migrations_sqlite/`, o banco real recém-criado pelo binário, o `DB_ALTERATIONS.md`, o estudo dos metadados da ISSUE-055 (`backlog/evidencias/issue-055/ESTUDO.md`), o inventário da ISSUE-194 (`backlog/evidencias/issue-194/COLUNAS.md`) e medições feitas para este estudo. **Nada foi alterado no código nem no banco**: o estudo só descreve e propõe.

Convenção do texto: o que foi **medido** ou **lido no código** está dito como tal. O que é **inferência** ou **proposta** vem marcado.

## 0. Resumo

- **O que existe.** O banco é um SQLite com 40 tabelas de dados (mais a `goose_db_version`), 279 colunas, 55 chaves estrangeiras, 55 índices, 55 gatilhos e 3 views. O Koito v2 tinha 12 tabelas, 49 colunas, 12 chaves estrangeiras, 13 índices, 1 gatilho e 3 views (seção 2.0). As migrações são embutidas no binário (goose) e rodam uma vez, em ordem, no primeiro start.
- **Como o acesso é protegido.** O SQLite **não tem segurança por linha** (não existe `CREATE POLICY`). A separação entre perfis é feita em camadas: estrutura (`profile_id` e chaves estrangeiras), gatilhos, serviço (`AuthorizeAction`) e a porta HTTP (404 uniforme para perfil protegido). A seção 4 mapeia cada regra e aponta a lacuna: nada no banco impede uma consulta nova de esquecer o `profile_id`.
- **O legado é grande.** Dos 55 gatilhos, **16 existem só por causa de código que já saiu** (sombra, resolvedor v9, ponte entre `tracks` e `occurrences`, política de relay antiga). Duas tabelas **não têm nenhum escritor** no código de produção, duas só recebem escrita e nunca são lidas, e um par de tabelas do relay é uma relação 1:1 mantida por cinco gatilhos. Há também 10 colunas sem nenhuma referência no Go e 3 colunas constantes. A seção 5 lista tudo com a evidência.
- **A simplificação é grande, mas em degraus.** Pacote A (sem mudar comportamento): **40 → 36 tabelas** e **55 → 46 gatilhos**. Pacote B (com ajuste em Go): **→ 33 tabelas** e **→ 40 gatilhos**. Pacote C (refatoração maior): **→ 32 tabelas** e **→ 33 gatilhos**.
- **A atualização do Koito v2 foi medida.** Um banco v2 sintético com **300 mil escutas (35 MB)** chegou ao schema 23 em **11,5 s** (inclui subir o servidor), sem perder nada, e o banco final tem **125 MB**. O backup automático tem 35 MB. É preciso reservar cerca de **5,5 vezes** o tamanho original em disco.
- **Os metadados da 055 cabem de forma aditiva.** Duas colunas e quatro tabelas novas, todas de catálogo compartilhado. O ideal é fazê-los **antes** de consolidar as migrações (ISSUE-195), para que a produção atravesse o caminho uma única vez.

## 1. Modelo de dados (Data model)

O modelo responde a três perguntas: **que informação existe** (entidades), **como as entidades se ligam** (relacionamentos) e **como os dados se organizam** (estrutura).

### 1.1 Entidades: que informação existe

| Domínio | Entidades | Para que servem |
| --- | --- | --- |
| **Catálogo** | `artists`, `releases`, `tracks`¹, `recordings`, `occurrences`, mais aliases e ligações | O que se ouve. Compartilhado por todos os perfis. |
| **Pessoas e acesso** | `users`, `sessions`, `account_tokens`, `listening_profiles`, `api_keys` | Quem usa o sistema e o que cada um pode fazer. Uma conta tem um perfil. |
| **Escutas** | `listens`, `event_origins`, `listen_export_order` | Cada vez que alguém ouviu uma faixa, e de onde veio o registro. |
| **Recebimento** | `integrations`, `ingest_events` (o recibo), detalhes, observações, revisões, descartes, caches de saúde | Como uma submissão vira escuta e por quê. |
| **Relay** | `profile_relays`, `relay_destinations`, `relay_allowlist`, `outbox`, e tabelas auxiliares | Reenvio das escutas a outro servidor compatível com ListenBrainz. |
| **Restauração** | `archive_*`, `recording_moves` | Restaurar um histórico exportado e desfazer fusões de gravações. |

¹ `tracks` é a faixa do Koito v2, mantida como ponte para o modelo novo (ver seção 5).

**Três ideias do catálogo que explicam o resto.** Uma *gravação* (`recordings`) é a música em si. Uma *ocorrência* (`occurrences`) é essa música numa versão específica de um lançamento, ou em nenhum. Uma *escuta* (`listens`) é um evento: alguém ouviu uma ocorrência num segundo exato. No Koito v2 só existia a faixa; a migração 8 separou as três ideias e manteve os números: cada `tracks.id` virou o `occurrences.id` do mesmo valor.

### 1.2 Relacionamentos: como as entidades se ligam

```
users 1 ── 1 listening_profiles 1 ── * api_keys
   │                 │
   │                 ├── 1 profile_relays 1 ── 1 relay_destinations ── * outbox ── * outbox_details
   │                 │                         relay_allowlist (por perfil)
   │                 ├── * integrations ── * ingest_events ── * ingest_revisions / ingest_observations
   │                 │                          │    │            ingest_event_details
   │                 │                          │    └──> occurrences, recordings
   │                 │                          └──> listens (event_id)
   │                 ├── * listens ──> occurrences ──> recordings
   │                 │                      │              (musicbrainz_id)
   │                 │                      ├──> releases ── * tracks (ponte legada)
   │                 │                      └── * occurrence_credits ──> artists
   │                 ├── ingest_discards, ingest_inbox_daily, ingest_source_summary
   │                 └── archive_catalog, archive_events, listen_export_order
   └── 1 account_tokens, * sessions

artists ── * artist_aliases     releases ── * release_aliases     tracks ── * track_aliases
artists ── * artist_releases ── releases        artists ── * artist_tracks ── tracks
```

| Relação | Cardinalidade | Regra que o banco garante |
| --- | --- | --- |
| conta ↔ perfil | 1 : 1 | `listening_profiles.user_id` é único, e um gatilho o torna obrigatório e imutável (a coluna em si aceita `NULL`). |
| perfil → escutas | 1 : N | `listens.profile_id`, com `ON DELETE CASCADE`. Apagar o perfil apaga o histórico. |
| perfil → chaves de API | 1 : N | `api_keys.profile_id`, imutável. `user_id` ficou como atribuição, sem chave estrangeira. |
| escuta → ocorrência → gravação | N : 1 : 1 | `listens.track_id` aponta para `occurrences(id)` (o nome da coluna é histórico). |
| ocorrência → lançamento | N : 0..1 | Uma ocorrência pode não ter lançamento. |
| recibo → escuta | 1 : 0..1 | Chave estrangeira composta `(profile_id, event_id)`; o `event_id` existe exatamente nos estados `accepted` e `duplicate`. |
| perfil → relay | 1 : 0..1 | `profile_relays` (URL e token) e `relay_destinations` (estado) são **1:1** e mantidos em sincronia por gatilhos. |

### 1.3 Estrutura: como os dados se organizam

- **Identificadores.** Inteiros autoincrementais para catálogo, contas e perfis. A escuta tem um `event_id` de texto (128 bits em hexadecimal, gerado pelo banco). O recibo tem um `receipt_id` próprio.
- **Tempo.** Sempre inteiro em segundos Unix (UTC). A precisão e a semântica de um tempo informado pela fonte ficam em colunas do recibo.
- **Enumerações.** Texto com `CHECK ... IN (...)` (por exemplo `users.role`, `users.status`, `ingest_events.state`). A única enumeração com tabela própria é `event_origins`.
- **Booleanos.** Inteiro 0 ou 1 com `CHECK`.
- **JSON.** Texto com `CHECK (json_valid(...))`, usado em evidências, códigos de motivo e cargas de diagnóstico.
- **Colunas geradas.** Quatro virtuais em `ingest_events` (`health_source`, `health_client`, `inbox_key`, `inbox_metadata`), que existem para contar a Inbox e a saúde sem ler cargas e não ocupam espaço na linha.
- **Exclusão.** Quase tudo é `ON DELETE CASCADE` a partir do perfil. O catálogo é compartilhado: apagar um perfil nunca apaga artistas, lançamentos ou gravações.
- **Dados pessoais.** Ficam em `users` (hash da senha), `sessions`, `account_tokens` (só o hash do segredo), `api_keys` (a chave em si) e `profile_relays` (o token do destino).

## 2. Esquema do banco (Database schema)

Esta seção resume as regras. A **referência completa**, tabela por tabela, com todas as colunas, tipos, padrões, chaves e restrições, está nos **apêndices A a D**, gerados do banco real.

### 2.0 Do Koito v2 ao schema 23, em números

Contagem feita no banco do Koito v2 (migrações 001 e 002 aplicadas) e no banco atual criado pelo binário. As tabelas de dados não incluem a `goose_db_version`, que existe nos dois (4 colunas). Colunas são as visíveis (`pragma table_info`); o schema atual tem ainda 4 colunas geradas em `ingest_events`.

| | Koito v2 | Schema 23 | Fator |
| --- | ---: | ---: | ---: |
| Tabelas de dados | 12 | 40 | 3,3× |
| Colunas | 49 | 279 | 5,7× |
| Chaves estrangeiras | 12 | 55 | 4,6× |
| Índices (com SQL) | 13 | 55 | 4,2× |
| Gatilhos | 1 | 55 | 55× |
| Views | 3 | 3 | 1× |

Só a tabela `ingest_events` (o recibo) tem **59 das 279 colunas** (21%) e 7 das 55 chaves estrangeiras. O crescimento vem quase todo do recebimento de submissões e do relay, que não existiam no Koito v2.

### 2.1 Tabelas e colunas: que dados são guardados

| Domínio | Tabela | Colunas | FKs | Linhas no banco de desenvolvimento¹ | Leituras / escritas no Go² | Classificação |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| Catálogo | `artists` | 4 | 0 | 24 | 22 / 11 | viva |
| Catálogo | `artist_aliases` | 4 | 1 | 31 | 24 / 8 | viva |
| Catálogo | `releases` | 5 | 0 | 23 | 18 / 14 | viva |
| Catálogo | `release_aliases` | 4 | 1 | 27 | 11 / 7 | viva |
| Catálogo | `tracks` | 4 | 1 | 90 | 37 / 8 | ponte legada |
| Catálogo | `track_aliases` | 4 | 1 | 90 | 17 / 7 | viva |
| Catálogo | `artist_releases` | 3 | 2 | 34 | 10 / 9 | viva |
| Catálogo | `artist_tracks` | 3 | 2 | 101 | 38 / 9 | viva |
| Catálogo | `recordings` | 3 | 0 | 90 | 14 / 5 | viva |
| Catálogo | `occurrences` | 8 | 2 | 90 | 38 / 9 | viva |
| Catálogo | `occurrence_credits` | 4 | 2 | 0 | 7 / 2 | viva |
| Pessoas e acesso | `users` | 5 | 0 | 2 | 15 / 9 | viva |
| Pessoas e acesso | `sessions` | 5 | 1 | 14 | 3 / 6 | viva |
| Pessoas e acesso | `account_tokens` | 4 | 1 | 0 | 0 / 4 | viva |
| Pessoas e acesso | `listening_profiles` | 9 | 1 | 2 | 26 / 7 | viva |
| Pessoas e acesso | `api_keys` | 6 | 1 | 2 | 5 / 7 | viva |
| Escutas | `listens` | 9 | 3 | 80 | 74 / 9 | viva |
| Escutas | `event_origins` | 2 | 0 | 9 | 0 / 1 | simplificável |
| Escutas | `listen_export_order` | 2 | 1 | 80 | 2 / 0 | redundante? |
| Recebimento (ingestão) | `integrations` | 9 | 1 | 0 | 2 / 3 | viva |
| Recebimento (ingestão) | `ingest_events` | 59 | 7 | 0 | 37 / 13 | viva |
| Recebimento (ingestão) | `ingest_event_details` | 8 | 2 | 0 | 0 / 0 | morta |
| Recebimento (ingestão) | `ingest_observations` | 10 | 2 | 0 | 1 / 1 | viva |
| Recebimento (ingestão) | `ingest_revisions` | 9 | 2 | 0 | 0 / 5 | só escrita |
| Recebimento (ingestão) | `ingest_discards` | 8 | 1 | 0 | 3 / 3 | viva |
| Recebimento (ingestão) | `ingest_source_summary` | 6 | 2 | 0 | 1 / 0 | cache derivado |
| Recebimento (ingestão) | `ingest_inbox_daily` | 7 | 1 | 0 | 1 / 0 | cache derivado |
| Recebimento (ingestão) | `profile_ingest_review` | 2 | 1 | 0 | 1 / 1 | simplificável |
| Relay (envio a outro servidor) | `profile_relays` | 3 | 1 | 1 | 13 / 4 | fundível |
| Relay (envio a outro servidor) | `relay_destinations` | 12 | 1 | 1 | 23 / 18 | viva |
| Relay (envio a outro servidor) | `relay_allowlist` | 6 | 1 | 0 | 2 / 8 | viva |
| Relay (envio a outro servidor) | `outbox` | 18 | 5 | 0 | 12 / 12 | viva |
| Relay (envio a outro servidor) | `outbox_details` | 6 | 2 | 0 | 0 / 0 | morta |
| Relay (envio a outro servidor) | `relay_sent_archive` | 3 | 1 | 0 | 0 / 2 | só escrita |
| Relay (envio a outro servidor) | `relay_delivery_gate` | 2 | 0 | 1 | 1 / 1 | legada |
| Relay (envio a outro servidor) | `legacy_relay_binding` | 2 | 1 | 1 | 2 / 1 | legada |
| Restauração e desfazer | `archive_identity` | 2 | 0 | 1 | 2 / 0 | viva |
| Restauração e desfazer | `archive_catalog` | 6 | 1 | 0 | 2 / 1 | viva |
| Restauração e desfazer | `archive_events` | 6 | 2 | 0 | 3 / 1 | viva |
| Restauração e desfazer | `recording_moves` | 7 | 1 | 0 | 2 / 6 | viva |
| Infraestrutura | `goose_db_version` | 4 | 0 | 23 | 1 / 1 | infra |

¹ Contagem de linhas na cópia do banco de desenvolvimento do usuário (schema 22), só números. A coluna **Colunas** conta as visíveis (`pragma table_info`), então `ingest_events` mostra 59 aqui e 63 linhas no apêndice A, que também lista as 4 colunas geradas. O banco de produção do Koito v2 não terá nada nas tabelas de recebimento e relay até começar a receber submissões. ² Ocorrências do nome da tabela em SQL do Go de produção (sem testes e sem a lista de layout do `ingest_schema.go`): leituras são `FROM` e `JOIN`, escritas são `INSERT`, `UPDATE` e `DELETE` (um `UPDATE ... RETURNING` conta como escrita, como em `account_tokens`).

### 2.2 Tipos de dados: que valores são permitidos

SQLite tem tipagem dinâmica, então o tipo declarado sozinho não protege. O TuneTrail compensa com restrições:

| Intenção | Tipo declarado | Restrição que a protege |
| --- | --- | --- |
| Identificador | `INTEGER PRIMARY KEY` | Único por definição. `AUTOINCREMENT` onde o ID nunca pode ser reaproveitado (`api_keys`, `listen_export_order`). |
| Tempo | `INTEGER` | Segundos Unix. |
| Papel e status da conta | `TEXT` | `role IN ('admin','user')` e `status IN ('active','pending')`. |
| Estado do recibo | `TEXT` | Sete estados fixos, com `CHECK` ligando `event_id` ao estado. |
| Duração tocada (023) | `INTEGER` | `CHECK (duration_played IS NULL OR (typeof(...)='integer' AND ... >= 0))`. |
| Cor do avatar (022) | `TEXT` | `CHECK` de formato `#rrggbb` minúsculo; a paleta de 12 cores é imposta pela aplicação. |
| Identificador de URL (`slug`) | `TEXT` | Só minúsculas, números e hífen, 1 a 64 caracteres, sem hífen nas pontas, e uma lista de nomes reservados. |
| JSON | `TEXT` | `json_valid(...)`, e `json_type(...)='array'` onde se espera lista. |
| Booleano | `INTEGER` | `IN (0,1)`. |

### 2.3 Restrições: regras que protegem a integridade

**Chaves estrangeiras.** O banco abre com `foreign_keys=ON`. A política é: dados do perfil são apagados em cascata com o perfil; dados do catálogo nunca são apagados por um perfil; a origem de uma escuta é `ON DELETE RESTRICT`.

**Gatilhos.** São 55, em sete categorias. Eles aplicam no próprio banco regras que o código de aplicação não pode pular:

| Categoria | Gatilhos |
| --- | ---: |
| Recebimento | 13 |
| Relay | 12 |
| Ponte tracks↔occurrences | 11 |
| Escutas | 6 |
| Contas e perfis | 5 |
| Cache derivado | 5 |
| Catálogo | 3 |
| **Total** | **55** |

A lista completa, com a finalidade de cada um e a marca de legado, está no apêndice C. Dos 55, **16 são legado**, e esse é o centro da seção 5.

**Índices.** São 55. Os mais pesados estão em `listens` (seção 5.4 mostra o custo medido). A lista completa está no apêndice B.

## 3. Migrações (Database migrations)

### 3.1 Como funcionam

- As migrações ficam em `db/migrations_sqlite/` e são **embutidas no binário** (`embed`), executadas pelo goose. Cada uma roda **na sua própria transação**.
- **Não há volta.** O `Down` de toda migração estrutural **aborta de propósito** com uma tabela temporária de proteção (coluna `restore_verified_backup`). O único caminho de volta é restaurar o backup que o upgrade faz antes.
- **Backup automático.** Antes de migrar, o servidor tira um instantâneo consistente (inclui o WAL confirmado). O nome muda conforme a versão de partida (`.pre-tunetrail-*`, `.pre-accounts-*`, `.pre-ingest-*` e assim por diante), e a permissão é `0600`.
- **Validação.** Depois de migrar, o servidor confere integridade, chaves estrangeiras, cardinalidade conta/perfil, gatilhos obrigatórios e o **layout** inteiro: o `sqlite_master` do banco é comparado com o DDL embutido (`validateSchemaLayout`), então uma versão forjada não habilita um schema incompleto.
- **Travas.** Um arquivo de trava de banco e um de instância impedem duas instâncias de migrar ou escrever ao mesmo tempo.
- **Migrações que reconstroem tabelas** (3, 8, 19) copiam os dados para tabelas temporárias e **comparam contagens e conteúdo** antes de confirmar. Se algo difere, a migração inteira volta.

### 3.2 Histórico de versões (Version history)

| Versão | Arquivo | Mudança | Efeito nos dados |
| --- | --- | --- | --- |
| 1 | `001_initial_schema.sql` | Catálogo, contas, escutas e chaves do Koito. | Base. |
| 2 | `002_unique_mbid.sql` | `musicbrainz_id` único em artistas, lançamentos e faixas. | Estado de partida do Koito 0.3. |
| 3 | `003_listening_profiles.sql` | Perfis de escuta. Reconstrói `api_keys` e `listens` com `profile_id`. | Um perfil por usuário; histórico e chaves passam para o perfil. |
| 4 | `004_ingest_infrastructure.sql` | Recibos, detalhes, revisões e fila de entrega. | Tabelas vazias. Cria dois índices que bancos antigos do Koito não têm. |
| 5 | `005_accounts_profiles.sql` | Dono do perfil, `status` da conta, guardas do último administrador. | Liga perfis a contas (1:1). Remove a ponte `legacy_user_profiles`. |
| 6 | `006_active_admin_guards.sql` | O último administrador conta só se puder entrar. | Só regra. |
| 7 | `007_account_tokens.sql` | Tokens de ativação e recuperação. | Tabela vazia. |
| 8 | `008_recording_identity.sql` | Gravações, ocorrências, origens; reconstrói `listens` com `event_id` e `origin`. | **A mais pesada**: copia e confere toda a tabela de escutas. |
| 9 | `009_ingest_activation.sql` | Modo do recibo e chave por perfil. | Recibos antigos ficam em sombra. |
| 10 | `010_occurrence_ingest.sql` | Resolvedor por ocorrência e créditos. | Reparos de recibos vinculados. |
| 11 | `011_occurrence_writer_guards.sql` | Gatilhos que barram o escritor antigo. | Só regra. |
| 12 | `012_relay_delivery.sql` | Relay durável: destinos, lista de exceções, arquivo de enviados. | Cria o destino de cada relay já configurado. |
| 13 | `013_ingest_retry.sql` | Estacionamento de recibos que falham. | Colunas novas. |
| 14 | `014_occurrence_archive.sql` | Restauração de histórico e desfazer de fusões. | Tabelas vazias. |
| 15 | `015_archive_export_boundary.sql` | Sequência de exportação que nunca se repete. | Numera as escutas existentes. |
| 16 | `016_ingest_review.sql` | `review_distinct` e índices da fila de revisão. | Coluna nova. |
| 17 | `017_ingest_source_summary.sql` | Resumo do último recibo por origem e cliente. | Cache novo. |
| 18 | `018_ingest_reprocess.sql` | Orçamento de claims e autoria do reprocessamento. | Colunas novas. |
| 19 | `019_ingest_inbox.sql` | Reconstrói `api_keys` (IDs nunca reutilizados); contagens diárias. | Confere a tabela de chaves. |
| 20 | `020_ingest_discards.sql` | Trilha de auditoria de descartes. | Tabela vazia. |
| 21 | `021_occurrence_only.sql` | Retira a sombra, as chaves por perfil e o índice legado. | **Recusa** rodar se houver recibos de sombra sem decisão. |
| 22 | `022_avatar_color.sql` | Cor de fundo do avatar (ISSUE-202). | Coluna nula; perfis existentes ficam sem cor. |
| 23 | `023_duration_played.sql` | Tempo realmente tocado por escuta (ISSUE-190). | Coluna nula; escutas antigas continuam usando a duração da faixa. |

As migrações 22 e 23 são as mais simples do histórico: um `ALTER TABLE ... ADD COLUMN` com `CHECK`, sem reescrever a tabela (o `ALTER` da 023 levou 85 ms em 300 mil escutas, sem crescer o número de páginas).

### 3.3 Reprodutibilidade (Reproducibility)

- **Mesmo SQL em todo ambiente.** O binário carrega as migrações embutidas; não há script externo nem passo manual.
- **Prova por teste.** A suíte tem testes de bootstrap v2 → atual (200 escutas, contagem e digest por perfil), de cada degrau relevante, de backup e restauração, e jornadas do e2e (`migration`, `ingest-migration`, `standalone`).
- **Scripts de ensaio.** `scripts/validate_migration.py` (v2 sintético e `--source-v2` com cópia real), `validate_accounts_migration.py` (v3/v4 → atual), `validate_ingest_migration.py`, e `validate_multiuser_browser.py` (com Playwright, fora do `make check`). A versão esperada vem dos nomes dos arquivos de migração, então acompanha migrações novas sem edição manual.
- **Lacuna.** Ainda não existe o teste de **equivalência** entre "aplicar 001 a 023 em ordem" e um futuro arquivo único. É a prova exigida pela ISSUE-195.

### 3.4 Mudanças seguras (Safe changes)

Checklist de uma migração nova, destilado das migrações 22 e 23:

1. Uma migração **aditiva** (coluna nula ou tabela nova) é o caso barato. Reconstruir uma tabela grande como `listens` é caro e exige comparação de contagens.
2. Criar o arquivo `0NN_nome.sql` com o `Down` de proteção (`restore_verified_backup`).
3. Subir `currentSchemaVersion` em `internal/db/sqlite/bootstrap.go` e acrescentar o arquivo à lista de `ingest_schema.go`, que valida o layout.
4. Atualizar as colunas lidas e gravadas (`profileColumns` e afins) e todos os lugares que escrevem a tabela.
5. Testes que usam a loja em um schema histórico (por exemplo schema 5) precisam escrever por SQL, porque a loja já consulta as colunas novas. O teste que espera a recusa do `Down` da última migração continua válido porque o texto da recusa é o mesmo.
6. Teste de migração N−1 → N com backup, e teste de ida e volta da loja e do `CHECK`.
7. Documentar (`DB_ALTERATIONS.md`, guia de operação e changelog) e regenerar a referência da interface se o HTML mudar.
8. **Se a migração entrar antes do primeiro upgrade de produção, ela também entra no arquivo único da ISSUE-195.**

## 4. Políticas de acesso por linha (Row-level security policies)

### 4.1 O que existe de verdade

SQLite **não implementa segurança por linha**: não há `CREATE POLICY`, não há papéis de banco, e todo processo que abre o arquivo enxerga todas as linhas. O isolamento entre usuários é, portanto, uma **propriedade do código e da estrutura**, em cinco camadas:

| Camada | O que faz | Onde está |
| --- | --- | --- |
| **1. Estrutura** | Toda linha de dado pessoal carrega `profile_id` e é apagada em cascata com o perfil. Chaves compostas, como `(profile_id, event_id)`, impedem que um recibo aponte para a escuta de **outro** perfil. | Tabelas `listens`, `ingest_*`, `outbox`, `integrations`, `api_keys`. |
| **2. Gatilhos** | Regras que não dependem de o código se comportar: perfil sempre com dono e dono imutável, chave não troca de perfil, último administrador protegido. | Apêndice C. |
| **3. Serviço** | `AuthorizeAction` é a política única para HTML, JSON e serviços. Ação desconhecida é sempre negada. O ator vem da sessão, e o recurso é carregado pelo servidor, nunca lido do pedido. | `internal/admin/auth.go`. |
| **4. Porta HTTP** | `CanReadListening` decide se um perfil protegido pode ser lido. A negativa devolve o **mesmo 404** de um perfil que não existe, com os mesmos cabeçalhos, inclusive em requisição condicional e prefetch. | `internal/adminweb/listening.go`, `engine/middleware/profiles.go`. |
| **5. Consultas** | As leituras de histórico filtram por `profile_id` (o contexto carrega o perfil por `db.WithProfile`), e a leitura de imagens só inclui perfis públicos ou o próprio. **Não há auditoria estática exaustiva** disso (ver 4.4). | `internal/db/sqlite/*`. |

### 4.2 Matriz de políticas (resultado de `AuthorizeAction`)

| Ação | Visitante | Usuário comum | Dono do perfil | Administrador (root) |
| --- | --- | --- | --- | --- |
| `profile.read-listening`, perfil **público** | permitido | permitido | permitido | permitido |
| `profile.read-listening`, perfil **protegido** (`require_login=1`) | negado | negado | permitido | permitido |
| `profile.manage` (perfil, avatar, chaves, histórico, relay) | não autenticado | negado em perfil alheio | permitido | permitido em qualquer perfil |
| `account.self` (conta própria) | não autenticado | permitido | permitido | permitido |
| `accounts.issue-token`, `accounts.change-role` | não autenticado | negado | negado | permitido (não em si mesmo, para o token) |
| `accounts.manage`, `catalog.manage`, `server.manage` | não autenticado | negado | negado | permitido |

Além da tabela: só conta **ativa** age; uma sessão expirada ou rebaixada perde a permissão imediatamente, porque o serviço **resolve a sessão de novo** na fronteira (`AuthorizeService`).

### 4.3 Política por tabela: quem lê e escreve quais linhas

| Dados | Quem lê | Quem escreve | Observação |
| --- | --- | --- | --- |
| Catálogo (`artists`, `releases`, `recordings`, `occurrences`, aliases) | Pelas páginas de um perfil, e só com a visibilidade desse perfil | Ingestão e root (`catalog.manage`) | Compartilhado entre perfis. Não contém dado pessoal. |
| `listens` | Dono e root sempre; os demais só se o perfil for público | Ingestão, importação, dono e root | A regra de leitura é aplicada na porta HTTP e nas consultas. |
| `ingest_*`, `integrations`, `outbox` | Root, e o dono nas páginas de saúde do próprio perfil | Servidor | Nunca expostos a terceiros. |
| `api_keys`, `profile_relays` | Dono e root | Dono e root | Contêm segredos. |
| `users`, `sessions`, `account_tokens` | Só o próprio servidor | Servidor | Nunca saem pela API. Do token só se guarda o hash. |
| `relay_allowlist`, configuração do servidor | Root | Root | Política de rede do relay. |

### 4.4 Lacunas e propostas

- **[Lacuna] O banco não impede uma leitura sem filtro.** Uma consulta nova que esqueça o `profile_id` vaza dados entre perfis, e nenhum gatilho barra um `SELECT`. A mitigação atual é de teste: a matriz da ISSUE-164 validou HTML, JSON e serviços para visitante, comum, dono, terceiro, root e sessão expirada ou rebaixada, sem achar desvio.
- **[Proposta, a decidir] Teste estático de consultas.** Um teste que percorra o SQL do Go de produção e falhe quando uma consulta a uma tabela com `profile_id` não filtrar por ele nem estiver numa lista explícita de exceções (catálogo, administração). Não substitui segurança por linha, mas torna a lacuna visível a cada mudança.
- **[Proposta, a decidir] Visões por perfil.** Visões que já embutam o filtro do perfil do contexto reduziriam o risco de esquecimento, ao custo de reescrever as consultas. Só vale a pena se o teste estático mostrar muitas exceções.
- **[Observação] A regra de proteção fica na aplicação.** `require_login` é uma coluna; quem a respeita é o código. Se o TuneTrail passar a ter outro processo lendo o arquivo (relatórios, backup lógico), esse processo precisa reaplicar a regra.
- **[Fora de questão] PostgreSQL.** Segurança por linha nativa exigiria outro banco. O TuneTrail é SQLite por decisão, então este estudo não propõe a troca.

## 5. Legado e simplificações

Pedido do usuário em 23/11/2026: registrar o que ficou de legado e deixou de ser usado, porque há muitas tabelas que podem ser simplificadas.

### 5.1 Como foi medido

1. **Uso no código** (medido): para cada tabela e cada coluna, contagem de referências no Go de produção, sem testes e sem a lista de layout. Uma coluna **sem nenhuma referência** é candidata a morta.
2. **Uso nos dados** (medido): contagem de linhas no banco de desenvolvimento do usuário.
3. **Custo em espaço** (medido): `dbstat` no banco resultante de um Koito v2 sintético de 300 mil escutas.
4. **Inventário prévio** da ISSUE-194 (`COLUNAS.md`), conferido e ampliado.

**Auditoria por coluna.** O doc-047 refaz este levantamento coluna a coluna (283), num banco populado com 5.000 escutas pela API, e mede o custo por escuta e as repetições. Ele prevalece sobre as estimativas abaixo onde divergirem.

**Cuidados.** (a) "Sem referência" não é "sem valor": colunas podem ter sido criadas para uma issue futura (as fontes das ISSUE-086 a 091 usam `integrations`). (b) A ausência de linhas no banco de desenvolvimento não prova que a tabela é inútil, só que ele nunca recebeu submissões pela API. (c) Cada remoção abaixo exige ler o plano de consulta e os testes antes de executar.

### 5.2 Tabelas

**Mortas: nenhum escritor no código de produção (medido).**

| Tabela | Evidência | Proposta |
| --- | --- | --- |
| `ingest_event_details` | Só aparece na lista de retenção (`ingest_retention.go`); nada insere nela. | Remover, junto da entrada de retenção e de `integrations.details_retention_days`. |
| `outbox_details` | Idem: só aparece na lista de retenção. | Remover, com a mesma entrada de retenção. |

**Só escrita: recebe dados e ninguém lê (medido).**

| Tabela | Evidência | Proposta |
| --- | --- | --- |
| `relay_sent_archive` | Alimentada por gatilho (`trg_relay_archive_sent`) e apagada pela retenção; **nenhuma leitura**. A coluna `sent_count` não é citada no Go. | Decidir: mostrar a contagem diária na saúde do relay ou remover tabela e gatilho. |
| `ingest_revisions` | Cinco `INSERT` e **nenhuma leitura** no Go de produção. É a trilha de auditoria das decisões. | Decidir: expor na página "por que esta escuta está aqui" (ISSUE-083) ou aceitar que é só auditoria por SQL. Não remover antes. |

**Legado de transição (código que já saiu).**

| Tabela | O que é | Proposta |
| --- | --- | --- |
| `legacy_relay_binding` | Uma linha que marca que o relay **legado por variável de ambiente** (`ENABLE_LBZ_RELAY`, `LBZ_RELAY_URL`, `LBZ_RELAY_TOKEN`) já foi associado uma vez a um perfil. Lida só em `initializeLegacyRelay`. | Sai **junto com** esse bootstrap por variáveis. A ISSUE-197 o mantinha até a 164, que já foi publicada: é decisão sua (seção 8). |
| `relay_delivery_gate` | Uma linha com a política `legacy` ou `durable`, que só vai de uma para a outra. Hoje é sempre `durable` depois do primeiro start. | Remover, junto de `ingest_events.delivery_policy` e dos gatilhos `trg_relay_policy_one_way` e `trg_relay_active_policy`. |
| `tracks` (ponte) | A faixa do Koito v2. Cada `tracks.id` é igual ao `occurrences.id`, e 11 gatilhos mantêm as duas tabelas em sincronia. **Ainda é lida em 37 pontos e escrita em 8** do Go. | Pacote C: maior ganho estrutural, maior custo. Ver 5.5. |

**Simplificáveis.**

| Tabela | Por que | Proposta |
| --- | --- | --- |
| `profile_ingest_review` | Uma linha por perfil só quando o padrão (revisão ligada) é mudado; lida como `coalesce(..., 1)`. | Virar uma coluna `listening_profiles.ingest_review INTEGER NOT NULL DEFAULT 1`. |
| `event_origins` | Tabela de 9 valores, só consultada pela chave estrangeira e por um `INSERT` de `manual`. Nenhuma leitura. | Trocar por `CHECK (origin IN (...))` em `listens`, e tirar o gatilho `trg_identity_listen_origin_insert`. |
| `profile_relays` + `relay_destinations` | Relação **1:1**: uma guarda URL e token, a outra o estado da fila. **Cinco gatilhos** só mantêm uma tabela em sincronia com a outra. | Fundir em uma tabela, removendo os cinco gatilhos. |

**Caches derivados** (`ingest_source_summary`, `ingest_inbox_daily`): são reconstruíveis a partir de `ingest_events`, mantidos por 5 gatilhos para a página de saúde não varrer todos os recibos. Estão **vivos e justificados**; só devem ser revistos se a página de saúde mudar.

### 5.3 Colunas e gatilhos

**Colunas sem nenhuma referência no Go de produção (medido):**

| Tabela | Colunas | Observação |
| --- | --- | --- |
| `ingest_events` | `uncertainty_ms`, `client_name`, `normalized_snapshot`, `normalizer_version` | O cliente usado hoje vem de `original_snapshot`. O par do normalizador ainda faz parte de restrições históricas. |
| `ingest_events` | `health_source` (gerada) | Só existe para um índice. Manter enquanto o índice existir. |
| `integrations` | `token_ref`, `revoked_at`, `diagnostic_enabled` | **Podem estar reservadas para as ISSUE-086 a 091** (fontes, credenciais, webhooks). Confirmar antes de remover. |
| `listens` | `review_distinct` | Era a exceção do escritor v9 à chave única do legado. Sem leitura nem escrita desde a ISSUE-194. |
| `relay_sent_archive` | `sent_count` | Ver acima. |

**Colunas constantes (legado de dimensão).** Em `ingest_events`, `mode`, `resolver_variant` e `delivery_policy` ainda têm os **padrões do tempo da sombra** (`'shadow'`, `'v9'`, `'legacy'`), mas todo recibo novo é obrigado a entrar com `active`, `occurrence` e `durable` (gatilhos de admissão). Na prática são constantes, e todas as consultas filtram pelo único valor possível. São 3 das 59 colunas da tabela.

**Gatilhos legados (16 de 55):**

| Grupo | Gatilhos | Observação |
| --- | --- | --- |
| Sombra, variante e política (**9**) | `trg_ingest_event_mode_update`, `trg_ingest_event_shadow_insert`, `trg_ingest_resolver_variant_immutable`, `trg_occurrence_receipt_admission`, `trg_relay_active_policy`, `trg_occurrence_link_origin_insert`, `trg_occurrence_link_origin_update`, `trg_identity_listen_sync_v9`, `trg_relay_policy_one_way` | Guardam dimensões que só têm um valor. Saem junto das colunas constantes. |
| Ponte `tracks`↔`occurrences` (**7**) | `trg_identity_occurrence_bridge_update`, `trg_identity_occurrence_insert`, `trg_identity_track_delete`, `trg_identity_track_id_immutable`, `trg_identity_track_insert`, `trg_identity_track_recording_mbid`, `trg_identity_track_update` | Saem só se `tracks` sair (pacote C). |

### 5.4 O custo de espaço (medido)

Banco resultante de um Koito v2 sintético com 300 mil escutas e 40 mil faixas (114 MB de páginas em uso; arquivo de 125 MB):

| Estrutura | Tamanho | Observação |
| --- | ---: | --- |
| `listens` (tabela) | 19,6 MB | 9 colunas. |
| Índice da chave primária `event_id` | 13,1 MB | Texto de 32 caracteres. |
| Índice `UNIQUE(profile_id,event_id)` | 13,5 MB | Necessário para a chave estrangeira composta dos recibos. |
| `idx_listens_event_cursor (profile_id,listened_at,event_id)` | 13,7 MB | Cursor de exportação. |
| `idx_listens_track_id_listened_at` | 4,6 MB | |
| `idx_listens_profile_listened_at` | 4,1 MB | **Prefixo do `idx_listens_event_cursor`.** |
| `idx_listens_listened_at` | 3,7 MB | |
| `idx_listens_track_id` | 3,2 MB | **Prefixo do `idx_listens_track_id_listened_at`.** |
| `idx_listens_user_id` | 2,6 MB | `user_id` é só atribuição. |
| `listen_export_order` + índice | 25 MB | Sequência de exportação que nunca se repete. |

Só as escutas e o que gira em torno delas somam cerca de **103 dos 114 MB**. Três índices parecem redundantes (`profile_listened_at`, `track_id` e `user_id`, cerca de **10 MB**, 9% do banco), mas **cada um precisa de verificação de plano**: a medição da ISSUE-190 mostrou que o planejador escolhe `idx_listens_track_id`, então removê-lo muda planos.

### 5.5 Proposta em três pacotes

| Pacote | Conteúdo | Mexe em Go? | Ganho | Quando |
| --- | --- | --- | --- | --- |
| **A. Sem mudar comportamento** | Remover `ingest_event_details`, `outbox_details`, `legacy_relay_binding`, `relay_delivery_gate`; as colunas constantes `mode`, `resolver_variant`, `delivery_policy`; as colunas mortas confirmadas; os 9 gatilhos de sombra e política; `listens.review_distinct`; e, depois de verificar planos, os 3 índices redundantes. | Sim, nas consultas de ingestão, saúde e exclusão que filtram `mode` e `resolver_variant`; e na retenção. | **40 → 36 tabelas, 55 → 46 gatilhos**, 3 índices e até 12 colunas a menos (7 de `ingest_events`, 1 de `listens` e 4 de `integrations`, estas últimas dependem de confirmar as ISSUE-086 a 091). A remoção de `legacy_relay_binding` depende de aposentar o bootstrap de relay por variáveis. | **Junto da ISSUE-195**, antes da consolidação. |
| **B. Com ajuste em Go** | `profile_ingest_review` vira coluna; `event_origins` vira `CHECK`; `profile_relays` é fundida em `relay_destinations`; decisão sobre `relay_sent_archive` e `ingest_revisions`. | Sim, no relay e na revisão. | **→ 33 tabelas, → 40 gatilhos** (os 5 de sincronização do relay e o de origem saem). | Logo depois do A, na mesma consolidação se houver tempo. |
| **C. Refatoração maior** | Fundir `tracks` em `occurrences`: sai a coluna `legacy_track_id`, os 7 gatilhos de ponte, e `listens.track_id` passa a se chamar `occurrence_id`. Avaliar também `listen_export_order` (25 MB). | Sim, em **37 leituras e 8 escritas** do Go. | **→ 32 tabelas, → 33 gatilhos**, e o modelo passa a ter um nome só por conceito. | Issue própria, depois da 195. Não há pressa: o custo é a refatoração, e o risco é regressão em estatísticas. |

**Riscos comuns.** Todo item toca consultas de ingestão, que são as mais cobertas por teste mas também as mais sensíveis; por isso o A e o B pedem Reviewer em cada corte, plano de consulta antes e depois, e o salto v2 → atual repetido com digest e contagens. Bancos de desenvolvimento nas versões 3 a 23 **perdem o caminho de atualização** (custo já aceito pela ISSUE-195): recriar ou reimportar pelo export da ISSUE-183.

## 6. Metadados da ISSUE-055 no modelo

O estudo da 055 (22/11/2026) propõe enriquecer a escuta com gêneros, tags e datas, por uma cadeia de fontes (valores enviados na escuta, Navidrome do usuário, ListenBrainz, MusicBrainz, e provedores opcionais desligados por padrão). Aqui, o que importa é **como isso entra no banco**.

### 6.1 Ponto de partida (lido no schema real)

- `recordings` tem só `id`, `musicbrainz_id` e `mbid_source`. **Não há tags nem datas.**
- `releases` tem `id`, `musicbrainz_id`, `image`, `image_source` e `various_artists`. **Não há data de lançamento.**
- O TuneTrail **já recebe** `tags` no `additional_info` da escuta e **não as guarda** (o campo existe em `LbzAdditionalInfo`).
- O catálogo é **compartilhado entre perfis** e não contém dado pessoal. Tags e datas são dado de catálogo.

### 6.2 Modelo proposto (rascunho para discussão, não aplicado)

Migração **aditiva** (a 024), seguindo as convenções da seção 3.4. Tags e gêneros ligam à **gravação** (a música); datas ligam ao **lançamento**.

```sql
-- Datas no lançamento: precisão parcial e procedência.
ALTER TABLE releases ADD COLUMN release_date TEXT
    CHECK (release_date IS NULL OR release_date GLOB '[0-9][0-9][0-9][0-9]*');
ALTER TABLE releases ADD COLUMN release_date_precision TEXT
    CHECK (release_date_precision IS NULL OR release_date_precision IN ('year','month','day'));
ALTER TABLE releases ADD COLUMN original_year INTEGER
    CHECK (original_year IS NULL OR original_year BETWEEN 1000 AND 3000);
ALTER TABLE releases ADD COLUMN date_source TEXT;
ALTER TABLE releases ADD COLUMN date_fetched_at INTEGER;

CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    normalized TEXT NOT NULL UNIQUE CHECK (length(normalized) BETWEEN 1 AND 100)
);

CREATE TABLE recording_tags (
    recording_id INTEGER NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('genre','tag')),
    weight INTEGER CHECK (weight IS NULL OR weight >= 0),
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (recording_id, tag_id, source)
) WITHOUT ROWID;
CREATE INDEX idx_recording_tags_tag ON recording_tags(tag_id, recording_id);

-- Cache negativo e auditoria da busca por provedor.
CREATE TABLE metadata_fetch_log (
    entity_kind TEXT NOT NULL CHECK (entity_kind IN ('recording','release','artist')),
    entity_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok','not_found','error','disabled')),
    fetched_at INTEGER NOT NULL,
    next_retry_at INTEGER,
    PRIMARY KEY (entity_kind, entity_id, provider)
) WITHOUT ROWID;

-- Edição manual do usuário, que sempre vence.
CREATE TABLE metadata_overrides (
    entity_kind TEXT NOT NULL CHECK (entity_kind IN ('recording','release','artist')),
    entity_id INTEGER NOT NULL,
    field TEXT NOT NULL,
    value TEXT NOT NULL,
    edited_by INTEGER NOT NULL,
    edited_at INTEGER NOT NULL,
    PRIMARY KEY (entity_kind, entity_id, field)
) WITHOUT ROWID;
```

Pontos de atenção do rascunho:

- **Artistas**: a decisão 4 da 055 (artistas recebem tags e gêneros?) está em aberto. Se sim, `artist_tags` tem a mesma forma de `recording_tags`.
- **`edited_by` sem chave estrangeira**, como a atribuição de `listens.user_id`: apagar uma conta não deve apagar a correção.
- **Snapshot do recibo**: tags, gêneros, ano e data declarados pela fonte entram em `original_snapshot` (já é JSON), com procedência como `mbid_source` faz hoje. Não exige coluna.
- **Exportação e reimportação** (ISSUE-183) precisam incluir tags e datas; é código, não esquema.

### 6.3 Acesso por linha dos metadados

Tags, datas e o cache de busca são **dado de catálogo**: mesma política do catálogo (lê quem vê as páginas; ninguém escreve por perfil). A edição manual (`metadata_overrides`) exige `catalog.manage`, ou seja, **root**, e não depende de perfil. Nenhuma tabela nova carrega `profile_id`, então não há isolamento por linha a acrescentar e nenhum dado pessoal novo.

### 6.4 Custo esperado

- **Escrita**: a busca é assíncrona, fora do caminho do ACK, e cada entidade é buscada uma vez por provedor (cache negativo com prazo). Não toca `listens`.
- **Tamanho**: proporcional ao número de **gravações distintas** e de tags por gravação, não ao de escutas. Para a amostra de 300 mil escutas e 40 mil faixas, 40 mil gravações com poucas tags cada ficam na casa de dezenas de MB no pior caso, **estimativa a medir** com o fixture antes de fechar.
- **Leitura**: rankings por gênero ou ano precisam de `recording_tags(tag_id, recording_id)` e de uma junção com `listens` por ocorrência. O custo entra na medição do corte 6 da 055.

### 6.5 Encaixe na sequência do banco

A 055 define cortes; os que mexem no esquema são o **corte 2** (guardar o que já chega na escuta, migração aditiva) e, depois, as tabelas de busca e cache. A ordem decidida em 22/11/2026 para a janela mínima antes do primeiro upgrade de produção é **194 → 055 → 190 → 193 → 195**. A 194 e a 190 estão publicadas (schema 23). Falta a 055 (aguardando as decisões do estudo dela), a 193 e a 195.

## 7. Plano para atualizar do Koito v2 até a última versão

### 7.1 Estado de partida e de chegada

- **Partida:** banco do Koito 0.3 em SQLite, arquivo `koito.db`, `goose_db_version` na versão 2. Bancos criados antes do PR #257 do Koito (abril de 2026) chegam à versão 2 sem dois índices; a migração 4 os cria se faltarem.
- **Chegada:** schema 23 hoje (v0.74), ou o que existir quando o upgrade de produção acontecer (a 055 e a 193 entram antes).
- **Premissa já decidida:** a produção sai da v2 **direto** para a última versão. Nenhum banco de produção passa pelos degraus intermediários, por isso a ISSUE-195 propõe um arquivo único.

### 7.2 O que o servidor faz no primeiro start (sequência real)

1. Toma as travas de banco e de instância.
2. Tira um **instantâneo consistente** do `koito.db` (inclui o WAL confirmado) em `koito.db.pre-tunetrail-*.bak`, com permissão `0600`.
3. **Migra uma cópia** e a publica como `tunetrail.db`, com cada migração na sua transação, de 3 a 23.
4. Valida integridade, chaves estrangeiras, 1 conta ↔ 1 perfil, gatilhos obrigatórios e o layout contra o DDL embutido.
5. **Mantém o `koito.db` original intacto.** O servidor passa a usar o `tunetrail.db`.

### 7.3 O que muda nos dados

| Etapa | Efeito |
| --- | --- |
| v3 | Um perfil por usuário (`/default` se houver um só, `user-<id>` se houver vários). `api_keys` e `listens` ganham `profile_id`; a chave primária das escutas passa a incluir o perfil. O relay único do Koito vai para `profile_relays`. |
| v5 a v7 | Cada perfil ganha dono (1:1). Contas sem senha ficam `pending`. Tokens de ativação e recuperação. |
| v8 | Nascem `recordings` e `occurrences`. **Cada `tracks.id` vira o `occurrences.id` de mesmo valor.** `listens` é reconstruída com `event_id` aleatório e `origin='legacy'`. A migração compara contagens e conteúdo e aborta se algo difere. |
| v9 a v21 | Estruturas de recebimento e relay, vazias para um banco que nunca recebeu submissões. A v21 retira a sombra. |
| v22 e v23 | `avatar_color` e `duration_played`, ambas nulas. |

Tudo o que é **dado do usuário** (contas, escutas, chaves, catálogo) é preservado com os mesmos números. O que é novo nasce **vazio**.

### 7.4 Medições

**Medido neste estudo** (binário da v0.74; Koito v2 sintético com 2 mil artistas, 4 mil lançamentos, 40 mil faixas, 2 usuários e 2 chaves; 300 mil escutas):

| Medida | Resultado |
| --- | --- |
| Tamanho do `koito.db` de partida | 35 MB |
| Tempo da atualização v2 → 23, até o servidor responder | **11,5 s** |
| Escutas, ocorrências, gravações e perfis depois | 300.000 / 40.000 / 40.000 / 2 (nada perdido) |
| Backup automático | 34,7 MB |
| `tunetrail.db` resultante | **124,6 MB** (3,5 vezes o original) |
| Disco necessário no pico | cerca de 195 MB (35 + 35 + 125), **5,5 vezes** o original |

**Referências anteriores.**

- A amostra real de Koito da ISSUE-164 (72 escutas) migrou do schema 2 ao 22 com o mesmo digest antes, depois e após restaurar o backup.
- Um backup real v4 preservou 80 escutas na migração para o schema atual.
- A cópia de desenvolvimento v20 com 2.826 escutas migrou para a v21 sem perda e foi restaurada em diretório novo.
- Na ISSUE-082, num fixture de 300 mil **recibos** com WAL de 278 MB, a v12 levou cerca de 21 s. Isso **não está no caminho de um Koito v2**, que não tem recibos; serve como pior caso para um banco já em uso.

**Não medido:** um Koito v2 com mais de 300 mil escutas, e o tempo em disco lento. A recomendação (7.6) é repetir o ensaio com o tamanho real do banco de produção, com `scripts/validate_migration.py --source-v2`.

### 7.5 Verificações antes e depois

```sql
-- Antes (no koito.db): ponto de partida e contagens para comparar depois.
PRAGMA integrity_check;
SELECT max(version_id) FROM goose_db_version WHERE is_applied=1;      -- esperado: 2
SELECT (SELECT count(*) FROM listens), (SELECT count(*) FROM tracks), (SELECT count(*) FROM artists),
       (SELECT count(*) FROM releases), (SELECT count(*) FROM users), (SELECT count(*) FROM api_keys);

-- Depois (no tunetrail.db):
PRAGMA integrity_check;  PRAGMA foreign_key_check;
SELECT max(version_id) FROM goose_db_version WHERE is_applied=1;      -- esperado: a última
SELECT (SELECT count(*) FROM listens), (SELECT count(*) FROM occurrences), (SELECT count(*) FROM artists),
       (SELECT count(*) FROM releases), (SELECT count(*) FROM users), (SELECT count(*) FROM api_keys);
-- Cada conta tem exatamente um perfil, e todo perfil tem dono:
SELECT count(*) FROM users u WHERE NOT EXISTS (SELECT 1 FROM listening_profiles p WHERE p.user_id=u.id);   -- 0
SELECT count(*) FROM listening_profiles WHERE user_id IS NULL;                                           -- 0
```

Os números "antes" e "depois" devem coincidir: escutas, artistas, lançamentos, usuários e chaves iguais; ocorrências iguais às faixas. O `scripts/validate_migration.py` automatiza isso com um digest por tabela.

### 7.6 Roteiro de execução (runbook)

1. **Congelar.** Parar todo processo que escreva no diretório de dados. Copiar o diretório inteiro (configuração, imagens, arquivos SQLite). Guardar a cópia fora do servidor.
2. **Espaço.** Conferir disco livre de pelo menos **6 vezes** o tamanho do `koito.db`.
3. **Ensaiar numa cópia** com a mesma versão do binário que vai para produção, porta própria, sem rede para relay: `TUNETRAIL_CONFIG_DIR=<cópia>`. Rodar as verificações da 7.5. Cronometrar. Repetir o `scripts/validate_migration.py --source-v2 <cópia>`.
4. **Decisão de seguir (go/no-go):** contagens iguais, `integrity_check` e `foreign_key_check` limpos, entrada com a conta administrativa funcionando, chaves de API apontando para os perfis certos, e o tempo do ensaio dentro da janela combinada.
5. **Produção:** subir a versão. O primeiro start faz o backup, migra e publica. Reabrir o acesso só depois das verificações.
6. **Depois:** conferir a Home, o histórico e uma submissão nova pela API; revisar visibilidade dos perfis e o relay. Manter o `koito.db` e o backup até aceitar.
7. **Volta:** parar, preservar o diretório atual (tem as escritas novas), restaurar a cópia do passo 1 ou o `koito.db`, subir a versão que combina com ele. As escritas feitas depois do backup ficam fora do instantâneo restaurado.

### 7.7 Plano com os metadados (055) e a simplificação

Duas formas de combinar, e a recomendada:

| Opção | Como | Prós | Contras |
| --- | --- | --- | --- |
| **1. Atualizar já para o 23 e evoluir depois** | Produção sobe para o schema 23 agora; 055 e simplificações viram migrações 024 em diante, aplicadas no banco vivo. | Simples de começar. | Cada migração roda no banco **de produção**, com dados reais; as simplificações (remover tabelas, fundir) são as mais arriscadas e ficariam para depois do uso real. |
| **2. Decidir tudo antes, consolidar e subir uma vez (recomendada)** | Fechar 055 (corte 2), 193 e as simplificações A e B; **consolidar tudo no arquivo único da ISSUE-195**; ensaiar em escala; só então a produção atravessa o caminho **uma única vez**. | A produção nunca vê o legado nem as tabelas mortas. Menos gatilhos, menos código de versão no binário. Um único caminho a testar. | Atrasa o primeiro upgrade. Bancos de desenvolvimento nas versões 3 a 23 perdem o caminho de atualização (custo já aceito pela 195). |

**Sequência recomendada (opção 2):**

1. **Decisões** da seção 8 (o usuário).
2. **055, corte 2** (guardar o que já chega) e as migrações aditivas dos metadados, como a 024 em diante, ainda no esquema incremental.
3. **193** (normalização de metadados por perfil), se houver coluna ou tabela nova.
4. **Simplificações A e B** (seção 5.5), com Reviewer em cada corte e planos de consulta antes e depois.
5. **ISSUE-195:** gerar o arquivo único que leva um Koito v2 ao schema final, com o **teste de equivalência** (aplicar 001 até a última em ordem e comparar o `sqlite_master` normalizado com o do arquivo único) e o salto v2 → final com digests e contagens por perfil.
6. **Ensaio em escala:** v2 sintético de 300 mil e de 1 milhão de escutas, e cópia real, com backup, restauração e `quick_check`.
7. **Upgrade de produção** pelo roteiro 7.6.
8. **Pacote C** (`tracks` em `occurrences`) como issue própria, **depois** do upgrade, porque pede refatoração de 37 pontos e não bloqueia a produção.

### 7.8 Riscos e mitigações

| Risco | Efeito | Mitigação |
| --- | --- | --- |
| Falta de disco no pico | A migração falha no meio. | Exigir 6 vezes o tamanho do banco livre; o `koito.db` original nunca é tocado. |
| Banco de produção bem maior que o medido | A janela passa de 11,5 s. | Ensaiar com o tamanho real; a migração 8 é a que reconstrói `listens`. |
| Remover estruturas "mortas" que uma issue futura usaria | Retrabalho. | Confirmar `integrations` com as ISSUE-086 a 091 antes de remover colunas dela. |
| Consolidar sem teste de equivalência | O arquivo único diverge do histórico. | Teste obrigatório da 195 (`sqlite_master` normalizado). |
| Planos de consulta mudam ao remover índices | Páginas ficam mais lentas. | Medir antes e depois, como fez a ISSUE-190. |
| Metadados externos com termos de uso restritivos | Risco jurídico. | Provedores comerciais opcionais e desligados; verificar termos antes de guardar (já na 055). |
| Escrita nova entre o backup e o fim do upgrade | Perda na volta. | Congelar escritas antes (passo 1 do roteiro). |

## 8. Decisões que preciso do usuário

1. **Aprovar o pacote A** como parte da ISSUE-195 (remover tabelas mortas, colunas constantes e gatilhos de sombra e política)? Antes, confirmar com as ISSUE-086 a 091 se `integrations.token_ref`, `revoked_at` e `diagnostic_enabled` ficam. E decidir se o **bootstrap de relay por variáveis de ambiente** (`ENABLE_LBZ_RELAY`, `LBZ_RELAY_URL`, `LBZ_RELAY_TOKEN`) é aposentado, o que libera remover `legacy_relay_binding`.
2. **`ingest_revisions`**: expor na página "por que esta escuta está aqui" ou aceitar que é só auditoria por SQL? **`relay_sent_archive`**: mostrar a contagem de enviados ou remover?
3. **Pacote B** (fundir `profile_relays` em `relay_destinations`, `profile_ingest_review` virar coluna, `event_origins` virar `CHECK`): entra na mesma consolidação ou depois?
4. **Pacote C** (`tracks` em `occurrences`): confirmar que é issue própria **depois** do upgrade de produção.
5. **Índices de `listens`:** autorizar a medição de planos para remover `idx_listens_profile_listened_at`, `idx_listens_track_id` e `idx_listens_user_id` (cerca de 10 MB em 300 mil escutas)? E avaliar `listen_export_order` (25 MB)?
6. **Metadados:** seguir as decisões abertas da 055 (fonte sobre o Last.fm, tags populares separadas dos gêneros, artistas com tags, interface só no último corte, subsistema único para 174, 175 e 055) e o rascunho da seção 6.2?
7. **Opção de atualização:** confirmar a opção 2 (decidir, consolidar e subir uma vez) e se o ensaio em escala usa 300 mil e 1 milhão de escutas.
8. **Segurança por linha:** quer o teste estático de consultas (seção 4.4)?

## Apêndice A. Referência completa do esquema: tabelas, colunas e restrições

Gerado do banco real no schema 23. "Nula: não" inclui as chaves primárias. "Padrão" é o valor padrão da coluna.

### Catálogo

#### `artists`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `musicbrainz_id` | TEXT | sim |  |  |  |
| `image` | TEXT | sim |  |  |  |
| `image_source` | TEXT | sim |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `UNIQUE (musicbrainz_id)`

#### `artist_aliases`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `artist_id` | INTEGER | não |  | PK | → `artists`.`id` (apagar: cascade) |
| `alias` | TEXT | não |  | PK |  |
| `source` | TEXT | não |  |  |  |
| `is_primary` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (artist_id, alias)`

#### `releases`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `musicbrainz_id` | TEXT | sim |  |  |  |
| `image` | TEXT | sim |  |  |  |
| `image_source` | TEXT | sim |  |  |  |
| `various_artists` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `UNIQUE (musicbrainz_id)`

#### `release_aliases`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `release_id` | INTEGER | não |  | PK | → `releases`.`id` (apagar: cascade) |
| `alias` | TEXT | não |  | PK |  |
| `source` | TEXT | não |  |  |  |
| `is_primary` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (release_id, alias)`

#### `tracks`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `musicbrainz_id` | TEXT | sim |  |  |  |
| `release_id` | INTEGER | não |  |  | → `releases`.`id` (apagar: cascade) |
| `duration` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `UNIQUE (musicbrainz_id)`

#### `track_aliases`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `track_id` | INTEGER | não |  | PK | → `tracks`.`id` (apagar: cascade) |
| `alias` | TEXT | não |  | PK |  |
| `source` | TEXT | não |  |  |  |
| `is_primary` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (track_id, alias)`

#### `artist_releases`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `artist_id` | INTEGER | não |  | PK | → `artists`.`id` (apagar: cascade) |
| `release_id` | INTEGER | não |  | PK | → `releases`.`id` (apagar: cascade) |
| `is_primary` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (artist_id, release_id)`

#### `artist_tracks`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `artist_id` | INTEGER | não |  | PK | → `artists`.`id` (apagar: cascade) |
| `track_id` | INTEGER | não |  | PK | → `tracks`.`id` (apagar: cascade) |
| `is_primary` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (artist_id, track_id)`

#### `recordings`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `musicbrainz_id` | TEXT | sim |  |  |  |
| `mbid_source` | TEXT | não | 'legacy_declared' |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (mbid_source IN ('legacy_declared','validated'))`
- `UNIQUE (musicbrainz_id)`

#### `occurrences`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `recording_id` | INTEGER | não |  |  | → `recordings`.`id` |
| `release_id` | INTEGER | sim |  |  | → `releases`.`id` (apagar: set null) |
| `legacy_track_id` | INTEGER | sim |  |  |  |
| `track_mbid` | TEXT | sim |  |  |  |
| `disc_number` | INTEGER | sim |  |  |  |
| `track_number` | INTEGER | sim |  |  |  |
| `presented_title` | TEXT | sim |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (legacy_track_id IS NULL OR legacy_track_id=id)`
- `CHECK (disc_number>0)`
- `CHECK (track_number>0)`
- `UNIQUE (track_mbid)`
- `UNIQUE (legacy_track_id)`

#### `occurrence_credits`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `occurrence_id` | INTEGER | não |  | PK | → `occurrences`.`id` (apagar: cascade) |
| `position` | INTEGER | não |  | PK |  |
| `name` | TEXT | não |  |  |  |
| `artist_id` | INTEGER | sim |  |  | → `artists`.`id` (apagar: set null) |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (position>=0)`
- `CHECK (length(name)>0)`
- `PRIMARY KEY (occurrence_id, position)`

### Pessoas e acesso

#### `users`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `username` | TEXT | não |  |  |  |
| `role` | TEXT | não | 'user' |  |  |
| `password` | BLOB | não |  |  |  |
| `status` | TEXT | não | 'active' |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (role IN ('admin', 'user'))`
- `CHECK (status IN ('active','pending') AND ((status='pending') = (length(password)=0)))`
- `UNIQUE (username)`

#### `sessions`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | TEXT | não |  | PK |  |
| `user_id` | INTEGER | não |  |  | → `users`.`id` (apagar: cascade) |
| `created_at` | INTEGER | não |  |  |  |
| `expires_at` | INTEGER | não |  |  |  |
| `persistent` | INTEGER | não | 0 |  |  |

#### `account_tokens`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `user_id` | INTEGER | não |  | PK | → `users`.`id` (apagar: cascade) |
| `token_hash` | BLOB | não |  |  |  |
| `purpose` | TEXT | não |  |  |  |
| `expires_at` | INTEGER | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(token_hash) = 32)`
- `CHECK (purpose IN ('activation', 'reset'))`

#### `listening_profiles`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `slug` | TEXT | não |  |  |  |
| `display_name` | TEXT | não |  |  |  |
| `avatar_image_id` | TEXT | sim |  |  |  |
| `avatar_icon` | TEXT | sim |  |  |  |
| `created_at` | INTEGER | não |  |  |  |
| `user_id` | INTEGER | sim |  |  | → `users`.`id` (apagar: restrict) |
| `require_login` | INTEGER | não | 0 |  |  |
| `avatar_color` | TEXT | sim |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK ( length(slug) BETWEEN 1 AND 64 AND slug NOT GLOB '*[^a-z0-9-]*' AND substr(slug, 1, 1) != '-' AND substr(slug, -1, 1) != '-' AND slug NOT IN ('admin', 'apis', 'image', 'public', 'artist', 'album', 'track', 'chart', 'listens', 'rewind', 'theme-helper') )`
- `CHECK (length(trim(display_name)) > 0)`
- `CHECK (require_login IN (0,1))`
- `CHECK (avatar_color IS NULL OR (length(avatar_color) = 7 AND avatar_color GLOB '#[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'))`
- `UNIQUE (slug)`

#### `api_keys`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `key` | TEXT | não |  |  |  |
| `user_id` | INTEGER | sim |  |  |  |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `created_at` | INTEGER | não |  |  |  |
| `label` | TEXT | não | '' |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `UNIQUE (key)`

### Escutas

#### `listens`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `track_id` | INTEGER | não |  |  | → `occurrences`.`id` (apagar: cascade) |
| `listened_at` | INTEGER | não |  |  |  |
| `user_id` | INTEGER | sim |  |  |  |
| `client` | TEXT | não | '' |  |  |
| `event_id` | TEXT | não | lower(hex(randomblob(16))) | PK |  |
| `origin` | TEXT | não | 'legacy' |  | → `event_origins`.`origin` (apagar: restrict) |
| `review_distinct` | INTEGER | não | 0 |  |  |
| `duration_played` | INTEGER | sim |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(event_id)>0)`
- `CHECK (review_distinct IN (0,1))`
- `CHECK (duration_played IS NULL OR (typeof(duration_played) = 'integer' AND duration_played >= 0))`
- `UNIQUE (profile_id,event_id)`
- `UNIQUE (profile_id, event_id)`

#### `event_origins`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `origin` | TEXT | não |  | PK |  |
| `description` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(origin)>0)`

#### `listen_export_order`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `sequence` | INTEGER | não |  | PK |  |
| `event_id` | TEXT | não |  |  | → `listens`.`event_id` (apagar: cascade) |

Restrições (CHECK, UNIQUE e chave primária composta):
- `UNIQUE (event_id)`

### Recebimento (ingestão)

#### `integrations`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | TEXT | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `source_type` | TEXT | não |  |  |  |
| `display_name` | TEXT | não | '' |  |  |
| `token_ref` | TEXT | sim |  |  |  |
| `revoked_at` | INTEGER | sim |  |  |  |
| `created_at` | INTEGER | não |  |  |  |
| `details_retention_days` | INTEGER | não | 90 |  |  |
| `diagnostic_enabled` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(id)>0)`
- `CHECK (length(source_type)>0)`
- `CHECK (details_retention_days BETWEEN 7 AND 3650)`
- `CHECK (diagnostic_enabled IN (0,1))`
- `UNIQUE (profile_id,id)`
- `UNIQUE (profile_id, id)`

#### `ingest_events`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `receipt_id` | TEXT | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `source_instance_id` | TEXT | não |  |  | → `integrations`.`id` |
| `schema_version` | INTEGER | não |  |  |  |
| `source_type` | TEXT | não |  |  |  |
| `adapter_version` | TEXT | não |  |  |  |
| `batch_id` | TEXT | não |  |  |  |
| `item_ordinal` | INTEGER | não |  |  |  |
| `operation` | TEXT | não |  |  |  |
| `received_at` | INTEGER | não |  |  |  |
| `external_namespace` | TEXT | sim |  |  |  |
| `external_event_id` | TEXT | sim |  |  |  |
| `external_id_trusted` | INTEGER | não | 0 |  |  |
| `client_idempotency_key` | TEXT | sim |  |  |  |
| `batch_digest` | TEXT | sim |  |  |  |
| `file_digest` | TEXT | sim |  |  |  |
| `file_format` | TEXT | sim |  |  |  |
| `format_version` | TEXT | sim |  |  |  |
| `listened_at` | TEXT | sim |  |  |  |
| `original_time` | TEXT | sim |  |  |  |
| `time_precision` | TEXT | não |  |  |  |
| `time_semantics` | TEXT | não |  |  |  |
| `original_offset` | TEXT | sim |  |  |  |
| `uncertainty_ms` | INTEGER | sim |  |  |  |
| `client_name` | TEXT | sim |  |  |  |
| `client_version` | TEXT | sim |  |  |  |
| `original_snapshot` | TEXT | não |  |  |  |
| `normalized_snapshot` | TEXT | sim |  |  |  |
| `normalizer_version` | TEXT | sim |  |  |  |
| `snapshot_digest` | TEXT | não |  |  |  |
| `digest_version` | TEXT | não |  |  |  |
| `state` | TEXT | não | 'received' |  |  |
| `identity_status` | TEXT | não | 'unresolved' |  |  |
| `duplicate_decision` | TEXT | sim |  |  |  |
| `reason_codes` | TEXT | não | '[]' |  |  |
| `evidence` | TEXT | não | '{}' |  |  |
| `rule_version` | TEXT | sim |  |  |  |
| `resolution_version` | INTEGER | não | 0 |  |  |
| `resolved_at` | INTEGER | sim |  |  |  |
| `event_id` | TEXT | sim |  |  | → `listens`.`event_id` (apagar: cascade) |
| `occurrence_id` | INTEGER | sim |  |  | → `occurrences`.`id` (apagar: set null) |
| `recording_id` | INTEGER | sim |  |  | → `recordings`.`id` (apagar: set null) |
| `attempt_count` | INTEGER | não | 0 |  |  |
| `next_attempt_at` | INTEGER | sim |  |  |  |
| `lease_owner` | TEXT | sim |  |  |  |
| `lease_until` | INTEGER | sim |  |  |  |
| `work_version` | INTEGER | não | 0 |  |  |
| `closed_at` | INTEGER | sim |  |  |  |
| `mode` | TEXT | não | 'shadow' |  |  |
| `resolver_variant` | TEXT | não | 'v9' |  |  |
| `delivery_policy` | TEXT | não | 'legacy' |  |  |
| `parked_at` | INTEGER | sim |  |  |  |
| `park_reason` | TEXT | sim |  |  |  |
| `retry_attempt_base` | INTEGER | não | 0 |  |  |
| `health_source` (gerada) | TEXT | sim |  |  |  |
| `health_client` (gerada) | TEXT | sim |  |  |  |
| `claim_attempt_base` | INTEGER | não | 0 |  |  |
| `reprocessed_at` | INTEGER | sim |  |  |  |
| `reprocessed_by` | INTEGER | sim |  |  |  |
| `api_key_id` | INTEGER | sim |  |  |  |
| `inbox_key` (gerada) | INTEGER | sim |  |  |  |
| `inbox_metadata` (gerada) | INTEGER | sim |  |  |  |
| `relay_admission_generation` | INTEGER | sim |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(receipt_id)>0)`
- `CHECK (schema_version>0)`
- `CHECK (length(source_type)>0)`
- `CHECK (length(adapter_version)>0)`
- `CHECK (length(batch_id)>0)`
- `CHECK (item_ordinal>=0)`
- `CHECK (length(operation)>0)`
- `CHECK (external_namespace IS NULL OR length(external_namespace)>0)`
- `CHECK (external_event_id IS NULL OR length(external_event_id)>0)`
- `CHECK (external_id_trusted IN (0,1))`
- `CHECK (client_idempotency_key IS NULL OR length(client_idempotency_key)>0)`
- `CHECK (file_digest IS NULL OR (length(file_digest)=64 AND file_digest NOT GLOB '*[^0-9a-f]*'))`
- `CHECK (time_precision IN ('unknown','second','millisecond','microsecond','nanosecond','minute','estimated'))`
- `CHECK (time_semantics IN ('start','end','unknown'))`
- `CHECK (uncertainty_ms>=0)`
- `CHECK (json_valid(original_snapshot) AND json_type(original_snapshot)='object')`
- `CHECK (normalized_snapshot IS NULL OR (json_valid(normalized_snapshot) AND json_type(normalized_snapshot)='object'))`
- `CHECK (length(snapshot_digest)>0)`
- `CHECK (length(digest_version)>0)`
- `CHECK (state IN ('received','pending_identity','pending_duplicate','invalid','accepted','duplicate','retryable'))`
- `CHECK (identity_status IN ('unresolved','pending','invalid','resolved'))`
- `CHECK (duplicate_decision IN ('same','different','review'))`
- `CHECK (json_valid(reason_codes) AND json_type(reason_codes)='array')`
- `CHECK (json_valid(evidence) AND json_type(evidence)='object')`
- `CHECK (resolution_version>=0)`
- `CHECK (attempt_count>=0)`
- `CHECK (work_version>=0)`
- `CHECK (mode IN ('shadow','active'))`
- `CHECK (resolver_variant IN ('v9','occurrence'))`
- `CHECK (delivery_policy IN ('legacy','durable'))`
- `CHECK ((parked_at IS NULL AND park_reason IS NULL) OR (parked_at IS NOT NULL AND park_reason IS NOT NULL AND mode='active' AND state='retryable' AND closed_at IS NULL AND lease_owner IS NULL AND lease_until IS NULL))`
- `CHECK (retry_attempt_base BETWEEN 0 AND attempt_count)`
- `CHECK (claim_attempt_base BETWEEN 0 AND attempt_count)`
- `CHECK ((reprocessed_at IS NULL AND reprocessed_by IS NULL) OR (reprocessed_at IS NOT NULL AND reprocessed_by IS NOT NULL AND reprocessed_by>0))`
- `UNIQUE (profile_id,receipt_id)`
- `CHECK ((state IN ('accepted','duplicate'))=(event_id IS NOT NULL))`
- `CHECK (external_id_trusted=0 OR (external_namespace IS NOT NULL AND external_event_id IS NOT NULL))`
- `CHECK ((external_namespace IS NULL)=(external_event_id IS NULL))`
- `CHECK (client_idempotency_key IS NULL OR (batch_digest IS NOT NULL AND length(batch_digest)>0))`
- `CHECK ((file_digest IS NULL AND file_format IS NULL AND format_version IS NULL) OR (file_digest IS NOT NULL AND file_format IS NOT NULL AND length(file_format)>0 AND format_version IS NOT NULL AND length(format_version)>0))`
- `CHECK ((normalized_snapshot IS NULL)=(normalizer_version IS NULL))`
- `CHECK ((lease_owner IS NULL)=(lease_until IS NULL))`
- `CHECK (closed_at IS NULL OR state IN ('invalid','accepted','duplicate'))`
- `CHECK (state!='invalid' OR (rule_version IS NOT NULL AND json_array_length(reason_codes)>0))`
- `UNIQUE (profile_id, receipt_id)`

#### `ingest_event_details`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `ingest_events`.`profile_id` (apagar: cascade) |
| `receipt_id` | TEXT | não |  |  | → `ingest_events`.`receipt_id` (apagar: cascade) |
| `resolution_version` | INTEGER | não |  |  |  |
| `kind` | TEXT | não |  |  |  |
| `payload` | TEXT | não |  |  |  |
| `closed_at` | INTEGER | sim |  |  |  |
| `protected` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (resolution_version>=0)`
- `CHECK (length(kind)>0)`
- `CHECK (json_valid(payload))`
- `CHECK (protected IN (0,1))`

#### `ingest_observations`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | TEXT | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `ingest_events`.`profile_id` (apagar: cascade) |
| `receipt_id` | TEXT | não |  |  | → `ingest_events`.`receipt_id` (apagar: cascade) |
| `schema_version` | INTEGER | não |  |  |  |
| `snapshot_digest` | TEXT | não |  |  |  |
| `digest_version` | TEXT | não |  |  |  |
| `original_snapshot` | TEXT | não |  |  |  |
| `received_at` | INTEGER | não |  |  |  |
| `closed_at` | INTEGER | sim |  |  |  |
| `protected` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(id)>0)`
- `CHECK (schema_version>0)`
- `CHECK (length(snapshot_digest)>0)`
- `CHECK (length(digest_version)>0)`
- `CHECK (json_valid(original_snapshot) AND json_type(original_snapshot)='object')`
- `CHECK (protected IN (0,1))`
- `UNIQUE (profile_id,receipt_id,digest_version,snapshot_digest)`
- `UNIQUE (profile_id, receipt_id, digest_version, snapshot_digest)`

#### `ingest_revisions`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `ingest_events`.`profile_id` (apagar: cascade) |
| `receipt_id` | TEXT | não |  | PK | → `ingest_events`.`receipt_id` (apagar: cascade) |
| `resolution_version` | INTEGER | não |  | PK |  |
| `author` | TEXT | não |  |  |  |
| `decision` | TEXT | não |  |  |  |
| `reason_codes` | TEXT | não |  |  |  |
| `evidence` | TEXT | não |  |  |  |
| `rule_version` | TEXT | não |  |  |  |
| `created_at` | INTEGER | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (resolution_version>0)`
- `CHECK (length(author)>0)`
- `CHECK (decision IN ('same','different','review','invalid'))`
- `CHECK (json_valid(reason_codes) AND json_type(reason_codes)='array' AND json_array_length(reason_codes)>0)`
- `CHECK (json_valid(evidence) AND json_type(evidence)='object')`
- `CHECK (length(rule_version)>0)`
- `PRIMARY KEY (profile_id, receipt_id, resolution_version)`

#### `ingest_discards`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `receipt_id` | TEXT | não |  |  |  |
| `actor_id` | INTEGER | não |  |  |  |
| `discarded_at` | INTEGER | não |  |  |  |
| `origin_state` | TEXT | não |  |  |  |
| `park_reason` | TEXT | sim |  |  |  |
| `reason_codes` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (actor_id > 0)`
- `CHECK (json_valid(reason_codes) AND json_type(reason_codes)='array')`

#### `ingest_source_summary`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `integrations`.`profile_id` (apagar: cascade) |
| `source_instance_id` | TEXT | não |  | PK | → `integrations`.`id` (apagar: cascade) |
| `source` | TEXT | não |  | PK |  |
| `client` | TEXT | não |  | PK |  |
| `received_at` | INTEGER | não |  |  |  |
| `receipt_id` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `PRIMARY KEY (profile_id, source_instance_id, source, client)`

#### `ingest_inbox_daily`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `day` | INTEGER | não |  | PK |  |
| `key_id` | INTEGER | não |  | PK |  |
| `client` | TEXT | não |  | PK |  |
| `state` | TEXT | não |  | PK |  |
| `metadata` | INTEGER | não |  | PK |  |
| `total` | INTEGER | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (total>=0)`
- `PRIMARY KEY (profile_id, day, key_id, client, state, metadata)`

#### `profile_ingest_review`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `review` | INTEGER | não | 1 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (review IN (0,1))`

### Relay (envio a outro servidor)

#### `profile_relays`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `url` | TEXT | não |  |  |  |
| `token` | TEXT | não |  |  |  |

#### `relay_destinations`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `generation` | INTEGER | não | 1 |  |  |
| `config_version` | INTEGER | não | 1 |  |  |
| `open_count` | INTEGER | não | 0 |  |  |
| `paused_until` | INTEGER | sim |  |  |  |
| `pause_class` | TEXT | sim |  |  |  |
| `backoff_count` | INTEGER | não | 0 |  |  |
| `lease_owner` | TEXT | sim |  |  |  |
| `lease_until` | INTEGER | sim |  |  |  |
| `work_version` | INTEGER | não | 0 |  |  |
| `last_claim_at` | INTEGER | não | 0 |  |  |
| `awaiting_approval` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (open_count BETWEEN 0 AND 100000)`
- `CHECK (backoff_count>=0)`
- `CHECK (awaiting_approval IN (0,1))`
- `CHECK ((lease_owner IS NULL)=(lease_until IS NULL))`

#### `relay_allowlist`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `host_port` | TEXT | não |  | PK |  |
| `cidr` | TEXT | não | '' | PK |  |
| `allow_http` | INTEGER | não |  |  |  |
| `allow_private` | INTEGER | não |  |  |  |
| `origin` | TEXT | não |  | PK |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (allow_http IN (0,1))`
- `CHECK (allow_private IN (0,1))`
- `CHECK (origin IN ('manual','seeded'))`
- `CHECK (allow_http=1 OR (allow_private=1 AND cidr<>''))`
- `CHECK ((allow_private=1)=(cidr<>''))`
- `PRIMARY KEY (profile_id, host_port, cidr, origin)`

#### `outbox`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | TEXT | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `listening_profiles`.`id` (apagar: cascade) |
| `destination_id` | TEXT | não |  |  | → `integrations`.`id` |
| `event_id` | TEXT | não |  |  | → `listens`.`event_id` (apagar: cascade) |
| `operation` | TEXT | não |  |  |  |
| `delivery_key` | TEXT | não |  |  |  |
| `state` | TEXT | não | 'pending' |  |  |
| `created_at` | INTEGER | não |  |  |  |
| `next_attempt_at` | INTEGER | sim |  |  |  |
| `attempt_count` | INTEGER | não | 0 |  |  |
| `lease_owner` | TEXT | sim |  |  |  |
| `lease_until` | INTEGER | sim |  |  |  |
| `work_version` | INTEGER | não | 0 |  |  |
| `closed_at` | INTEGER | sim |  |  |  |
| `result` | TEXT | não | '{}' |  |  |
| `format_version` | INTEGER | não | 0 |  |  |
| `last_attempt_at` | INTEGER | sim |  |  |  |
| `relay_generation` | INTEGER | não | 1 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (length(id)>0)`
- `CHECK (length(event_id)>0)`
- `CHECK (length(operation)>0)`
- `CHECK (length(delivery_key)>0)`
- `CHECK (state IN ('pending','retryable','failed','sent','discarded'))`
- `CHECK (attempt_count>=0)`
- `CHECK (work_version>=0)`
- `CHECK (json_valid(result) AND json_type(result)='object')`
- `CHECK (format_version>=0)`
- `CHECK (relay_generation>0)`
- `UNIQUE (profile_id,id)`
- `UNIQUE (profile_id,destination_id,event_id,operation)`
- `UNIQUE (profile_id,destination_id,delivery_key)`
- `CHECK ((lease_owner IS NULL)=(lease_until IS NULL))`
- `CHECK ((state IN ('sent','discarded'))=(closed_at IS NOT NULL))`
- `UNIQUE (profile_id, destination_id, delivery_key)`
- `UNIQUE (profile_id, destination_id, event_id, operation)`
- `UNIQUE (profile_id, id)`

#### `outbox_details`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `profile_id` | INTEGER | não |  |  | → `outbox`.`profile_id` (apagar: cascade) |
| `outbox_id` | TEXT | não |  |  | → `outbox`.`id` (apagar: cascade) |
| `payload` | TEXT | não |  |  |  |
| `closed_at` | INTEGER | sim |  |  |  |
| `protected` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (json_valid(payload))`
- `CHECK (protected IN (0,1))`

#### `relay_sent_archive`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `sent_day` | INTEGER | não |  | PK |  |
| `sent_count` | INTEGER | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (sent_count>0)`
- `PRIMARY KEY (profile_id, sent_day)`

#### `relay_delivery_gate`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `policy` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (id=1)`
- `CHECK (policy IN ('legacy','durable'))`

#### `legacy_relay_binding`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `profile_id` | INTEGER | sim |  |  | → `listening_profiles`.`id` (apagar: set null) |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (id = 1)`

### Restauração e desfazer

#### `archive_identity`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `singleton` | INTEGER | não |  | PK |  |
| `namespace` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (singleton=1)`
- `UNIQUE (namespace)`

#### `archive_catalog`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listening_profiles`.`id` (apagar: cascade) |
| `namespace` | TEXT | não |  | PK |  |
| `kind` | TEXT | não |  | PK |  |
| `source_id` | INTEGER | não |  | PK |  |
| `local_id` | INTEGER | não |  |  |  |
| `trusted` | INTEGER | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (trusted IN (0,1))`
- `PRIMARY KEY (profile_id, namespace, kind, source_id)`

#### `archive_events`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `profile_id` | INTEGER | não |  | PK | → `listens`.`profile_id` (apagar: cascade) |
| `namespace` | TEXT | não |  | PK |  |
| `source_event_id` | TEXT | não |  | PK |  |
| `event_id` | TEXT | não |  |  | → `listens`.`event_id` (apagar: cascade) |
| `origin` | TEXT | não |  |  |  |
| `provenance` | TEXT | não |  |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (json_valid(provenance))`
- `UNIQUE (profile_id,event_id)`
- `UNIQUE (profile_id, event_id)`
- `PRIMARY KEY (profile_id, namespace, source_event_id)`

#### `recording_moves`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `source_track` | INTEGER | não |  | PK |  |
| `destination_track` | INTEGER | não |  | PK |  |
| `occurrence_id` | INTEGER | não |  | PK | → `occurrences`.`id` (apagar: cascade) |
| `original_recording` | INTEGER | não |  |  |  |
| `merged_recording` | INTEGER | não |  |  |  |
| `stage` | INTEGER | não |  |  |  |
| `expires_at` | INTEGER | não | 0 |  |  |

Restrições (CHECK, UNIQUE e chave primária composta):
- `CHECK (stage IN (0,1,2))`
- `PRIMARY KEY (source_track, destination_track, occurrence_id)`

### Infraestrutura

#### `goose_db_version`

| Coluna | Tipo | Nula | Padrão | Chave | Referência |
| --- | --- | --- | --- | --- | --- |
| `id` | INTEGER | não |  | PK |  |
| `version_id` | INTEGER | não |  |  |  |
| `is_applied` | INTEGER | não |  |  |  |
| `tstamp` | TIMESTAMP | sim | datetime('now') |  |  |


## Apêndice B. Índices

| Tabela | Índice | Definição |
| --- | --- | --- |
| `api_keys` | `idx_api_keys_profile_id` | `(profile_id)` |
| `api_keys` | `idx_api_keys_user_id` | `(user_id)` |
| `archive_catalog` | `idx_archive_catalog_local` | `(kind,local_id)` |
| `archive_catalog` | `idx_archive_catalog_trusted` | `(namespace,kind,source_id) WHERE trusted=1` |
| `artist_aliases` | `idx_artist_aliases_alias` | `(alias)` |
| `artist_aliases` | `idx_artist_aliases_artist_id` | `(artist_id)` |
| `artist_releases` | `idx_artist_releases_release_id` | `(release_id)` |
| `artist_tracks` | `idx_artist_tracks_artist_id` | `(artist_id)` |
| `artist_tracks` | `idx_artist_tracks_track_id` | `(track_id)` |
| `ingest_discards` | `idx_ingest_discard_receipt` | `(profile_id,receipt_id,id DESC)` |
| `ingest_discards` | `idx_ingest_discard_retention` | `(profile_id,discarded_at,id)` |
| `ingest_event_details` | `idx_ingest_details_purge` | `(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0` |
| `ingest_event_details` | `idx_ingest_details_receipt` | `(profile_id,receipt_id)` |
| `ingest_events` | `idx_ingest_active_queue` | `(state,next_attempt_at,received_at) WHERE mode='active' AND closed_at IS NULL` |
| `ingest_events` | `idx_ingest_client_key` | `UNIQUE (profile_id,source_instance_id,operation,client_idempotency_key,item_ordinal) WHERE client_idempotency_key IS NOT NULL` |
| `ingest_events` | `idx_ingest_discard_queue` | `(profile_id,received_at,receipt_id) WHERE event_id IS NULL AND state IN ('received','retryable','pending_identity','pending_duplicate','invalid')` |
| `ingest_events` | `idx_ingest_event` | `(profile_id,event_id)` |
| `ingest_events` | `idx_ingest_external_key` | `UNIQUE (profile_id,source_instance_id,external_namespace,external_event_id) WHERE external_id_trusted=1` |
| `ingest_events` | `idx_ingest_file_key` | `UNIQUE (profile_id,source_instance_id,file_format,format_version,file_digest,item_ordinal) WHERE file_digest IS NOT NULL` |
| `ingest_events` | `idx_ingest_health_decisions` | `(profile_id,coalesce(resolved_at,received_at) DESC,receipt_id DESC) WHERE duplicate_decision IS NOT NULL` |
| `ingest_events` | `idx_ingest_health_received` | `(profile_id,received_at DESC,receipt_id DESC)` |
| `ingest_events` | `idx_ingest_health_review` | `(profile_id,received_at DESC,receipt_id DESC) WHERE mode='active' AND state='pending_duplicate' AND closed_at IS NULL` |
| `ingest_events` | `idx_ingest_inbox_edges` | `(profile_id,received_at,inbox_key,health_client,state,inbox_metadata)` |
| `ingest_events` | `idx_ingest_inbox_example` | `(profile_id,inbox_key,health_client,inbox_metadata,received_at DESC,receipt_id DESC) WHERE inbox_metadata IN (1,2)` |
| `ingest_events` | `idx_ingest_occurrence` | `(occurrence_id) WHERE occurrence_id IS NOT NULL` |
| `ingest_events` | `idx_ingest_parked` | `(profile_id,parked_at DESC,receipt_id DESC) WHERE parked_at IS NOT NULL` |
| `ingest_events` | `idx_ingest_profile_time` | `(profile_id,listened_at)` |
| `ingest_events` | `idx_ingest_queue` | `(state,next_attempt_at,received_at) WHERE closed_at IS NULL` |
| `ingest_events` | `idx_ingest_recording` | `(recording_id) WHERE recording_id IS NOT NULL` |
| `ingest_events` | `idx_ingest_resolver_queue` | `(received_at,receipt_id) WHERE mode='active' AND closed_at IS NULL AND parked_at IS NULL AND state IN ('received','retryable')` |
| `ingest_events` | `idx_ingest_source_receipt` | `( profile_id,source_instance_id,health_source,health_client,received_at DESC,receipt_id DESC )` |
| `ingest_observations` | `idx_ingest_observations_purge` | `(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0` |
| `ingest_source_summary` | `idx_ingest_source_summary_recent` | `(profile_id,received_at DESC,receipt_id DESC)` |
| `listening_profiles` | `idx_listening_profiles_user_id` | `UNIQUE (user_id)` |
| `listens` | `idx_listens_event_cursor` | `(profile_id,listened_at,event_id)` |
| `listens` | `idx_listens_listened_at` | `(listened_at)` |
| `listens` | `idx_listens_profile_listened_at` | `(profile_id,listened_at)` |
| `listens` | `idx_listens_track_id` | `(track_id)` |
| `listens` | `idx_listens_track_id_listened_at` | `(track_id,listened_at)` |
| `listens` | `idx_listens_user_id` | `(user_id)` |
| `occurrence_credits` | `idx_occurrence_credits_artist` | `(artist_id)` |
| `occurrences` | `idx_occurrences_recording` | `(recording_id)` |
| `occurrences` | `idx_occurrences_release` | `(release_id)` |
| `outbox` | `idx_outbox_event` | `(profile_id,event_id)` |
| `outbox` | `idx_outbox_queue` | `(state,next_attempt_at,created_at) WHERE closed_at IS NULL` |
| `outbox` | `idx_outbox_relay_capacity` | `(profile_id,destination_id,relay_generation,created_at,id) WHERE format_version>0 AND state='failed' AND json_extract(result,'$.reason')='queue_capacity'` |
| `outbox` | `idx_outbox_relay_due` | `(profile_id,destination_id,relay_generation,next_attempt_at,id) WHERE format_version>0 AND state IN ('pending','retryable')` |
| `outbox_details` | `idx_outbox_details_parent` | `(profile_id,outbox_id)` |
| `outbox_details` | `idx_outbox_details_purge` | `(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0` |
| `recording_moves` | `idx_recording_moves_expiry` | `(expires_at) WHERE stage=1` |
| `recording_moves` | `idx_recording_moves_occurrence` | `(occurrence_id)` |
| `release_aliases` | `idx_release_aliases_alias` | `(alias)` |
| `sessions` | `idx_sessions_expires_at` | `(expires_at)` |
| `track_aliases` | `idx_track_aliases_alias` | `(alias)` |
| `tracks` | `idx_tracks_release_id` | `(release_id)` |

## Apêndice C. Gatilhos

| Categoria | Gatilho | Tabela | Quando | Finalidade | Legado |
| --- | --- | --- | --- | --- | --- |
| Cache derivado | `trg_ingest_inbox_delete` | `ingest_events` | AFTER DELETE | Idem, ao apagar um recibo. |  |
| Cache derivado | `trg_ingest_inbox_insert` | `ingest_events` | AFTER INSERT | Atualiza as contagens diárias da Inbox. |  |
| Cache derivado | `trg_ingest_inbox_update` | `ingest_events` | AFTER UPDATE | Idem, ao mudar um recibo. |  |
| Cache derivado | `trg_ingest_source_admit` | `ingest_events` | AFTER INSERT | Atualiza o último recibo por origem e cliente. |  |
| Cache derivado | `trg_ingest_source_delete` | `ingest_events` | AFTER DELETE | Idem, ao apagar um recibo. |  |
| Catálogo | `trg_delete_orphan_releases` | `artist_releases` | AFTER DELETE | Apaga o lançamento que ficou sem artistas. |  |
| Catálogo | `trg_archive_artist_delete` | `artists` | AFTER DELETE | Remove mapeamentos de restauração de um artista apagado (o SQLite reaproveita IDs). |  |
| Catálogo | `trg_archive_release_delete` | `releases` | AFTER DELETE | Remove mapeamentos de restauração de um lançamento apagado. |  |
| Contas e perfis | `trg_api_key_profile_immutable` | `api_keys` | BEFORE UPDATE OF profile_id ON api_keys | Uma chave de API não muda de perfil. |  |
| Contas e perfis | `trg_profile_owner_immutable` | `listening_profiles` | BEFORE UPDATE OF user_id ON listening_profiles | O dono de um perfil não muda. |  |
| Contas e perfis | `trg_profile_owner_required` | `listening_profiles` | BEFORE INSERT | Um perfil sempre tem dono. |  |
| Contas e perfis | `trg_last_admin_delete` | `users` | BEFORE DELETE | O último administrador ativo não pode ser apagado. |  |
| Contas e perfis | `trg_last_admin_demote` | `users` | BEFORE UPDATE OF role,status ON users | O último administrador ativo não pode ser rebaixado ou desativado. |  |
| Escutas | `trg_identity_event_id_immutable` | `listens` | BEFORE UPDATE | O `event_id` de uma escuta não muda. |  |
| Escutas | `trg_identity_listen_occurrence_guard` | `listens` | BEFORE UPDATE OF track_id ON listens | Uma escuta resolvida por ocorrência não pode ser movida. |  |
| Escutas | `trg_identity_listen_origin_immutable` | `listens` | BEFORE UPDATE OF origin ON listens | A origem de uma escuta não muda. |  |
| Escutas | `trg_identity_listen_origin_insert` | `listens` | BEFORE INSERT | A origem precisa estar registrada em `event_origins`. |  |
| Escutas | `trg_identity_listen_sync_v9` | `listens` | AFTER UPDATE OF track_id ON listens | Ponte do resolvedor v9: exige revisão ao mover a faixa de uma escuta. | sim |
| Escutas | `trg_listen_export_order` | `listens` | AFTER INSERT | Dá a cada escuta nova um número de sequência de exportação. |  |
| Ponte tracks↔occurrences | `trg_identity_occurrence_bridge_update` | `occurrences` | BEFORE UPDATE OF legacy_track_id,release_id ON occurrences | Impede mudar a ligação com a faixa antiga. | sim |
| Ponte tracks↔occurrences | `trg_identity_occurrence_insert` | `occurrences` | BEFORE INSERT | Exige faixa antiga correspondente, ou título sem lançamento. | sim |
| Ponte tracks↔occurrences | `trg_identity_occurrence_recording_guard` | `occurrences` | BEFORE UPDATE OF recording_id ON occurrences | Impede reatribuir a gravação de uma ocorrência já ligada. |  |
| Ponte tracks↔occurrences | `trg_identity_recording_orphan` | `occurrences` | AFTER DELETE | Apaga a gravação que ficou sem ocorrências. |  |
| Ponte tracks↔occurrences | `trg_occurrence_linked_release_guard` | `occurrences` | BEFORE UPDATE OF release_id ON occurrences | Impede mover o lançamento de uma ocorrência já ligada. |  |
| Ponte tracks↔occurrences | `trg_recording_move_revisions` | `occurrences` | AFTER UPDATE OF recording_id ON occurrences | Registra revisão nos recibos afetados ao mover uma ocorrência de gravação. |  |
| Ponte tracks↔occurrences | `trg_identity_track_delete` | `tracks` | AFTER DELETE | Propaga a exclusão de uma faixa antiga para a ocorrência. | sim |
| Ponte tracks↔occurrences | `trg_identity_track_id_immutable` | `tracks` | BEFORE UPDATE | O ID de uma faixa antiga não muda. | sim |
| Ponte tracks↔occurrences | `trg_identity_track_insert` | `tracks` | AFTER INSERT | Cria a ocorrência com o mesmo ID ao inserir uma faixa antiga. | sim |
| Ponte tracks↔occurrences | `trg_identity_track_recording_mbid` | `tracks` | BEFORE UPDATE OF musicbrainz_id ON tracks | Impede trocar o MBID de gravação compartilhada ou validada pela faixa antiga. | sim |
| Ponte tracks↔occurrences | `trg_identity_track_update` | `tracks` | AFTER UPDATE OF release_id,musicbrainz_id ON tracks | Mantém a ocorrência em sincronia quando a faixa antiga muda. | sim |
| Recebimento | `trg_ingest_event_mode_update` | `ingest_events` | BEFORE UPDATE | O modo (sombra ou ativo) de um recibo é imutável. | sim |
| Recebimento | `trg_ingest_event_shadow_insert` | `ingest_events` | BEFORE INSERT | Um recibo de sombra não pode materializar decisão. | sim |
| Recebimento | `trg_ingest_link_insert` | `ingest_events` | BEFORE INSERT | Exige um vínculo coerente entre recibo, escuta, ocorrência e gravação. |  |
| Recebimento | `trg_ingest_link_update` | `ingest_events` | BEFORE UPDATE OF state,identity_status,profile_id,event_id,occurrence_id,recording_id ON ingest_events | Idem, na atualização. |  |
| Recebimento | `trg_ingest_resolution_version` | `ingest_events` | BEFORE UPDATE OF resolution_version,state,identity_status,duplicate_decision,event_id,occurrence_id,recording_id ON ingest_events | Uma decisão nova exige uma versão de resolução maior. |  |
| Recebimento | `trg_ingest_resolver_variant_immutable` | `ingest_events` | BEFORE UPDATE OF resolver_variant ON ingest_events | A variante do resolvedor (v9 ou ocorrência) é imutável. | sim |
| Recebimento | `trg_ingest_snapshot_immutable` | `ingest_events` | BEFORE UPDATE | A identidade e o conteúdo original do recibo não mudam. |  |
| Recebimento | `trg_occurrence_link_origin_insert` | `ingest_events` | BEFORE INSERT | A origem do vínculo precisa combinar com a variante do resolvedor. | sim |
| Recebimento | `trg_occurrence_link_origin_update` | `ingest_events` | BEFORE UPDATE OF event_id,state ON ingest_events | Idem, na atualização. | sim |
| Recebimento | `trg_occurrence_receipt_admission` | `ingest_events` | BEFORE INSERT | Recibos novos só entram com ingestão por ocorrência ativa. | sim |
| Recebimento | `trg_relay_active_policy` | `ingest_events` | BEFORE INSERT | A política de entrega do recibo precisa combinar com o escritor ativo. | sim |
| Recebimento | `trg_ingest_observation_immutable` | `ingest_observations` | BEFORE UPDATE | As variantes observadas não mudam. |  |
| Recebimento | `trg_ingest_revision_active_insert` | `ingest_revisions` | BEFORE INSERT | Uma revisão exige um recibo ativo. |  |
| Relay | `trg_relay_admission_generation` | `ingest_events` | AFTER INSERT | Grava no recibo a geração do destino aprovada na admissão. |  |
| Relay | `trg_outbox_active_insert` | `outbox` | BEFORE INSERT | Uma intenção de envio exige um recibo aceito e ativo. |  |
| Relay | `trg_relay_archive_sent` | `outbox` | BEFORE DELETE | Move a contagem diária de itens enviados para o arquivo antes de apagá-los. |  |
| Relay | `trg_relay_count_delete` | `outbox` | AFTER DELETE | Idem, ao apagar um item. |  |
| Relay | `trg_relay_count_insert` | `outbox` | AFTER INSERT | Mantém o contador de itens na fila do destino. |  |
| Relay | `trg_relay_count_update` | `outbox` | AFTER UPDATE OF state,format_version,relay_generation ON outbox | Idem, ao mudar o estado de um item. |  |
| Relay | `trg_relay_destination_delete` | `profile_relays` | AFTER DELETE | Remove o destino quando `profile_relays` perde a linha. |  |
| Relay | `trg_relay_destination_insert` | `profile_relays` | AFTER INSERT | Cria o destino quando `profile_relays` ganha uma linha. |  |
| Relay | `trg_relay_destination_update` | `profile_relays` | AFTER UPDATE | Atualiza o destino quando URL ou token mudam. |  |
| Relay | `trg_relay_draft_change` | `profile_relays` | AFTER UPDATE | Volta a rascunho quando a URL muda. |  |
| Relay | `trg_relay_draft_insert` | `profile_relays` | AFTER INSERT | Marca o relay novo como rascunho aguardando aprovação. |  |
| Relay | `trg_relay_policy_one_way` | `relay_delivery_gate` | BEFORE UPDATE OF policy ON relay_delivery_gate | A política de entrega só vai de legada para durável. | sim |

## Apêndice D. Views

As três views mostram o nome principal (o alias com `is_primary=1`) de cada entidade do catálogo. São usadas em 4 (`tracks_with_title`), 8 (`releases_with_title`) e 14 (`artists_with_name`) pontos do Go de produção.

**`artists_with_name`**

```sql
CREATE VIEW artists_with_name AS
SELECT a.*, aa.alias AS name
FROM artists a
JOIN artist_aliases aa ON a.id = aa.artist_id
WHERE aa.is_primary = 1
```

**`releases_with_title`**

```sql
CREATE VIEW releases_with_title AS
SELECT r.*, ra.alias AS title
FROM releases r
JOIN release_aliases ra ON r.id = ra.release_id
WHERE ra.is_primary = 1
```

**`tracks_with_title`**

```sql
CREATE VIEW tracks_with_title AS
SELECT t.*, ta.alias AS title
FROM tracks t
JOIN track_aliases ta ON t.id = ta.track_id
WHERE ta.is_primary = 1
```

