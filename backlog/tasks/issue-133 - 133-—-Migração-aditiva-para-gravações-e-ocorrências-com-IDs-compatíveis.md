---
id: ISSUE-133
title: 133 — Migração aditiva para gravações e ocorrências com IDs compatíveis
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-19T19:06:00Z'
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
  - backlog/evidencias/issue-133/VERIFICACAO.md
  - backlog/evidencias/issue-133/PARECER-REVIEWER.md
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
- [x] #2 Nenhuma fusão automática por título; mapeamento old_track_id → ocorrência/gravação guardado
- [x] #3 IDs e URLs anteriores continuam válidos
- [ ] #4 Interrupção simulada reverte a transação; restauração do backup comprovada
- [x] #5 event_id próprio introduzido e ledger/outbox da ISSUE-080 relacionados a ele antes de ativar escritores novos
- [x] #6 Exclusão por event_id remove os recibos e as chaves vinculados, sem tombstone, e uma reimportação posterior cria evento novo
- [x] #7 Ponte de compatibilidade ou corte coordenado com os leitores da ISSUE-135 definidos e testados
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

author: codex
created: 2026-11-19T17:56:00Z

---

Fase 2, passos 1–6: rebase sobre a main v0.42, migração 008 e ponte legada verificadas em fixtures sintéticas. AC2/3/5/6/7 verificados no escopo da 133; AC1 e AC4 seguem abertos até ensaio autorizado. make check passou (13/13 E2E), replay da sombra manteve as decisões e os binários v0.41/v0.42 recusaram v8. Integração dos leitores e escritores novos fica para o corte coordenado. Evidência em VERIFICACAO.md. Aguardar indicação de snapshot autorizado e nova instrução para o passo 7. Mantido Em andamento.

---

author: advisor
created: 2026-11-19T18:03:00Z

---

Parecer do Advisor (Modo A, implementação 5c863ee/52248c1/ae09b13): REPROVADO COM RESSALVAS. Nenhuma falha crítica de corretude; a 008 cumpre o plano aprovado (índice único completo da tripla; event_id PK com DEFAULT randomblob, UNIQUE(profile_id,event_id) e origin legacy; recordings com sequência própria; occurrences AUTOINCREMENT com id=NEW.id estrito; guardas até o corte; guardas de outbox/revisão mantidas; asserções TEMP com FK check; checagem reversa; limites v8; dependência da 162; teste de falha com FS do goose). Ordem de colunas do ledger idêntica à 004.
Ressalvas antes do ensaio: (R1) 008:119-121 cria FKs ON DELETE SET NULL em ingest_events.occurrence_id/recording_id sem índice filho; cada exclusão/merge de faixa e cada gravação órfã varre ingest_events inteiro sob lock de escrita (a sombra da 171 acumula recibos). Criar índices parciais WHERE ... IS NOT NULL. (R2) Desvio do plano: 008:21-24 copia o MBID declarado de tracks para recordings.musicbrainz_id sem proveniência, e doc-039:42 define essa coluna como MBID validado. Decidir agora: coluna de proveniência com DEFAULT legacy na própria 008 (tabela nova, custo zero) ou entrada obrigatória na 134 antes da regra 3 do resolvedor.
Entradas para 134/135 (adicionáveis sem rebuild): coerência occurrence_id=listens.track_id e recording_id=occurrences.recording_id em recibos vinculados, identity_status coerente com accepted/duplicate e resolution_version monotônica (hoje só convenção); título/créditos apresentados da ocorrência sem tracks (doc-039:44); merge de gravações reversível para colisões de MBID legadas.
Desvios menores: checagem reversa só para origem >=7 (ingest_schema.go:53; v3-v6 descartam objetos extras de listens em silêncio); sem evidência de kill de processo, disco cheio e bootstrap concorrente; caminho RAISE de 008:272 não exercitado.
Melhoria opcional: comparar listens com listens_new antes do DROP elimina a cópia TEMP integral (008:56) da maior tabela.
Pré-condições do ensaio: snapshot indicado e autorizado; serviço e ferramentas externas parados; cópia quiescente DB/WAL/SHM em diretório privado com hashes da origem antes/depois; espaço >=3x medido no diretório do banco E no diretório temporário do SQLite (TMPDIR); inventário só leitura na cópia (versão 7, outbox/revisões vazias, objetos extras em listens/tracks/ledger, integrity e FK ok); R1/R2 resolvidos; medir tempo, lock, pico de disco (DB+WAL+temp+backup) e crescimento/freelist; segundo boot; restauração do .pre-recording-identity com v0.42; evidência só agregada.
Antes do aceite: nota de operador no CHANGELOG (upgrade irreversível, binário anterior recusa v8, backup, janela e espaço), restauração perde escritas pós-upgrade e o relay já entregue não volta. Entrada da purga de sombra sem event_id já registrada na ISSUE-081.

---

author: reviewer
created: 2026-11-19T18:04:00Z

---

Parecer do Reviewer (leitura estática, sem testes): APROVADO COM RESSALVAS. Sem caminho de perda/corrupção no upgrade; filho-primeiro com FK ON, asserções TEMP CHECK(ok=1), cópia 1:1 de listens e ledger idêntico à 004 conferidos. Ressalvas: (1) 008 removeu os CHECKs de inatividade de ingest_events da 004 sem guarda removível substituta (trg_ingest_event_inactive); (2) faltam índices em ingest_events.occurrence_id/recording_id (varredura do ledger a cada exclusão de faixa); (3) checagem reversa só para origem v7; (4) dois alocadores de ID tracks/occurrences colidem pós-corte (134/135); (5) coerência recibo↔evento↔ocorrência sem trigger (081); (6) AC6/AC7 provados só por primitivas SQL; (7) origin sem enum; (8) avaliar linha no CHANGELOG. Detalhes em PARECER-REVIEWER.md.

---

author: ebrgr
created: 2026-11-19T19:06:00Z

---

Decisão em 2026-11-19: opção B para listens.origin. Criar event_origins e cadastrar somente legacy antes de reconstruir listens. Usar FK sem cascata, DEFAULT legacy e cadastrar por INSERT as origens definidas e validadas pela ISSUE-134. Aplicar os pareceres na 008 ainda inédita e refazer o ensaio em cópias sintéticas.

---
<!-- COMMENTS:END -->
