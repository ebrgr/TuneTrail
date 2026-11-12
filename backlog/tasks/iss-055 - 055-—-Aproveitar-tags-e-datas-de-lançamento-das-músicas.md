---
id: ISS-055
title: 055 — Aproveitar tags e datas de lançamento das músicas
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-8
dependencies: []
references: []
type: feature
ordinal: 55000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/055-metadados-tags-lancamento.md` — numeração local 055. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Aberta · **Criada:** 10/11/2026 · **Roadmap:** [metadados musicais](<../docs/roadmaps/doc-001 - ROADMAP.md>)

## Objetivo

Investigar os dados enviados pelo Navidrome e outros clientes compatíveis com ListenBrainz e adaptar a ingestão e o banco SQLite para aproveitar **tags, incluindo gêneros quando identificáveis, ano e data de lançamento**. Usar primeiro as informações efetivamente recebidas na API de escutas; quando ausentes, consultar a API do MusicBrainz, desde que haja identificação confiável e a integração esteja habilitada.

Esta issue é um planejamento local. Não altera o esquema, não executa migração e não publica issue no GitHub.

## Situação confirmada no código

- O TuneTrail aceita `additional_info.tags` na estrutura [LbzAdditionalInfo](../../internal/listenbrainz/types.go), mas o [handler de escutas](../../engine/handlers/lbz_submit_listen.go) não passa as tags para `catalog.SubmitListenOpts`; logo elas não são persistidas por esse caminho.
- `LbzAdditionalInfo` também não declara campos de ano ou data de lançamento; é necessário confirmar nomes, tipos, unidade e disponibilidade efetivos nos clientes antes de suportá-los. O protocolo aceita metadados adicionais, mas isso não garante que Navidrome envie gênero ou ano em toda escuta.
- As tabelas de catálogo [tracks e releases](../../db/migrations_sqlite/001_initial_schema.sql) não têm campos de gênero, ano ou data de lançamento. O esquema vigente é **SQLite consolidado v3**.
- O cliente [MusicBrainz](../../internal/mbz/mbz.go) já consulta gravações e lançamentos por MBID, mas os modelos atuais não representam os campos necessários para este enriquecimento.
- O [bootstrap SQLite](../../internal/db/sqlite/bootstrap.go) limita a versão do esquema a 3 e valida explicitamente o layout v3. Uma migração futura precisa atualizar e testar esse fluxo, inclusive o banco legado `koito.db`; não basta adicionar um arquivo SQL.

## Contrato proposto para implementação

1. **Confirmar a origem:** capturar ou montar fixtures representativas das versões usadas do Navidrome, sem dados pessoais, e conferir quais campos de tags/gênero, ano e data realmente chegam. Documentar os campos ListenBrainz padronizados e eventuais extensões de clientes sem presumir que existem. `tags` pode representar palavras livres, humor ou descrições, não apenas gêneros.
2. **Prioridade por campo:** preservar valores enviados pela API como primeira fonte; quando estiverem ausentes ou inválidos, usar MusicBrainz. Uma resposta vazia, erro, consulta desabilitada ou indisponibilidade nunca deve apagar informação existente. Definir a atualização quando uma escuta posterior trouxer informação mais completa, preservando a origem e sem substituir edição manual por dados automáticos.
3. **Correspondência confiável:** preferir os MBIDs da gravação, lançamento e grupo já recebidos. Se não houver ID, estudar busca por artista, faixa e álbum, com validação e tratamento explícito de ambiguidade; não associar automaticamente a primeira versão encontrada. Respeitar configurações, limites de requisição, cache e funcionamento offline; não impedir o registro de escutas quando o provedor falhar.
4. **Tags e gênero:** guardar tags multivaloradas e sua procedência; normalizar duplicações sem perder a forma original. Não classificar indiscriminadamente toda tag como gênero. Definir como apresentar ou filtrar gêneros confiáveis separadamente de tags livres e como lidar com alterações, remoções e conflitos.
5. **Ano e data:** distinguir ano original, data do lançamento específico e a data da gravação quando aplicável. Preservar precisão parcial (`YYYY`, `YYYY-MM`, `YYYY-MM-DD`), ausência e origem; não inventar mês/dia, nem inferir datas por fuso. Ano derivado de uma data deve manter coerência sem sobrescrever um valor de origem diferente de forma silenciosa.
6. **Persistência e compartilhamento:** estudar tabelas de tags e vínculos com faixas e/ou lançamentos, além de colunas ou tabela de metadados de lançamento. O catálogo é compartilhado entre perfis: definir política para dados conflitantes ou tags pessoais de usuários distintos, sem expor informação privada de um perfil a outro. Avaliar limites de tamanho, índices, consultas e mesclagem/deleção de catálogo.
7. **Migração:** projetar migração aditiva versionada a partir do SQLite v3, sem editar retroativamente as migrações consolidadas nem descartar escutas. Se a próxima versão for v4, atualizar os limites e validações do bootstrap, os fluxos de `tunetrail.db` e de migração/backup de `koito.db`, documentação e compatibilidade. Cobrir banco novo, v3 existente e recuperação de falha; a versão atual permanece v3 até a implementação.
8. **Disponibilização:** planejar onde os dados ficarão disponíveis em consultas, API e interface, com compatibilidade retroativa. Mudanças de UI e filtros devem respeitar o significado e a procedência definidos acima; não assumir que dados históricos já possuem tags ou datas.

## Critérios de aceite

- [ ] Documentar o payload efetivo do Navidrome testado e os campos do protocolo; testar ausência, múltiplas tags e valores inválidos.
- [ ] Persistir tags e metadados de lançamento disponíveis, com origem e regras de prioridade por campo; preservar edições manuais e distinguir tag livre de gênero.
- [ ] Usar MusicBrainz apenas como fallback seguro; testar identificador correto, dados parciais, divergência, ambiguidade, indisponibilidade e integração desabilitada.
- [ ] Preservar ano e precisão da data, sem confundir edição com primeiro lançamento; testar ano isolado, mês e data completos.
- [ ] Migrar banco v3 existente e banco legado suportado com backup, reinício, rollback/recuperação e integridade SQLite; testar também instalação nova, históricos e isolamento entre perfis.
- [ ] Cobrir atualizações repetidas, deduplicação, concorrência, importação, edição, mesclagem, exclusão, exportação e compatibilidade das APIs existentes.
- [ ] Atualizar documentação e changelog; registrar resultados e, se houver alteração visual, capturas para comparação.
- [ ] Manter a issue aberta até o teste funcional e aceite do usuário.

## Referências

- [Formato JSON de escutas ListenBrainz](https://listenbrainz.readthedocs.io/en/latest/users/json.html) — `tags` são opcionais e podem conter classificações livres; há exemplo de envio por Navidrome.
- [API MusicBrainz](https://musicbrainz.org/doc/MusicBrainz_API) — consultas por entidade e dados adicionais como gêneros e tags.
- [Integrações externas do Navidrome](https://www.navidrome.org/docs/usage/external-integrations/) — metadados e uso de IDs MusicBrainz.
<!-- SECTION:DESCRIPTION:END -->
