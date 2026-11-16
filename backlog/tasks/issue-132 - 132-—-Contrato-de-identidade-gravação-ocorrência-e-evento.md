---
id: ISSUE-132
title: '132 — Contrato de identidade: gravação, ocorrência e evento'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-16T17:06:00Z'
labels:
  - catalogo
  - banco
milestone: m-11
dependencies:
  - ISSUE-131
references:
  - backlog/evidencias/issue-131/verificacao.md
  - backlog/evidencias/issue-132/fixture-identidade.sql
  - backlog/evidencias/issue-132/resultado-fixture.txt
  - backlog/evidencias/issue-132/parecer-reviewer.md
  - >-
    backlog/decisions/decision-008 -
    Identidade-musical-separada-em-gravação-ocorrência-e-evento.md
documentation:
  - doc-036
  - doc-039
type: spike
ordinal: 145000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 2 do estudo da ISSUE-116 (doc-036). Definir o contrato de identidade e de estatística (gravação, ocorrência num lançamento, evento), alinhado às ISSUE-079, 080, 081 e 092, antes de qualquer migração. Inclui a decisão do usuário: ranking por gravação como padrão, com opção por lançamento.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fixtures distinguem gravação, ocorrência e evento
- [x] #2 Decididos: álbum ausente, colisão no mesmo segundo e revisão
- [x] #3 Exemplo A/B: 2 escutas na gravação e 1 em cada álbum, sem vazamento entre dois perfis
- [x] #4 Decisão registrada em backlog/decisions após aceite
- [x] #5 Contrato define preservação durável e revisão/reprocessamento de eventos com alias ambíguo ou MBIDs conflitantes, sem associação silenciosa, com fixtures e isolamento por perfil
- [x] #6 Contrato define a resposta da API ListenBrainz para conflitos e a continuidade/retomada da importação Last.fm, incluindo prevenção de perda e duplicação de escutas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T16:26:00Z

---

Pendências encaminhadas pelo usuário após o aceite da ISSUE-131 em 16/11/2026: atualmente um alias ambíguo ou MBIDs incompatíveis recusam a associação e a escuta não é salva; a importação Last.fm é interrompida e o ListenBrainz recebe HTTP 500 pelo tratamento atual. Esta spike deve definir o contrato de preservação durável do evento e metadados originais, motivo da pendência, revisão e reprocessamento idempotente por perfil; definir também a resposta da API e a continuidade/retomada da importação. Registrar fixtures para esses casos e distribuir a implementação nas ISSUE-133/134 ou tarefas próprias conforme o contrato aprovado, antes de qualquer migração. A correção da ISSUE-131 permanece sem migração e sem fila de pendências.

---

author: ebrgr
created: 2026-11-16T16:28:00Z

---

Retomada na branch `spike/issue-132-contrato-de-identidade-gravacao-ocorrencia-e-evento`.

---

author: advisor
created: 2026-11-16T16:28:00Z

---

Estudo Advisor iniciado sobre a base 75b1aff, que inclui a ISSUE-131 aceita mas ainda não publicada. Branch de estudo derivada dessa base para não perder a correção nem as pendências encaminhadas. Análise direta, sem delegação; nenhuma alteração de runtime ou migração prevista nesta spike.

---

author: advisor
created: 2026-11-16T16:32:00Z

---

Advisor: estudo doc-039 concluído sobre 75b1aff. Parecer APROVADO para planejamento, sujeito ao aceite das propostas. Contrato separa gravação/ocorrência/evento/recebimento; define álbum ausente sem atribuição inventada, colisões no mesmo segundo, revisão por perfil e reprocessamento idempotente, ACK ListenBrainz após recebimento durável e retomada Last.fm por digest/ordinal. Fixture conceitual SQLite: 21 verificações verdes (A/B por perfil, repetições, pendências, resolução, rollback e isolamento). Não prova implementação de transporte, migração ou concorrência real. Riscos: sem ID confiável não há exatamente uma vez automático; retenção/payload continuam na ISSUE-079; alinhar mudanças de erro com ISSUE-081 e total sem lançamento com ISSUE-135. Critérios 2 e 4 aguardam aceite do contrato e registro posterior da decisão accepted. Nenhum runtime alterado; make check não repetido por se tratar somente de documentação/fixture isolada.

---

author: codex
created: 2026-11-16T16:32:00Z

---

Reviewer: APROVADO. Revisão direta dos artefatos do estudo e deslop; nenhuma mudança de runtime. A fixture é conceitual e suas limitações estão explícitas; não é migração de produção. Contrato documenta tentativas com mesmo ID e conteúdo divergente sem sobrescrever snapshot. Verificação: 21 invariantes SQLite passaram; git diff --check sem erros.

---

author: advisor
created: 2026-11-16T16:32:00Z

---

Verificação: doc-039 e fixture conceitual concluídos; 21 verificações passaram. Estudo disponível para aceite das propostas sobre álbum ausente, colisões e revisão; retenção/payload dependem da ISSUE-079. Nenhuma decisão de arquitetura marcada accepted; critério 4 será cumprido após aceite explícito.

---

author: codex
created: 2026-11-16T17:06:00Z

---

Aceite: aprovada pelo usuário em 16/11/2026. Contrato doc-039 aceito e formalizado na decision-008 (accepted); álbum ausente, colisões e revisão definidos. Todos os critérios cumpridos. Implementação encaminhada às tarefas dependentes; ISSUE-079 mantém retenção/payload pendentes. Sem alteração de runtime ou publicação.

---
<!-- COMMENTS:END -->
