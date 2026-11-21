# Parecer do Advisor — ISSUE-183 (commit 66e843f)

Modo A, effort high. Eixos: design, segurança/concorrência, escalabilidade, regras do projeto.
Base: migração 014, `archive.go`, `mutation_plan.go`, `track.go` (merge/undo/delete), `exports.go`, importador. Não executei código nem modifiquei arquivos; os itens marcados "não executado" vêm de leitura.

**Parecer do Advisor: REPROVADO COM RESSALVAS** (1 item de segurança a corrigir; demais não bloqueantes)

## Falhas críticas
1. **Vazamento entre perfis pela restauração de arquivo** (`internal/db/sqlite/archive.go:mapID`, condição `c.namespace == c.localNamespace && (trusted || table == "occurrences")`).
   O `namespace` da instalação é gravado em todo export do próprio usuário. Um dono de perfil (não root) pode editar o JSON, manter o namespace local e trocar `OccurrenceID` por qualquer ID existente. `mapID` devolve `SELECT id FROM occurrences WHERE id=?` sem verificar que o perfil já ouviu essa ocorrência, e o INSERT em `listens` vincula uma escuta do perfil a ela. Como `requireRelated` libera as páginas do que o perfil ouviu, o usuário passa a ler título, artistas e capa de itens que só outro perfil ouviu (catálogo compartilhado), enumerando IDs. Não executado; não há teste para esse caso (os testes cobrem ida e volta, plano do export e reconciliação).
   Correção mínima: para importação não confiável (`ImportedCatalogProfile != 0`), reutilizar uma ocorrência local só se `EXISTS(SELECT 1 FROM listens WHERE profile_id=? AND track_id=?)` ou se já houver mapeamento em `archive_catalog` do próprio perfil; caso contrário criar uma nova. Adicionar teste com perfil A importando ID de ocorrência ouvida só por B.

## Melhorias (ganho real)
2. **Exportação pode perder eventos concorrentes**: o cursor é `event_id`, aleatório (`randomblob`) para eventos novos, então um evento gravado durante a exportação com ID menor que o cursor é omitido sem aviso. O Codex documenta "consistência por página". Sugestão: fixar um teto de `rowid` de `listens` no início (cursor `(rowid)` com o `event_id` apenas exportado) ou devolver o teto no primeiro lote e filtrar por ele nas páginas seguintes.
3. **Fusão em transação longa**: o gatilho `trg_recording_move_revisions` roda por ocorrência movida (UPDATE em `ingest_events` + INSERT em `ingest_revisions`). Em gravações com milhares de ocorrências/recibos bloqueia o escritor único, afetando o ACK da ingestão. Medir numa fixture grande e, se passar de centenas de milissegundos, mover a revisão para uma operação em lotes.
4. **Semântica da fusão mudou** (agora une identidade de gravação e mantém faixa e lançamento de origem). Registrar no CHANGELOG/doc que fundir faixas não remove mais a faixa de origem nem une lançamentos; conferir que a UI de revisão de catálogo explica isso (o Codex alterou `catalog-review.html`).
5. `recording_moves` stage=1 nunca expira; definir política (ex.: descartar ao editar a ocorrência) ou aceitar explicitamente.

## Regras do projeto
Issue-first, comentários e CHANGELOG em inglês: o diff declara conformidade; não reauditei o texto. Migração v14 com backup de v13 e guarda de downgrade: conforme.

## Correção alternativa
Ver item 1 (trecho mínimo). Após o ajuste, rever só `archive.go` e o novo teste.
