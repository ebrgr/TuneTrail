---
id: ISSUE-195
title: >-
  195 — Consolidar as migrações de SQL em um único salto da v2 para o schema
  final
status: Aberta
assignee: []
created_date: '2026-11-23T00:31:00Z'
updated_date: '2026-11-23T00:38:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-194
  - ISSUE-055
  - ISSUE-190
  - ISSUE-193
type: chore
ordinal: 208000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 22/11/2026 sobre simplificar as migrações. Produção sai do schema v2 (Koito) direto para a última versão; os 19 arquivos 003 a 021 (cerca de 2.006 linhas, 86 gatilhos criados, parte deles derrubada pelo próprio 021) são andaime de transição que nenhum banco de produção atravessa passo a passo. Hoje: 21 arquivos .sql mais migrations.go; schema atual com 40 tabelas, 56 índices, 60 gatilhos e 3 views (cerca de 980 linhas de DDL); código que conhece versões em internal/db/sqlite/{bootstrap,relay_schema,sqlite}.go; cerca de 23 arquivos de teste cobrem caminhos de atualização. PROPOSTA (Opção A, a confirmar): um único arquivo 003 que leva um banco Koito v2 ao schema final, preservando só as transformações de dados que importam (1:1 de contas e perfis, chaves de API sob o perfil, mapeamento de faixas para ocorrências e gravações com IDs estáveis, reconstrução de listens com event_id e origin, relay) e eliminando sombra, flags, guardas temporárias e gatilhos criados e depois derrubados. Ficam 001, 002 e o novo 003. PROVAS EXIGIDAS: (1) teste de equivalência: aplicar 001 a 021 (a partir da tag v0.65) e comparar o sqlite_master normalizado com o do arquivo único; (2) salto v2 para a última com digests e contagens por perfil iguais (reaproveitar TestBootstrapKoitoV2WithoutUpstreamIndexes, 200 escutas, e um banco v2 maior sintético); (3) ensaio em cópia, sem rede, com backup e restauração e quick_check; (4) o histórico das migrações permanece no Git e na tag v0.65. CUSTOS ACEITOS: bancos de desenvolvimento nas versões 3 a 21 (inclusive .local/koito/tunetrail.db e cópias de ensaio) perdem o caminho de atualização: recriar ou reimportar pelo export da ISSUE-183, com recusa clara quando o goose vir versão maior que a conhecida; os testes de atualização intermediária (~23) são substituídos pelos de equivalência. ORDEM: esta issue vem POR ÚLTIMO, depois da ISSUE-194 (que limpa o código legado e deixa o schema final sem colunas de transição como mode, resolver_variant e delivery_policy de ingest_events, se nada mais as usar) e da ISSUE-055 (tags, gêneros, ano e data de lançamento), e depois das demais issues que alteram o banco que o usuário decidir entregar antes do primeiro upgrade de produção. Como produção nunca viu as migrações 003 a 021, qualquer migração nova (022 em diante) feita antes do primeiro upgrade também entra no arquivo único. ISSUES ABERTAS QUE PODEM ALTERAR O BANCO (levantamento de 22/11/2026, a confirmar em cada uma): ALTA: 055 (tags, gêneros, ano e datas de lançamento com procedência e precisão parcial; descrição ainda cita schema v3, precisa ser refeita sobre o modelo de recordings, occurrences e releases do schema 21); 190 (coluna duration_played em listens, NULL para desconhecido, mais snapshot e exportação); 193 (regras de normalização por perfil com prévia e versão da regra aplicada no recibo); 121 (escopos nas chaves de API, coluna ou tabela de escopos) e 145 (tokens por dispositivo para apps nativos, tabelas novas); 118 (favoritos e feedback do ListenBrainz por perfil e gravação, tabela nova). MÉDIA: 096 (grupos de artistas em tabela própria) e 137 (agrupamento de remaster e deluxe por perfil, desligado por padrão, coerente com os grupos de 096); 086 (checkpoints de fontes e agendador), 088 (fonte Subsonic por perfil), 089 (segredos de webhook por perfil), 090 (tokens OAuth do Spotify fora do JSON) e 091 (sessão Audioscrobbler): provavelmente compartilham tabelas de configuração, credencial e checkpoint de fontes, a desenhar uma vez; 174 e 175 (MBID obtido de serviço externo como afirmação declarada com fonte e data: colunas de procedência fora do snapshot; recordings.mbid_source hoje só aceita legacy_declared e validated) e 187 item 4; 097 (duração observada separada da do catálogo, sobrepõe a 190); 120 (exclusão de artistas e álbuns das estatísticas globais). BAIXA ou A CONFIRMAR: 177 (índices de desempenho, cerca de +17,85 MB em 1 milhão de escutas), 136 (associação sem diferença de maiúsculas, talvez índice sobre alias normalizado), 165 (limite do cache de imagens por tamanho total e idade, talvez controle de último acesso), 112 (visualização de logs, só gera tabela se persistir logs), 192 (origem das escutas na central de saúde, usa listens.origin existente, talvez índice), 138 (fusão global de remaster e deluxe, operação de dados com backup). SEM ALTERAÇÃO PREVISTA: 084, 093, 094, 095, 098, 099, 101, 102, 103, 104, 105, 127, 128, 146, 147, 148 a 153, 164 a 169, 176, 178, 189. Reviewer ao final; Advisor (effort high) recomendado por tocar migração de dados.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-23T00:38:00Z

---

Decisão do usuário (22/11/2026, 'aceito a sugestão'): o mínimo de issues que alteram o banco e entram ANTES do primeiro upgrade de produção, e portanto antes desta consolidação, é ISSUE-055 (tags, gêneros, ano e datas), ISSUE-190 (duration_played) e ISSUE-193 (normalização de metadados por perfil), além da limpeza do código legado (ISSUE-194). As demais issues listadas como possíveis alterações de banco (121, 145, 118, 096, 137, 086 a 091, 174, 175, 097, 120, 177, 136, 165, 112, 192, 138) ficam fora desta janela: se vierem depois, entram como migrações incrementais normais sobre o schema final. Ordem sugerida: 194, 055, 190, 193, 195.

---
<!-- COMMENTS:END -->
