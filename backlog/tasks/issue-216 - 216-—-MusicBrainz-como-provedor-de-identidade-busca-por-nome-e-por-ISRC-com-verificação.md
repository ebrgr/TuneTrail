---
id: ISSUE-216
title: >-
  216 — MusicBrainz como provedor de identidade: busca por nome e por ISRC com
  verificação
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-27T14:54:00Z'
labels:
  - ingestao
  - catalogo
milestone: m-11
dependencies:
  - ISSUE-214
  - ISSUE-215
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
  - backlog/evidencias/issue-216/HANDOFF.md
documentation:
  - doc-048
priority: medium
type: feature
ordinal: 229000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. É a peça nova: hoje o MusicBrainz só é consultado por MBID que já veio na escuta (recording, release, release-group e apelidos do artista); não existe busca por nome. Cadeia A, etapa de identidade: buscar a gravação em /ws/2/recording?query= (sintaxe Lucene, com escape) e, quando o Navidrome fornecer o ISRC, consultar por ISRC. Todo candidato é verificado por duração, créditos de artista, título e álbum, com a regra e o limiar definidos no estudo da issue B; o score sozinho não basta. Ambíguo não é aceito e segue para a próxima fonte. O resultado é afirmação declarada, com fonte e data, nunca prova (contrato da ISSUE-134), e não sobrescreve MBID existente. Usa o cliente e a fila de 1 requisição por segundo já existentes, divididos com o resolvedor; a prioridade baixa e o teto por minuto para trabalho em lote não fazem parte desta issue: as importações de arquivo nem entram na fila de enriquecimento, que só recebe escutas novas pela API (source_type 'listenbrainz'), e o enriquecimento em lote do histórico é da ISSUE-213; mantém o User-Agent identificado, o espelho próprio por MUSICBRAINZ_URL e a chave TUNETRAIL_DISABLE_MUSICBRAINZ. Roda fora do caminho do ACK e fora do lock de escrita.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Escuta sem MBID recebe o MBID de uma gravação verificada por duração, créditos e álbum, como declaração com fonte e data
- [x] #2 Candidato ambíguo, divergente ou abaixo do limiar não é aceito e a cadeia segue; nenhum MBID existente é sobrescrito
- [ ] #3 Consulta por ISRC quando o Navidrome o fornece, tratando ISRC com mais de uma gravação como ambíguo
- [x] #4 Fila compartilhada com o resolvedor e funcionamento com o MusicBrainz desligado (a prioridade baixa e o teto por minuto para trabalho em lote ficam na ISSUE-213)
- [x] #5 Testes com respostas simuladas: versões ao vivo e de estúdio, título repetido, duração divergente, 503 e limite de taxa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-26T20:09:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-216` na branch `feat/issue-216-musicbrainz-como-provedor-de-identidade` criada a partir de `main`.

---

author: claude
created: 2026-11-26T20:09:00Z

---

Despacho (26/11/2026): implementação delegada ao Codex (modelo gpt-6.1-terra, effort medium) nesta worktree, com a regra de aceitação medida na ISSUE-215 (score >= 95, duração ±5 s, álbum e créditos, exatamente um candidato; ambíguo nunca é gravado). Ficam fora desta issue: cache negativo persistente (só proposta) e a ligação da consulta por ISRC ao fluxo, porque o Snapshot ainda não carrega ISRC (ISSUE-218 e 219). Depois do retorno: Advisor (Opus, effort high) sobre contrato, ambiguidade, uso do orçamento compartilhado e escape Lucene; depois o Reviewer; só então o aceite.

---

author: codex
created: 2026-11-26T20:15:00Z

---

Contradição com o código: o critério 4 e a descrição exigem prioridade baixa em importações e teto por minuto, mas Budget.Do (internal/enrichment/registry.go) não recebe prioridade nem classe de trabalho; MusicBrainzClient.Do usa RequestQueue.EnqueueContext, cuja fila FIFO e limitador global (queue/queue.go) não distinguem importações nem impõem teto separado. A arquitetura da ISSUE-214, pedida para reaproveitamento sem reescrita, não oferece essas garantias. Implementação suspensa para decisão, sem marcar o critério 4.

---

author: codex
created: 2026-11-26T20:15:00Z

---

Verificação parcial: testes direcionados do provedor e integração SQLite passaram; mutações de tolerância e unicidade falharam como esperado. make check e race foram interrompidos por suspensão do trabalho e não foram aprovados. A consulta ISRC está isolada, sem ligação ao Snapshot; cache negativo persistente não foi implementado: uma proposta futura deve definir TTL, invalidação e escopo por perfil/configuração, medindo custo e evitando reter resultados transitórios.

