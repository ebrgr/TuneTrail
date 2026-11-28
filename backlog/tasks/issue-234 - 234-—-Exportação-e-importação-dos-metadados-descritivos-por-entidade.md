---
id: ISSUE-234
title: 234 — Exportação e importação dos metadados descritivos por entidade
status: Aberta
assignee: []
created_date: '2026-11-27T21:48:00Z'
updated_date: '2026-11-28T02:22:00Z'
labels:
  - catalogo
  - operacao
milestone: m-13
dependencies:
  - ISSUE-218
priority: medium
type: feature
ordinal: 247000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: limitação encontrada na implementação da ISSUE-218 e pedido do usuário em 27/11/2026 para abrir a issue. Os metadados descritivos (gênero, tag, ano, datas, país de origem, ISRC e BPM) passam a ficar em tabelas próprias, por perfil e entidade, com um cadastro de fontes (decision-012). O formato de exportação atual só leva declarações de MBID (internal/db/sqlite/export_enrichment.go), portanto restaurar um perfil a partir do arquivo de exportação não recupera esses metadados; até esta issue, só a cópia do banco os preserva. Definir como levá-los no arquivo e importá-los: compatibilidade com exportações antigas, a instância da fonte nunca exposta (o arquivo já usa um número no lugar dela nas declarações de MBID), vínculo à faixa, ao lançamento e ao artista depois que as escutas forem recriadas (a gravação só é aceita para entidade sustentada por escuta do perfil) e reimportação sem duplicar valores. A mudança de formato precisa da aprovação do usuário antes de implementar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Formato de exportação com os metadados descritivos aprovado pelo usuário e registrado, compatível com arquivos antigos
- [ ] #2 Exportar e importar um perfil recupera os valores com a fonte, sem expor a instância, com teste de ida e volta
- [ ] #3 Reimportação não duplica valores e arquivos sem metadados continuam importando
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-28T02:22:00Z

---

27/11/2026: a ISSUE-217 cria marcas de resposta sem dado (migração 027). Decidir, com o formato, se elas entram na exportação ou se são estado operacional que não viaja.

---
<!-- COMMENTS:END -->
