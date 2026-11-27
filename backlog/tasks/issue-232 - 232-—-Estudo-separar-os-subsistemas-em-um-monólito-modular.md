---
id: ISSUE-232
title: '232 — Estudo: separar os subsistemas em um monólito modular'
status: Aberta
assignee: []
created_date: '2026-11-27T15:45:00Z'
updated_date: '2026-11-27T15:45:00Z'
labels:
  - processo
  - banco
milestone: m-15
dependencies: []
references:
  - backlog/evidencias/issue-232/BASELINE.md
priority: medium
type: spike
ordinal: 245000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pergunta do usuário em 27/11/2026 sobre a separação dos subsistemas e levantamento feito no código da v0.79 (linhas por pacote e grafo de imports). Levantamento inicial em backlog/evidencias/ da própria issue. Quadro: 8 áreas de negócio mais a base; internal/db/sqlite tem 12.685 linhas (30% do Go de produção) e reúne ingestão, catálogo, relay, metadados, contas e infraestrutura; não há SQL fora dele e só engine e relayoutbox o importam, mas ele importa catalog, enrichment, relay, mbz e dedup. Estudar e propor: (3) interfaces pequenas declaradas por quem usa, no lugar de db.DB, que reúne 8 interfaces; (4) tirar os orquestradores do banco, começando pelos metadados (a lógica vai para internal/enrichment e o SQLite fica com consultas e transações), depois o relay e a resolução de escutas, tratando as transações e o bloqueio de escrita compartilhados e os fences do enriquecimento; (5) se e quando dividir internal/db/sqlite em subpacotes, o que hoje não é recomendado; e as dependências da interface web (webui importa adminui; adminui usa importer, mbz, relay e ingest; há funções de mais de 300 linhas). A mudança grande só começa depois das ISSUE-217, 218, 203 a 206 e 195, que mexem nos mesmos arquivos. Não implementa: entrega parecer, ordem, riscos e as issues de implementação propostas. Advisor recomendado pelo peso das transações e da concorrência.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mapa dos subsistemas com as dependências atuais e as desejadas, reproduzível por script
- [ ] #2 Proposta para a interface db.DB e para os orquestradores, com a fronteira de transação e de bloqueio de escrita descrita caso a caso
- [ ] #3 Ordem das mudanças, riscos e relação com as ISSUE-217, 218, 203 a 206 e 195
- [ ] #4 Parecer do Advisor registrado
- [ ] #5 Decisão do usuário registrada em backlog/decisions e issues de implementação abertas se aprovadas
<!-- AC:END -->
