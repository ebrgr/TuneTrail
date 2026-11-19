---
id: ISSUE-133
title: 133 — Migração aditiva para gravações e ocorrências com IDs compatíveis
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-19T17:45:00Z'
labels:
  - banco
milestone: m-11
dependencies:
  - ISSUE-132
  - ISSUE-080
  - ISSUE-162
references:
  - backlog/decisions/decision-009
  - backlog/evidencias/issue-133/PLANO.md
documentation:
  - doc-036
  - doc-039
  - doc-040
type: feature
ordinal: 146000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 3 do estudo da ISSUE-116 (doc-036). Migração aditiva: faixas atuais viram ocorrências com os mesmos IDs, cada uma ganha uma gravação, escutas migram 1:1 com origem legacy. Coordenar a numeração do schema com a ISSUE-080 (v5 é sugestão, não reserva); backup e restauração validados antes. Contrato normativo: doc-040 e decision-009 (ISSUE-079, aceitas em 2026-11-17). Acrescenta (doc-040 §10): introduzir event_id próprio e relacionar o ledger e a outbox da 080 ANTES de ativar escritores novos; permitir ocorrência sem lançamento e várias ocorrências por gravação/lançamento; remover a chave composta de listens pode exigir reconstrução controlada da tabela; definir ponte de compatibilidade ou corte coordenado com os leitores essenciais da ISSUE-135 (nunca ativar escritores novos com leitores assumindo gravação única por álbum ou evento único por segundo); exclusão por event_id e cursor estável acompanham a ponte; a exclusão de um evento remove recibos e chaves vinculados e nunca deixa tombstone. Restaurar backup antigo exige reconciliação antes de reativar workers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Banco com vários perfis, aliases, MBIDs nulos e FKs migra com contagem de eventos idêntica por perfil
- [ ] #2 Nenhuma fusão automática por título; mapeamento old_track_id → ocorrência/gravação guardado
- [ ] #3 IDs e URLs anteriores continuam válidos
- [ ] #4 Interrupção simulada reverte a transação; restauração do backup comprovada
- [ ] #5 event_id próprio introduzido e ledger/outbox da ISSUE-080 relacionados a ele antes de ativar escritores novos
- [ ] #6 Exclusão por event_id remove os recibos e as chaves vinculados, sem tombstone, e uma reimportação posterior cria evento novo
- [ ] #7 Ponte de compatibilidade ou corte coordenado com os leitores da ISSUE-135 definidos e testados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Planejar ocorrências sem lançamento, múltiplas ocorrências por gravação/lançamento, eventos com identidade própria e migração 1:1 preservando IDs/URLs. Coordenar ledger com ISSUE-080. Exigir ponte de compatibilidade ou corte coordenado entre leitores/escritores antes de habilitar o novo modelo.

---

author: codex
created: 2026-11-18T18:37:00Z

---

Coordenação de schema aprovada pelo usuário/Advisor em 18/11/2026: a ISSUE-158 usa 005_accounts_profiles.sql (v5), após a v4 publicada pela 080. Esta issue deve usar a próxima versão disponível, v6 ou posterior, sem editar migrações já aplicadas; pipeline e identidade musical não fazem parte da 158.

---

