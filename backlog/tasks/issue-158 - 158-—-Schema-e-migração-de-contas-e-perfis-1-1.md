---
id: ISSUE-158
title: 158 — Schema e migração de contas e perfis 1:1
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-18T21:25:00Z'
labels:
  - banco
  - admin
milestone: m-10
dependencies:
  - ISSUE-100
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-158/PRE-IMPLEMENTACAO.md
  - backlog/evidencias/issue-158/VERIFICACAO.md
  - backlog/evidencias/issue-158/migracoes-agregadas.json
  - backlog/evidencias/issue-158/PARECER-REVIEWER.md
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
- [x] #1 FK ON DELETE RESTRICT; dono obrigatório e único por ADD COLUMN + índice UNIQUE + triggers (equivalência funcional a NOT NULL UNIQUE aprovada pelo Advisor, sem reconstruir os pais); require_login=0 para perfis novos e legados; cardinalidade 1:1 verificada
- [x] #2 Admin, hashes, IDs, escutas, chaves de API e vínculos de relay preservados; trigger legado removido; bootstrap atômico
- [x] #3 Migração testada em cópia de banco real, com rollback em falha e restauração de backup comprovada; nenhuma escuta perdida
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-18T18:19:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-158` na branch `feat/issue-158-schema-e-migracao-de-contas` criada a partir de `main`.

---

author: codex
created: 2026-11-18T18:21:00Z

---

Pré-implementação: AGENTS/CLAUDE, doc-041, decision-010, plano ai-memory 01a1ea88-202b-7aa1-ad3b-59be658fbb6b e grafo conferidos. ISSUE-080 publicou v4; ISSUE-133 aberta sem versão reservada. Dúvida submetida ao usuário para revisão do Advisor: coordenação da v5 para 158 e inclusão das adaptações mínimas de criação/exclusão transacional da dupla conta/perfil e preservação do último admin, pois o contrato atual de exclusão conserva conta sem perfil. Evidência PRE-IMPLEMENTACAO.md. Nenhum SQL escrito, AC marcado ou banco real aberto. Mantido Em andamento.

---

author: advisor
created: 2026-11-18T18:30:00Z

---

Parecer do Advisor (Modo A, análise estática em 18/11/2026, main dff68d6): REPROVADO COM RESSALVAS. A direção da PRE-IMPLEMENTACAO está correta; seguir com os ajustes abaixo. Plano completo no ai-memory: [PLANO ISSUE-158] (plans/issue-158-advisor-schema-contas.md).

1. Versão: APROVADO v5 (005_accounts_profiles.sql). Existem só 001–004; o plano da 080 já previa "133 e 158 escolhem a próxima disponível". Registrar comentário na ISSUE-133 (próxima livre, v6 ou posterior). Ajustes obrigatórios: bootstrap.go:124 (limite 4→5), :159 (aceitar v5), :170 (v5 exige ausência de trg_legacy_user_profile), :162 (legacy_user_profiles só v3/v4), :263 (version < 4 → < 5; sem isso uma base v4 nunca migra), backup.go:51 (nome .pre-accounts-* para v4), ingest_schema.go:20 (incluir 005), bootstrap_recovery_test.go:296 (usa 5 como versão desconhecida → 6).

2. CRÍTICO — não recriar listening_profiles nem users. O DSN liga foreign_keys(ON) (bootstrap.go:105) e o goose roda em transação, onde PRAGMA foreign_keys é ignorado. DROP TABLE do pai executa DELETE implícito e dispara ON DELETE CASCADE em listens, api_keys, profile_relays e nas tabelas v4 (004:10,24,159): apaga todas as escutas. RENAME do pai reescreve as FKs dos filhos para o nome antigo. Usar ALTER TABLE ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE RESTRICT (default NULL é permitido), UNIQUE INDEX, triggers BEFORE INSERT/UPDATE que bloqueiam user_id NULL ou alterado e checagem de cardinalidade em checkDatabase. É equivalente a NOT NULL UNIQUE; registrar a equivalência no AC1. require_login entra por ADD COLUMN NOT NULL DEFAULT 0 CHECK.

3. Exclusão: cabe na 158, no nível da persistência. Sem isso o AC1 cai no primeiro DELETE após a migração. ON DELETE RESTRICT protege a direção perfil→conta e não impede excluir o perfil; um trigger bloqueando o DELETE do perfil impediria qualquer exclusão (mudança visível maior). Não existe alternativa sem mudança visível. DeleteListeningProfile apaga perfil e conta na mesma transação, com escrita primeiro e DELETE de users condicionado a role<>'admin' OR admins>1; sessões caem por CASCADE (001:80). Acrescentar trigger de último admin (BEFORE DELETE e UPDATE OF role em users). Ajustar adminui_catalog_review_test.go:135 para recusar (409) o perfil do último admin, mantendo conta e sessão. Telas, cópia e confirmação ficam para 160/162. Fazer CHANGELOG de uma linha.

4. CreateListeningProfile (profile_store.go:165): conta pendente, perfil e chave na mesma transação, com a conta antes (user_id NOT NULL). Username = slug[:32] (validateUsername limita a 32, user.go:27; o slug vai até 64). Em colisão, sufixo -2..-9 determinístico; se esgotar, ErrConflict.

5. Bootstrap (engine.go:148): hoje ignora o erro de CountUsers; uma falha leva a userCount=0 e cria admin numa base com contas. O risco de concorrência é baixo (instance_lock), mas o AC exige atomicidade. Criar Sqlite.BootstrapAdministrator: bcrypt fora da transação; INSERT ... WHERE NOT EXISTS (SELECT 1 FROM users) como primeira escrita; perfil 'default' e chave 'Default'; propagar erros. Base com contas e sem admin: não criar admin; logar orientação de recuperação local, que fica fora da 158. Com o trigger removido, SaveUser precisa criar conta e perfil numa transação, repetindo a regra do trigger (default/user-<id>).

6. Ponte legada: profiles.go:24 e profile_store.go:132 passam a ler listening_profiles.user_id. legacy_relay_binding já gravado é preservado (count>0 retorna). A migração copia legacy_user_profiles para user_id e depois faz DROP da ponte (é tabela filha; não cascateia).

7. Pendente: users.status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','pending') AND ((status='pending') = (length(password)=0))) via ADD COLUMN, que valida as linhas existentes. Pendentes ficam com password=X'' e role user. Login rejeita pendente de forma explícita, antes do bcrypt (auth.go:74). SaveAuthenticatedSession (sessions.go:73) passa a exigir status='active', porque hoje password=? casaria hash vazio com hash vazio. UpdateUser com senha marca active.

8. Migração SQL em uma transação do goose, nesta ordem: DROP TRIGGER legado; ADD COLUMNs; user_id pela ponte; perfil para conta sem perfil (default se não houver perfis, senão user-<id>); plano temporário de usernames (slug[:32] ou profile-<id>) com UNIQUE; inserir pendentes; vincular; guarda via TEMP trigger (idioma do 003 Down) para user_id NULL ou conta sem perfil; índices e triggers; DROP ponte; DROP TEMP. Nada pode abortar em base vazia (instalação nova e referência de validateSchemaLayout). Base com contas e sem admin: migrar sem promover. A falha real a testar é a colisão de username: base intacta em v4 e reinício após a correção.

9. Falha do engine na baseline: ambiental. engine_test.go:86 roda engine.Run numa goroutine e descarta o erro; connection refused em :63409 indica que o servidor não subiu ou que o sandbox bloqueia loopback. Não há código alterado e sqlite/admin passam. Rodar make check fora do sandbox (quem despachou). Opcional, só no teste: propagar o erro de Run por canal.

Testes mínimos: v3/v4/legado/vazio; WAL; backup/restore; rollback por colisão; reinício; FK/UNIQUE/NULL/imutável/último admin (incluindo concorrência); contagens e IDs de escutas e chaves por perfil; hashes e sessões do admin preservados; pendente não loga; 25 INSERTs diretos em fixtures passam a usar um helper com user_id. Sem push/merge/publish/Done.

---

author: codex
created: 2026-11-18T19:15:00Z

---

Verificação: implementada v5 aditiva conforme parecer/aceite; AC1 registra equivalência funcional NOT NULL UNIQUE com índice e triggers sem reconstruir pais. make check passou (Go, UI/referência e 13/13 e2e); go test -race ./internal/db/sqlite passou; teste extra de layout incompleto passou. Cópias reais v3: P1 80→80 e P1/P2 79/1→79/1; cópia privada com 004 publicada: v4→v5, 80→80. IDs, hashes, sessões, chaves e relay preservados; origens físicas inalteradas; backup/restore, rollback real por colisão e reinício comprovados. Defaults originais admin/changeme mantidos por instrução do usuário; v2 vazia cria admin active com perfil/chave e v2 com contas não cria extra. Três ACs comprovados; relatório e JSON anexados. Mantido Em andamento para Reviewer; sem push/merge/publish/Done.

---

author: reviewer
created: 2026-11-18T21:11:00Z

---

Parecer do Reviewer: APROVADO COM RESSALVAS. Nenhum BUG confirmado. RISCO 1: slug de fallback 'user-<id>' pode colidir com perfil existente (accounts.go:26 e 005:13), abortando SaveUser/migração. RISCO 2: login pendente derivado do slug sem validateUsername (accounts.go:36). Detalhes em backlog/evidencias/issue-158/PARECER-REVIEWER.md.

---

author: codex
created: 2026-11-18T21:20:00Z

---

Correções dos dois riscos do Reviewer concluídas a pedido do usuário, sem re-delegação. SaveUser/Bootstrap usam slug livre com sufixos -2..-9 e ErrConflict tipado com rollback se esgotados. Na 005 ainda não publicada, N+1 candidatos garantem slug livre para N perfis existentes, inclusive faixa curta ocupada, sem bloquear o upgrade por esse motivo. Logins pendentes explicitam lowercase e truncamento reservando o sufixo; cada candidato passa por validateUsername, e o plano SQL valida a mesma projeção ASCII/32. Testes novos de colisão em SaveUser/migração, esgotamento, slugs de 64 caracteres e hífens passaram. make check passou (13/13 e2e e referência idêntica); go test -race ./internal/db/sqlite passou. Revalidação em cópias v4 (80) e v3 (79+1): dados, backup/restore e reinício preservados, origens inalteradas. Evidências atualizadas em VERIFICACAO.md e JSON; parecer original incluído. Defaults admin/changeme configuráveis, equivalência NOT NULL via triggers e tabelas pai preservados. Mantido Em andamento, sem push/merge/publish/Done.

---

author: advisor
created: 2026-11-18T21:22:00Z

---

Parecer do Advisor (2a passagem, d414e9f): APROVADO. Riscos do Reviewer resolvidos: (a) slug de fallback livre em SaveUser/BootstrapAdministrator (accounts.go insertAccountProfile, -2..-9, ErrConflict sem conta orfa) e na 005 (N+1 candidatos; bases user-<id> nao se sobrepoem, logo a 005 nao aborta por slug; 005 ausente de qualquer tag publicada, v0.35 incluida, e da main); (b) login pendente limitado a 32, minusculo e validado por validateUsername no Go e por CHECK equivalente em account_plan no SQL. Ponto critico do 1o parecer intacto: sem DROP/RENAME de users/listening_profiles, transacao unica, base vazia ok, rollback em v4. ErrConflict->409 e ErrInvalidInput->400 via profile_store.go:171 e adminweb/errors.go. Testes novos passam (rodados). Ressalvas nao bloqueantes: 005:33 o fallback 'profile-<id>' nao e checado contra usuarios existentes nem contra outro slug literal 'profile-<id>' -> aborta com rollback seguro (caso aceito no 1o parecer; esquema diverge do Go, que usa sufixos -2..-9, ambos validos); SaveUser nao tem chamador em runtime; CHANGELOG [Unreleased] vai conflitar com a v0.35 no rebase. Pode ir a aceite.

---

author: claude
created: 2026-11-18T21:23:00Z

---

Aceite explícito do usuário em 2026-11-18, incluindo o desvio do AC1 (user_id sem NOT NULL literal, imposto por UNIQUE INDEX, triggers e checkDatabase) e o admin padrão admin/changeme mantido. Pareceres: Advisor (plano e 2ª passagem APROVADO) e Reviewer (APROVADO COM RESSALVAS, riscos corrigidos em d414e9f).

---

author: ebrgr
created: 2026-11-18T21:25:00Z

---

Publicação: versão 0.36 (tag `v0.36`), branch `feat/issue-158-schema-e-migracao-de-contas` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
