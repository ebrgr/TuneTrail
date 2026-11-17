---
id: ISSUE-161
title: 161 — Gate de visibilidade das escutas, imagens e cache
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-17T17:33:00Z'
labels:
  - api
  - ui
  - imagens
milestone: m-10
dependencies:
  - ISSUE-159
  - ISSUE-160
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
priority: high
type: feature
ordinal: 174000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: diretório público em / com todos os perfis (nome, slug e avatar, sem contagens nem capas); opção Exigir login para ver as escutas por perfil; gate central antes de buscar ou serializar em páginas, API, busca, fragmentos, gráficos, now-playing, widget, exportação; rota de imagem com contexto de perfil e sem bypass do cache global; 304 e ETag só depois da autorização; cache privado para dados pessoais; alterar o checkbox invalida caches; ingestão e relay inalterados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Perfil protegido não vaza em páginas, API, busca, widget, imagens, cache nem 304; visitante e usuário sem acesso recebem 401 ou 404 genéricos
- [ ] #2 Diretório mostra todos os perfis; perfil público continua legível sem login; gestão sempre exige login
- [ ] #3 Ingestão e relay de entrada e saída funcionam com perfil protegido (testes)
<!-- AC:END -->
