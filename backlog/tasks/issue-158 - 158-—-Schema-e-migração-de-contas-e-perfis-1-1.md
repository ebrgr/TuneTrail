---
id: ISSUE-158
title: 158 — Schema e migração de contas e perfis 1:1
status: Aberta
assignee: []
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-17T17:33:00Z'
labels:
  - banco
  - admin
milestone: m-10
dependencies:
  - ISSUE-100
references:
  - backlog/decisions/decision-010
documentation:
  - doc-041
priority: high
type: feature
ordinal: 171000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: listening_profiles.user_id NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT e listening_profiles.require_login (padrão 0), um usuário por perfil existente (senha temporária ou token de ativação), perfis e escutas e chaves de API e vínculos de relay preservados, trigger legado removido, bootstrap atômico do primeiro admin, coordenação da numeração do schema com as ISSUE-080 e 133 (sem impor o pipeline inteiro), backup e restauração antes de migrar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 FK NOT NULL UNIQUE e require_login=0 para perfis novos e legados; cardinalidade 1:1 verificada
- [ ] #2 Admin, hashes, IDs, escutas, chaves de API e vínculos de relay preservados; trigger legado removido; bootstrap atômico
- [ ] #3 Migração testada em cópia de banco real, com rollback em falha e restauração de backup comprovada; nenhuma escuta perdida
<!-- AC:END -->
