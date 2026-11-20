---
id: ISSUE-134
title: 134 — Ingestão por ocorrência em todas as fontes
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-15T16:43:00Z'
updated_date: '2026-11-20T21:26:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-133
  - ISSUE-081
references:
  - backlog/decisions/decision-009
  - backlog/evidencias/issue-134/PLANO.md
  - backlog/evidencias/issue-134/ENSAIO-E3.md
documentation:
  - doc-036
  - doc-039
  - doc-040
type: feature
ordinal: 147000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Recorte 4 do estudo da ISSUE-116 (doc-036). Ingestão grava a ocorrência ouvida em todas as fontes (ListenBrainz, Last.fm, importadores). Contrato normativo: doc-040 e decision-009 (ISSUE-079, aceitas em 2026-11-17). Acrescenta (doc-040 §3–§5 e §10): adotar o envelope e o resolvedor comuns, junto da ISSUE-081; manter ocorrência ouvida, digest e ordinal, precisão temporal, Pending e retomada independente do staging; deduplicação conservadora (Igual, Diferente, Revisar) com janelas apenas para selecionar candidatos e evidência persistida; testar idempotência enquanto os recibos existem e a nova importação após a exclusão (evento novo, sem bloqueio), inclusive arquivos e lotes parcialmente excluídos: a chave de lote não bloqueia os itens excluídos e itens vivos reaproveitam seus recibos; coordenar a revisão por perfil com a ISSUE-083 e a outbox elegível com a ISSUE-082; sem ativação geral antes de ser possível resolver as pendências.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Mesmo MBID em A/B gera uma gravação e duas ocorrências
- [x] #2 Duas faixas 'Intro' com IDs ou posições diferentes continuam separadas
- [x] #3 Sem desambiguador, o evento é preservado para revisão
- [x] #4 Reimportação idempotente e repetição legítima testadas conforme a ISSUE-081
- [x] #5 Reimportação após a exclusão cria evento novo; arquivo ou lote parcialmente excluído reaproveita os recibos vivos e recria só os excluídos
- [x] #6 Ambiguidade não prova igualdade: candidatos indistinguíveis vão para Revisar com evidência e regra versionadas, sem apagar nem contar uma segunda escuta
- [x] #7 Fontes e importadores usam o envelope comum, com precisão temporal e retomada independente do staging
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: codex
created: 2026-11-16T17:06:00Z

---

Contrato da ISSUE-132 aceito (decision-008). Implementar recebimentos duráveis, conflitos preservados para revisão por perfil, idempotência por fonte/ID ou digest/ordinal, Pending na importação e retomada sem depender do staging. Reenvios indistinguíveis sem ID confiável não permitem exatamente uma vez automático. Coordenar revisão/consulta com ISSUE-083 e relay elegível com ISSUE-082.

---

author: codex
created: 2026-11-19T19:06:00Z

---

Entradas obrigatórias da ISSUE-133: cadastrar por INSERT em event_origins as origens usadas no pipeline e validá-las no escritor. Validar a proveniência legacy_declared de recordings.musicbrainz_id antes da regra 3 do resolvedor. Garantir por trigger removível que recibo vinculado tem occurrence_id=listens.track_id e recording_id=occurrences.recording_id, identity_status coerente com accepted/duplicate e resolution_version monotônica. No corte, aposentar ou compatibilizar o alocador de tracks e occurrences para evitar colisão; definir como guardar título e créditos apresentados sem linha em tracks (doc-039 §Contrato de entidades).

---

