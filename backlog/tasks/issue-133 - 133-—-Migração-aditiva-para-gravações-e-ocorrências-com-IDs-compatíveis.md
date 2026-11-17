---
id: ISSUE-133
title: 133 — Migração aditiva para gravações e ocorrências com IDs compatíveis
status: Aberta
assignee: []
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-17T16:18:00Z'
labels:
  - banco
milestone: m-11
dependencies:
  - ISSUE-132
  - ISSUE-080
references:
  - backlog/decisions/decision-009
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
<!-- COMMENTS:END -->