author: ebrgr
created: 2026-11-19T17:26:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-133` na branch `feat/issue-133-migracao-aditiva-para-gravacoes-e` criada a partir de `main`.

---

author: codex
created: 2026-11-19T17:30:00Z

---

Fase 1 registrada em PLANO.md. Proposta: migração 008 após a 007 da ISSUE-162, reconstrução de listens e das famílias do ledger/outbox com FKs ligadas e troca filho-primeiro, mapeamento de IDs e recibos de sombra preservados, compatibilidade legada até corte coordenado. Inclui backup/restore e matriz dos 7 ACs. Aguardando parecer do Advisor com effort high e aprovação explícita para Fase 2. Nenhum SQL, runtime, banco real ou AC alterado.

---

author: advisor
created: 2026-11-19T17:37:00Z

---

Parecer do Advisor (Modo A+B, effort high, leitura estática na v0.41 791fe9b): REPROVADO COM RESSALVAS. A direção do plano está certa e nenhuma ressalva bloqueia; corrigir PLANO.md antes da Fase 2.
Confirmado: FKs reais. tracks é pai de artist_tracks (001:50), track_aliases (001:56) e listens (003:54); releases é pai de tracks (002:11), artist_releases (001:28) e release_aliases (001:34), com trg_delete_orphan_releases (002:60). ingest_events é pai de details/observations/revisions (004:98,115,148); outbox é pai de outbox_details (004:195). listens não tem filhos na v6 e a 007 da 162 referencia só users. O filho-primeiro com FK ON (DSN em bootstrap.go:105) é correto.
Ressalvas obrigatórias:
1. Reconstruir listens agora, na forma final: depois da 008 ela vira pai do ledger e da outbox. event_id TEXT NOT NULL PRIMARY KEY (TEXT PK em rowid table aceita NULL) com DEFAULT (lower(hex(randomblob(16)))), UNIQUE(profile_id,event_id) como alvo das FKs, origin DEFAULT 'legacy'. Manter o triplo como UNIQUE INDEX completo, não parcial: o índice parcial quebra listen.go:38-39 (ON CONFLICT sem WHERE) e o cursor legado. O corte faz DROP INDEX, sem reconstrução. Adiar tempo original e duração observada para ADD COLUMN (134/097).
2. Regra de forma: nas tabelas novas (recordings, occurrences, listens, ledger), FKs, CHECKs e enums já no contrato final. Ponte e corte só em TRIGGERS e ÍNDICES removíveis, porque mudar FK depois exige reconstruir pais com filhos. Sincronizar tracks→occurrence/recording com triggers AFTER INSERT/UPDATE OF release_id,musicbrainz_id/DELETE em tracks (track.go:178,322,395,750; album.go:694; sqlite.go:299), em vez de dual write em Go ou FK CASCADE em legacy_track_id. occurrences.release_id anulável ON DELETE SET NULL; listens.track_id→occurrences ON DELETE CASCADE.
3. Um único alocador de IDs: occurrences.id INTEGER PRIMARY KEY AUTOINCREMENT; o trigger insere id=NEW.id de forma estrita (nunca OR IGNORE). Guardas até o corte: proibir occurrence sem track ou sem release, listens.origin<>'legacy' e UPDATE de tracks.id. recordings com sequência própria (não igualar a track_id) e limpeza de órfã via trigger. Merge legado (track.go:707-716) continua igual até o corte.
4. Manter trg_ingest_revision_inactive e trg_outbox_inactive (004:151,183), só com nova mensagem; quem ativar remove depois. Reconstruir o ledger uma vez com os CHECKs finais (estados do doc-040 §3, accepted/duplicate ⇔ event_id NOT NULL), FK (profile_id,event_id)→listens ON DELETE CASCADE no ledger e na outbox, mesma ordem de colunas, 3 índices únicos parciais, triggers de imutabilidade e FK de integração DEFERRABLE intactos.
5. Asserções na 008 via TEMP TABLE com CHECK(ok=1): contagem por perfil, EXCEPT nos dois sentidos, pragma_foreign_key_check vazio, contagem do ledger e das filhas. validateSchemaLayout (ingest_schema.go:20-45) só compara num sentido: incluir checagem reversa limitada aos objetos da 008. Os limites em bootstrap.go:124,159,281 e backup.go:51-57 colidem com a 162: registrar --depends-on ISSUE-162. Goose recusa 007 aplicada depois da 008.
6. Interrupção (AC4): goose com FS de teste em que a 008 falha no fim. Conferir versão 7 e digests intactos, nova tentativa e banco v8 recusado pelo binário v0.41 (bootstrap.go:124).
Recomendações sobre as decisões: (1) aprovar filho-primeiro com track_id=ocorrência; o nome pode ser renomeado na 135 e o risco real é a colisão de IDs, tratada no item 3. (2) Manter o merge legado até o corte; mapear por occurrences.legacy_track_id, sem tabela de mapa sem FK. (3) Não vincular recibos de sombra por heurística. Preservá-los byte a byte, manter a limpeza da 171 enquanto só existir o escritor legado e exigir, como gate da 081, a purga dos recibos de sombra com event_id NULL, com prévia e contagens. Recibos estimated em outro segundo ficam fora da exclusão individual, e suas chaves virariam tombstone de fato. (4) Ensaio em snapshot autorizado pelo usuário, com serviço parado e disco ≥3x o banco. Ordem: 162 → 133 (sem mudança de comportamento) → 134/081 atrás das guardas → 135 + migração de ativação no mesmo binário.

---

author: ebrgr
created: 2026-11-19T17:45:00Z

---

Decisão do usuário em 2026-11-19: aceito todas as recomendações do Advisor. Reconstrução filho-primeiro com FKs ligadas e track_id igual à ocorrência (nome pode mudar na 135); manter merge legado até o corte e o mapa em occurrences.legacy_track_id, com guardas contra colisão. Preservar byte a byte os recibos de sombra sem vínculo, sem associação heurística, e manter a limpeza da 171 enquanto só houver o escritor legado. Antes de ativar a 081, purgar os recibos de sombra com event_id nulo, com prévia e contagens. Ensaio somente em snapshot expressamente autorizado, serviço parado e disco livre de pelo menos 3× o banco; até indicação, só fixtures ou cópias sintéticas. Ordem: 162 publicada na main → 133 sem mudança de comportamento → 134/081 atrás das guardas → 135 com a migração de ativação no mesmo binário. Aguardar nova mensagem para iniciar a Fase 2.

---
<!-- COMMENTS:END -->