author: ebrgr
created: 2026-11-20T18:27:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-134` na branch `feat/issue-134-ingestao-por-ocorrencia-em-todas` criada a partir de `main`.

---

author: codex
created: 2026-11-20T18:30:00Z

---

Fase 1: plano entregue. Mapeia as fontes e a cobertura herdada da 081, propõe testes para os sete ACs e migração 010 para identidade, origens e guardas, com paridade v0.44 na via desligada. Explicita os cortes com 135, 082, 083 e 173, inclusive R-b na 173. Aguarda parecer do Advisor antes de implementar. ACs continuam abertos.

---

author: codex
created: 2026-11-20T19:19:00Z

---

Fase 1, plano v2. Parecer do Advisor: REPROVADO COM RESSALVAS. O plano incorpora as seis correções sobre proteção da gravação compartilhada, cascatas, 010 aditiva com reparo auditado de drift v9, exclusão real de recibos e chaves, replay na troca de modo e visibilidade dos leitores. Decisões de 20/11/2026: D1 álbum conhecido com faixa de ponte e aliases; D2 sem álbum pendente em perfis reais e ocorrência autônoma apenas no ensaio; D3 modo novo só em banco descartável até 135/083; D4 relay em espera, com habilitação bloqueada para perfil com relay até 082; D5 flag de mão única, opt-ins existentes em v9 e variante fixada no job e recibo; D6 MBID salvo/recebido, depois Subsonic (174), ListenBrainz (175) e campo vazio, com proveniência e resolução por metadados. Sequência E1–E4 registrada. E1 segue bloqueada para revisão do Advisor. ACs abertos.

---

author: codex
created: 2026-11-20T19:39:00Z

---

E1 concluída e parada para Advisor high. Migração 010 aditiva, guardas SQL, origens, flag de mão única, variantes e créditos. Drift v9 reparado com revisão migration. Merge legado sincroniza vínculos sobreviventes pelo trigger AFTER UPDATE antes do DELETE. Doc-040 §4 confirma a cascata das revisões no descarte do recibo (AC5 em E3). Replay desligado e restore com v0.44 passaram (18 escutas e 18 recibos em sombra). Ensaio de 300 mil: backup 136,1 ms, DDL/commit 896,2 ms, reparo de um vínculo 20,8 ms. make check passou com 13/13 jornadas. E2/E3 aguardam; ACs e status Em andamento preservados.

---

author: codex
created: 2026-11-20T19:58:00Z

---

Emenda da E1 após parecer Advisor high (REPROVADO COM RESSALVAS). Corrigida a própria 010 em sete commits: flag ativa com chave fixa e cascata, título e ponte, aborto de merge sem revisão sob OR IGNORE, asserções de reparo, índice de artista, comentários em inglês e IDs obrigatórios no primeiro vínculo. Testes focados e make check (13/13 jornadas) passaram. Conforme pedido, preservadas as evidências anteriores de replay v0.44 e carga de 300 mil sem repetir os ensaios. Pendências registradas no plano: BEGIN IMMEDIATE para MergeTracks e guarda de release para MergeAlbums (E2 ou 135), risco de INSERT OR REPLACE nas flags, e CHANGELOG em inglês antes do Reviewer da E3. Orientações da E2 registradas no plano. E1 parada para nova revisão, com ACs abertos.

---

author: codex
created: 2026-11-20T20:13:00Z

---

E2 concluída e parada para revisão do Advisor. Resolvedor occurrence-v1 com prova tipada fora do lock, decisões e catálogo atômicos, A/B e Intro, pendências conservadoras, recuperação por lease e COMMIT. 011 protege variante, origem, corte e relay no banco, com justificativa no plano e 010 intacta. Perfil convertido com mestre desligado retorna 503 e Retry-After, sem recibo nem escuta. Replay desligado v0.44 e restore passaram com 18 escutas e 18 recibos, e make check passou 13/13 jornadas. Fixture sobreposta: uma escuta, duas pendências, 0,91 ms até visibilidade no SQLite e 1,96 ms somados em três resoluções. Adaptadores na E3. BEGIN IMMEDIATE do merge legado e proteção de MergeAlbums ficam na 135. ACs abertos.

---

author: codex
created: 2026-11-20T20:50:00Z

---

Emenda da E2 após parecer REPROVADO COM RESSALVAS. A 011 e occurrence-v1 incorporam dedup de créditos, Duplicate com origem original, corte com sombra limpa (D9), redirecionamentos e fallback textual auditável (D7), descarte de afirmação externa conflitante (D8), evidência tipada, HTTP 409 e guarda de release vinculado. Admissão desligada usa leitura sob contenção e duração mantém o MBID legado. Testes dos pacotes afetados e make check 13/13 passaram. Replay v0.44 e restore repetidos com paridade. Pendências registradas no plano: posições legadas e fusão provisória na 135, fragmentação em MB indisponível e compilação na E4, consulta por alias e EXPLAIN de ocorrências no ensaio, cache/timeout antes dos 300 mil e ordinais do mesmo lote LB na E3/135. O texto de conflito do listen-add segue como pendência de apresentação. E3 segue aguardando liberação, com sequência E3 → Reviewer → E4.

---

author: codex
created: 2026-11-20T21:11:00Z

---

E3 concluída sobre v0.45, com commits por fatia e estado de retomada no PLANO. As seis origens foram verificadas, com modo e variante fixados no job, metadados e precisão preservados, replay de chave antiga com observação, exclusão e reimportação total/parcial, leases revogados e retomada sem staging. ACs 1–7 verificados por testes ou ensaio. O perfil separado conservou 300 mil recibos de sombra durante a resolução. Onda inicial de 101 escutas, 20 pendências na sobreposição, rollback seguido de recuperação após reinício e paridade v0.44 com controles desligados. Testes dos pacotes afetados, make check 13/13, sintaxe do ensaio e graphify AST concluídos. Aguarda Reviewer antes da E4. Status Em andamento.

---

author: codex
created: 2026-11-20T21:26:00Z

---

Correções do Reviewer da E3 concluídas. ListenBrainz HTTP e ZIP preservam zero segundos como instante ausente nos recibos. Exclusão v9 mantém a chave encerrada e a revisão profile_history_deleted; occurrence remove o recibo selecionado e permite novo evento no replay. Documentação agora usa a versão publicada v0.45, cujo binário recusou v11. Proveniência usa marcadores reservados, imports foram agrupados e o ensaio distingue retenção em outro perfil de contenção no mesmo perfil. O EXPLAIN conferiu o índice de alias. Testes dos pacotes afetados e make check 13/13 passaram. Pendências da E4/135/083 registradas no PLANO. Status Em andamento, aguarda releitura do Reviewer.

---
<!-- COMMENTS:END -->
