---
id: ISSUE-161
title: 161 — Gate de visibilidade das escutas, imagens e cache
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-17T17:33:00Z'
updated_date: '2026-11-19T16:34:00Z'
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
  - backlog/evidencias/issue-161/VERIFICACAO.md
  - backlog/evidencias/issue-161/PARECER-REVIEWER.md
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
- [x] #1 Perfil protegido não vaza em páginas, API, busca, widget, imagens, cache nem 304; visitante e usuário sem acesso recebem 401 ou 404 genéricos
- [x] #2 Diretório mostra todos os perfis; perfil público continua legível sem login; gestão sempre exige login
- [x] #3 Ingestão e relay de entrada e saída funcionam com perfil protegido (testes)
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-19T15:12:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-161` na branch `feat/issue-161-gate-de-visibilidade-das-escutas` criada a partir de `main`.

---

author: codex
created: 2026-11-19T15:14:00Z

---

Verificação preliminar: cache legado de capas permite reutilização por até 30 dias sem consultar a origem. Inventário inicial e decisão sobre o corte registrados em VERIFICACAO.md. Implementação pausada para decisão do usuário.

---

author: ebrgr
created: 2026-11-19T15:26:00Z

---

Decisão do usuário em 19/11/2026 — opção 1. Respostas legadas de /image/{image_id}/{filename} com Cache-Control public, max-age=2592000 são cópias já distribuídas. A origem autoriza todas as requisições antes de servir bytes ou 304. Imagens de perfil protegido usam private, no-store. Alternar require_login muda o validador da imagem. O usuário aceita que caches fora do controle do servidor possam reutilizar os cabeçalhos antigos por até 30 dias. A decisão dispensa corte operacional e purga de caches compartilhados.

---

author: codex
created: 2026-11-19T16:06:00Z

---

Verificação: matriz público/protegido por papel em HTML, JSON, busca, fragmentos, now-playing, exportação e imagens. GET e HEAD condicionais são autorizados antes de 304. Diretório público e gestão por sessão conferidos. Ingestão por chave e relay HTTP exercitados com perfil protegido. make check passou com 13/13 jornadas E2E, go test -race passou nos pacotes tocados e a referência determinística coincide. Revisão local aprovada. A tarefa segue Em andamento.

---

author: reviewer
created: 2026-11-19T16:23:00Z

---

Reviewer: APROVADO COM RESSALVAS, sem vazamento de privacidade. O gate ReadListening reaproveita a política da 160 em JSON, HTML, transicionais, legado, busca (CTE), exportação e imagens, antes de 304/ETag. A opção 1 está registrada. Riscos não bloqueantes: falta índice em releases/artists.image para a checagem por imagem, agora sem cache longo; 404 em texto puro na rota legada de perfil único; aria-label do cadeado; REVIEWER.md ambíguo; lacunas de teste (imagem legada do root, PATCH com campos extras). Advisor opcional (@Opus medium). Parecer: backlog/evidencias/issue-161/PARECER-REVIEWER.md

---

author: ebrgr
created: 2026-11-19T16:28:00Z

---

Decisão do usuário em 19/11/2026 após o parecer do Reviewer. Imagens públicas, incluindo capas e avatares, voltam a usar Cache-Control public, max-age=2592000 com ETag. Imagens de perfil protegido usam private, no-store. A origem autoriza toda requisição antes de bytes ou 304. O usuário aceita que, depois de marcar require_login, cópias públicas já servidas possam permanecer frescas por até 30 dias em navegadores e caches. A consulta de associação continua nas requisições à origem, sem novos índices ou migração 007. Por decisão do usuário, os riscos 2 a 5 seguem como pendências conhecidas: 404 da rota legada sem slug para o perfil único, acessibilidade do cadeado, ambiguidade de REVIEWER.md e lacunas de cobertura (imagem legada do root, PATCH com campos extras e rotas adicionais na comparação de negativas).

---

author: codex
created: 2026-11-19T16:32:00Z

---

Verificação do risco 1: cache público de 30 dias com ETag em capas de perfil público, avatares e capas da rota legada elegíveis ao público. Capas protegidas usam private, no-store. GET e HEAD condicionais de visitante e de usuário sem acesso retornaram 404, enquanto uma condicional autorizada recebeu 304. make check passou com 13/13 E2E e go test -race passou em engine e engine/handlers. Custo residual registrado em VERIFICACAO.md. Os riscos 2 a 5 seguem como pendências por decisão do usuário.

---

author: claude
created: 2026-11-19T16:33:00Z

---

Aceite explícito do usuário em 2026-11-19, incluindo a opção 1 para o cache legado de imagens, o cache público de 30 dias restaurado para imagens públicas (11400e8) e o risco residual documentado no doc-041 §3. Parecer do Reviewer (Opus, effort high): APROVADO COM RESSALVAS, sem vazamento de privacidade; o risco 1 (desempenho das capas) foi tratado com o max-age de 30 dias, sem migração nem índice, e o diff dessa correção foi revisado por leitura pelo Claude. Pendências conhecidas e NÃO implementadas por decisão do usuário: (2) rota antiga sem slug com perfil único protegido responde 404 em texto puro e sem Vary: Cookie; (3) aria-label do cadeado em span genérico em assets/templates/public/live.html; (4) arquivo backlog/evidencias/issue-161/REVIEWER.md é só autoavaliação do Codex; (5) lacunas de teste (imagem ao root pela URL antiga, PATCH com campos extras, 404 idêntico em mais rotas). Considerar na ISSUE-163.

---

author: ebrgr
created: 2026-11-19T16:34:00Z

---

Publicação: versão 0.40 (tag `v0.40`), branch `feat/issue-161-gate-de-visibilidade-das-escutas` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
