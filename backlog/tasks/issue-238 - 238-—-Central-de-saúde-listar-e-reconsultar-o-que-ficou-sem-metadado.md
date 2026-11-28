---
id: ISSUE-238
title: '238 — Central de saúde: listar e reconsultar o que ficou sem metadado'
status: Aberta
assignee: []
created_date: '2026-11-28T02:22:00Z'
updated_date: '2026-11-28T02:22:00Z'
labels:
  - admin
  - catalogo
  - ui
milestone: m-13
dependencies:
  - ISSUE-217
type: feature
ordinal: 251000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 27/11/2026 sobre a ISSUE-217: o que o Navidrome e o MusicBrainz responderem não ter não é consultado de novo, salvo por outro serviço disponível, e fica registrado no banco para a Central de saúde reconsultar depois. A ISSUE-217 cria o registro (migração 027: uma marca por perfil, entidade, fonte e instância para resposta definitiva sem dado). Esta tarefa lista, por perfil, as faixas, lançamentos e artistas sem dado em todas as fontes disponíveis, calculado na leitura contra o registro atual de fontes, e permite reconsultá-los: apagar a marca e enfileirar de novo. Serviço habilitado depois já é consultado sozinho na escuta seguinte; a reconsulta atende o que não voltará a ser ouvido e o caso de a música ser acrescentada depois à biblioteca do Navidrome. O reenfileiramento do histórico, os limites por provedor e a prioridade na fila compartilhada são da ISSUE-213; coordenar com ela.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Listar na central de saúde, por perfil, faixas, lançamentos e artistas sem dado em todas as fontes disponíveis, respeitando permissões e isolamento de perfis
- [ ] #2 Reconsultar um item ou um conjunto apaga as marcas escolhidas e enfileira a consulta sem duplicar trabalho, com teste usando fontes simuladas
- [ ] #3 Fonte habilitada depois aparece como disponível para a reconsulta
- [ ] #4 Consultas da listagem apoiadas em índice, medidas com uma biblioteca grande fictícia; interface verificada em en e pt-BR
<!-- AC:END -->