---

author: claude
created: 2026-11-26T20:20:00Z

---

Decisão do usuário (26/11/2026): a prioridade baixa e o teto por minuto para importações saem desta issue e vão para a ISSUE-213. Motivo verificado no código: as importações de arquivo (source_type 'file') não entram na fila de enriquecimento, que só aceita escutas novas pela API; e o orçamento compartilhado do MusicBrainz é uma fila FIFO com um limite global. O texto original da issue, escrito pelo coordenador, foi além da arquitetura da ISSUE-214. A 216 segue sem ampliar o orçamento.

---

author: codex
created: 2026-11-26T20:27:00Z

---

Verificação: make check passou (13/13 percursos); go test -race nos pacotes solicitados passou, inclusive internal/mbz sem filtro para executar o teste de fila compartilhada. Testes locais simulam versões, créditos, álbum, duração, ambiguidades, 429, 503, JSON e limites; a cadeia SQLite verifica a ordem e que o ListenBrainz não faz requisição quando os campos foram resolvidos. Mutações: remover duração quebrou studio_versus_live, duration_outside_tolerance e unknown_duration; aceitar o primeiro candidato quebrou ambiguous_within_tolerance. Consulta ISRC testada isoladamente, ainda sem ligação ao Snapshot: critério 3 permanece aberto. Sem teste com MusicBrainz real. Cache negativo persistente segue como proposta registrada no comentário 4.

---

author: advisor
created: 2026-11-27T14:37:00Z

---

Parecer do Advisor (Opus, effort medium, Modo A, 27/11/2026) sobre 74b60ee: APROVADO, sem falha crítica. Quatro melhorias: (1) quatro testes de ramos explícitos que hoje não pegam mutação (mesmo MBID em caixas diferentes, truncamento só por 100 resultados, Snapshot.Recording divergente, Failed não entra no cache); (2) normalizedMetadata remove pontuação e ficou mais frouxo que a regra medida na 215; (3) título só de pontuação vira busca só por artista e álbum; (4) opcional, ambiguidade por truncamento só se o último resultado tem score >= 95. A consulta com release na frase procede e invalida a comparação direta com os números da 215. A fila cheia virando Failed pertence à ISSUE-213. Texto completo em backlog/evidencias/issue-216/PARECER-ADVISOR.md.

---

author: claude
created: 2026-11-27T14:49:00Z

---

Parecer do Reviewer (27/11/2026) sobre o diff final 3d6c9f3: nenhum item obrigatório. Recomendado corrigir antes do aceite: (1) Instance() devolve a URL crua do MusicBrainz e NewMusicBrainz não valida, ao contrário do Subsonic, então uma URL de espelho com credencial seria gravada em metadata_values.instance e nos backups; (2) créditos de artista comparados só com EqualFold, enquanto o álbum usa a normalização medida na 215. Riscos menores: escuta de vários artistas sem artist_names não casa; o teste de integração não confere duração, estado do job nem instância; consulta com release sem medição; lookupISRC sem chamador. Detalhes em backlog/evidencias/issue-216/PARECER-REVIEWER.md. Revisão feita por quem aplicou as melhorias do Advisor, portanto não independente.

---

author: ebrgr
created: 2026-11-27T14:51:00Z

---

Aceite do usuário (27/11/2026): aceita a ISSUE-216 como está, sem corrigir os riscos do Reviewer. Ficam registrados e NÃO corrigidos: (1) Instance() devolve a URL crua do MusicBrainz e NewMusicBrainz não valida a URL, então um espelho com credencial embutida em KOITO_MUSICBRAINZ_URL seria gravado em metadata_values.instance e nos backups; (2) créditos de artista comparados com EqualFold, sem a normalização usada no álbum; (3) escuta de vários artistas sem artist_names não é resolvida; (4) o teste de integração não confere duração, estado do job nem instância; (5) a consulta inclui release e a ISSUE-215 mediu sem ela; (6) lookupISRC sem chamador. O critério 3 (ISRC) fica aberto até as ISSUE-218 e 219. Prioridade e teto por minuto em lote estão na ISSUE-213.

---

author: ebrgr
created: 2026-11-27T14:54:00Z

---

Publicação: versão 0.79 (tag `v0.79`), branch `feat/issue-216-musicbrainz-como-provedor-de-identidade` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
