---
id: ISSUE-160
title: 160 — Autorização de gestão própria e administração global
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T15:11:00Z'
labels:
  - admin
  - api
milestone: m-10
dependencies:
  - ISSUE-158
  - ISSUE-159
references:
  - backlog/decisions/decision-010
  - backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md
  - backlog/evidencias/issue-160/VERIFICACAO.md
  - backlog/evidencias/issue-160/REVIEWER.md
  - backlog/evidencias/issue-160/PARECER-REVIEWER.md
  - backlog/evidencias/issue-160/REVISAO-CORRECOES.md
documentation:
  - doc-041
priority: high
type: feature
ordinal: 173000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: ISSUE-100 (aceita em 2026-11-17). Contrato normativo: doc-041 e decision-010. Padrão de senhas atual mantido (bcrypt e validação existentes), sem novos limites ou bloqueios por usuário ou origem; o limitador de login compartilhado existente fica como está; perfis existentes e novos nascem públicos (require_login=0); o relay segue ativo mesmo com escutas protegidas (consequência aceita). Escopo: o dono maneja somente os seus dados, chaves, importação e exportação; root maneja todos os perfis e o servidor; catálogo compartilhado (capas, aliases, merges) só root; papel lido no servidor a cada requisição; último admin ativo protegido em transação; sem mass assignment de role, status ou dono.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Matriz de permissões do doc-041 coberta por testes (visitante, usuário, root) em HTML, JSON e serviços
- [x] #2 Último admin protegido em transação, com teste concorrente; usuário não se promove
- [x] #3 Catálogo compartilhado exclusivo do root; chave de ingestão não administra nem lê dados privados
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T00:05:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-160` na branch `feat/issue-160-autorizacao-de-gestao-propria-e` criada a partir de `main`.

---

author: codex
created: 2026-11-19T00:07:00Z

---

Verificação parcial: análise estática identificou ambiguidade entre a importação própria permitida no doc-041 §2 e a criação de capas/aliases compartilhados pelo ImportKoitoFile (internal/importer/koito.go:105,141,184). Execução interrompida conforme instrução explícita, aguardando decisão registrada em backlog/evidencias/issue-160/AMBIGUIDADE-IMPORTACAO.md. Nenhum runtime alterado, nenhum AC marcado; make check e race não executados. Mantido Em andamento.

---

author: codex
created: 2026-11-19T00:20:00Z

---

Decisão do usuário: opção 2. Importação própria pode criar entidades novas com capas e aliases; alterações editoriais de entidades existentes continuam exclusivas do root. Implementação retomada na mesma worktree.

---

author: codex
created: 2026-11-19T00:36:00Z

---

Verificação: política central de gestão aplicada em HTML, JSON e serviços; dono restrito ao próprio perfil, catálogo/servidor/contas exclusivos root; serviços reconsultam sessão e rejeitam API key/role obsoleto; mass assignment negado. Migração v6 protege último admin ativo e serviço de acesso revalida root dentro da transação, com concorrência em duas conexões para excluir/rebaixar/desativar. Opção 2 da importação testada. make check passou (13/13 E2E); race nos pacotes tocados passou com timeout 10m, e casos finais 160 repetidos com race. Graphify AST atualizado nesta worktree, diff/deslop e revisão local APROVADO. ACs comprovados no escopo de gestão da 160; gate de leitura require_login/imagens/cache (161), fluxos de token (162) e navegação/destino (163/157) permanecem fora do escopo. Evidências: VERIFICACAO.md e REVIEWER.md. Mantido Em andamento, sem push/merge/publish/Done.

---

author: reviewer
created: 2026-11-19T10:30:00Z

---

Parecer Reviewer: APROVADO COM RESSALVAS, sem BUG de autorização. A política central nega por padrão em HTML, JSON e serviços, o IDOR e o mass assignment estão fechados e o último admin ativo é protegido com o lock de escrita na 1ª instrução (migração 006 nova, fora de tag). Ressalvas: (1) a opção 2 expõe SSRF cego via image_url importado por não-root e a pré-semeadura de MBID/aliases no catálogo compartilhado; a exceção não está no doc-041, só no comentário #3 do codex. (2) Após uma ação HTML pessoal, o redirect /admin?profile=N leva o dono ao 403. (3) O teste concorrente não usa WAL. Recomenda-se o Advisor para a opção 2 e a migração. Detalhes em PARECER-REVIEWER.md.

---

author: ebrgr
created: 2026-11-19T10:47:00Z

---

Decisão do usuário (19/11/2026, reafirmada ao Claude e ao executor): a importação pelo usuário comum, opção 2 de AMBIGUIDADE-IMPORTACAO.md, DEVE ser mantida; limitar demais tiraria funcionalidade do não-root. Confirmo a exceção para importar entidades novas com imagens e aliases, mantendo alterações editoriais de entidades existentes exclusivas do root e aplicando as mitigações solicitadas para image_url e pré-semeadura. Esta confirmação não é aceite da implementação nem autorização de publicação.

---

author: codex
created: 2026-11-19T15:01:00Z

---

Correções pós-Reviewer verificadas: confirmação da opção 2 registrada com autoria ebrgr e no doc-041 §2, com os riscos e mitigações. A: origem de imagem importada por não-root persistida; fetch público valida DNS, fixa IP e confere peer, sem proxy, revalida redirects e mantém limites de tempo/tamanho; fontes locais root preservadas. B: aliases usam proveniência por destino no campo source existente, MBIDs desconhecidos não são gravados e enriquecimento automático não promove a entidade; imagens/aliases/histórico e reimportação preservados, sem novo schema. C: redirect /admin?profile=N autorizado como profiles, com aviso de sucesso testado. D: concorrência do último admin em WAL, com asserção do modo. E: tolerância de campos extras mantida por compatibilidade; destino permanece do servidor. make check passou (13/13 E2E); race completo nos pacotes afetados e casos finais passaram. Graphify AST atualizado, diff/deslop e revisão diferencial pelo executor APROVADO, sem re-delegar. Parecer e comentário originais do Reviewer preservados e incluídos no commit. Migração 006 intacta, sem 007, sem mudanças na 159/limites de login; Em andamento, sem push/merge/publish/Done. Evidências detalhadas em VERIFICACAO.md.

---

author: advisor
created: 2026-11-19T15:09:00Z

---

Parecer do Advisor (Modo A, por leitura, sem rodar testes): APROVADO. Sem falhas críticas. (1) SSRF: imagefetch/public.go resolve uma vez, valida todos os IPs, disca o IP literal e confere o peer; sem proxy; redirect revalidado pelo mesmo transport (máx. 3, 10 s, 10 MiB); a marca #tunetrail-public-only é aplicada pelo servidor a toda URL de import não-root (koito.go:92-97) antes de qualquer uso e acompanha image_source até imagecache/automatic_image; o dono não consegue retirá-la, e o root forjá-la só restringe. Formas alternativas (decimal/octal) caem na validação do IP resolvido. (2) Pré-semeadura: MBID descartado em Save*, promoção bloqueada em Update* sem WithCatalogEditorial (só root), aliases ProfileImport:N filtrados em GetArtist/GetAlbum/releaseByArtistAndTitles/trackByTrackInfo, proveniência herdada no enriquecimento; IDs de perfil são AUTOINCREMENT, então não há reuso de escopo. (3) Migração 006 só recria triggers dentro da transação do goose, Down bloqueado de propósito, e não está em nenhuma tag; UpdateAccountAccess começa com UPDATE (lock de escrita) que revalida o ator, então rebaixamento mútuo em paralelo falha com ErrConflict. (4) AuthorizeService reautentica a cada chamada; API key (SessionID nulo) e pending recebem negação. (5) Commits ISSUE-160, CHANGELOG, comentários em inglês e doc-041 §2 conferem. Melhorias opcionais: bloquear também 192.0.0.0/24 (metadata OCI legada 192.0.0.192), 192.88.99.0/24 e TEST-NETs em blockedPublicPrefixes; DROP TRIGGER IF EXISTS na 006; doc-041 registrar como risco residual que scrobbles e outros importadores do dono (não Koito) continuam gravando aliases e MBIDs globais, como antes. Atenção: a branch parte da v0.37 (8bbccb4), não da v0.38; é preciso fazer rebase sobre a main (ISSUE-075/169) antes do publish. Pode ir a aceite.

---

author: claude
created: 2026-11-19T15:10:00Z

---

Aceite explícito do usuário em 2026-11-19, incluindo a exceção da importação por dono (doc-041 §2) já confirmada por ele. Pareceres: Reviewer (APROVADO COM RESSALVAS; SSRF, pré-semeadura, redirect do dono e teste em WAL corrigidos em 9de0bd6) e Advisor (APROVADO, effort medium). Melhorias opcionais do Advisor (bloqueios 192.0.0.0/24, 192.88.99.0/24 e TEST-NETs; DROP TRIGGER IF EXISTS na 006; risco residual de aliases/MBIDs de importadores não-Koito no doc-041) ficam registradas, sem bloquear.

---

author: ebrgr
created: 2026-11-19T15:11:00Z

---

Publicação: versão 0.39 (tag `v0.39`), branch `feat/issue-160-autorizacao-de-gestao-propria-e` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
