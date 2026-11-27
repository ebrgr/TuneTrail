---
id: ISSUE-208
title: '208 — Recibos aceitos: por quanto tempo guardar o conteúdo original'
status: Aberta
assignee: []
created_date: '2026-11-24T00:51:00Z'
updated_date: '2026-11-27T15:21:00Z'
labels:
  - ingestao
  - banco
milestone: m-11
dependencies: []
references:
  - >-
    backlog/docs/estudos/doc-047 -
    Auditoria-de-uso-e-repetição-dos-dados-do-banco-schema-23.md
  - backlog/evidencias/db-auditoria/colunas.csv
priority: low
type: spike
ordinal: 221000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-047 §5 (nível 3) e R9. O original_snapshot (277 bytes) e o snapshot_digest (64 bytes) de cada recibo aceito somam 341 bytes por escuta e ficam para sempre; o conteúdo repete artista, título, lançamento e MBIDs que já estão no catálogo. Estudar a compactação depois de N dias: o que a página por que esta escuta está aqui (ISSUE-083) precisa do original, por quanto tempo a deduplicação depende do digest, o que a exportação e o relay leem, e o efeito na reprocessagem de um recibo. Entregar parecer com a janela proposta, a economia medida em 300 mil escutas e o impacto nas páginas. É uma troca de custo por funcionalidade: compactar apaga a prova de como a escuta chegou. Não implementar sem decisão do usuário (doc-047 §7, decisão 3).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Parecer com as dependências do original_snapshot e do digest (páginas, deduplicação, exportação, relay, reprocessagem)
- [ ] #2 Janela proposta em dias e a economia medida em 300 mil escutas
- [ ] #3 Decisão do usuário registrada; se aprovada, issue de implementação aberta
- [ ] #4 Definir o que a limpeza e a exclusão após o período configurado podem apagar, e o comportamento de cada leitor do conteúdo original depois disso
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026 (B4): em vez de guardar para sempre, haver a opção de limpar e excluir o conteúdo original depois de um período configurado. Este estudo passa a definir o que pode ser apagado e o efeito em cada leitor; a implementação é a ISSUE-225.

---
<!-- COMMENTS:END -->
