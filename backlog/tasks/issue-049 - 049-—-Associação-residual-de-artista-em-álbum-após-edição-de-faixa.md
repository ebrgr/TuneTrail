---
id: ISSUE-049
title: 049 — Associação residual de artista em álbum após edição de faixa
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-14T02:46:00Z'
labels:
  - catalogo
  - banco
milestone: m-9
dependencies: []
references:
  - backlog/evidencias/issue-049/INVESTIGACAO.md
  - backlog/evidencias/issue-049/observacao_test.go.txt
  - backlog/evidencias/issue-049/ENTREGA.md
  - backlog/evidencias/issue-049/VERIFICACAO.txt
priority: high
type: bug
ordinal: 49000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/049-associacao-residual-artista-album.md` — numeração local 049. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Aberta · **Identificada:** 09/11/2026

## Problema reproduzido

A página do artista Engenheiros do Hawaii (ID 1) lista o álbum **1**, dos
Beatles (ID 23), apesar da remoção da associação do artista na faixa **Love Me
Do** (ID 89). A faixa já não aparece entre as faixas do artista; o álbum ainda
aparece na seção de álbuns associados. Verificado pelas páginas públicas, sem
alterar o banco do usuário.

Ao associar um artista à faixa, `UpdateTrack` também cria uma relação
`artist_releases` com o álbum. Ao remover o artista da faixa, a relação da faixa
é excluída, mas a do álbum é preservada. As consultas de álbuns por artista usam
essa segunda relação. Alias de faixa e relação entre artistas são dados distintos.

## Risco e direção

As relações existentes de álbum não registram procedência: uma associação
secundária pode ser crédito independente e legítimo, ou ter sido criada por
uma faixa. Não é seguro apagar automaticamente todas as relações sem faixa
correspondente; álbuns podem ter créditos próprios e catálogos incompletos.
Uma trigger também apaga o álbum se a última associação for removida.

Estudar separação entre créditos explícitos e relações derivadas das faixas,
inclusive o caso em que ambos existem simultaneamente. Preservar créditos
legados cuja origem não pode ser determinada; propor revisão ou remoção manual
específica e transparente. Qualquer mudança de esquema deve ser planejada e
validada com migração, backup e compatibilidade do SQLite v3, sem alterar
migrations consolidadas retroativamente.

## Critérios de aceite

- [ ] Adicionar e remover artista de faixa atualiza relações derivadas, sem
  deixar um álbum indevido na página do artista.
- [ ] Preservar créditos explícitos, primários, legados ambíguos e relações
  ainda justificadas por outra faixa; contemplar origem dupla e álbuns parciais.
- [ ] Tratar dados antigos com revisão explícita, sem limpeza global automática
  e sem exclusão acidental de álbum, faixas, escutas ou dados de outros perfis.
- [ ] Cobrir persistência, transação, rollback, páginas e rankings por artista,
  testes de regressão e integridade SQLite; registrar procedimento de recuperação.
- [ ] Atualizar changelog e documentação após a correção.

Advisor/Sol apontou que não se pode inferir a procedência com os campos atuais.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Adicionar e remover artista de faixa atualiza a participação derivada sem migração do esquema v3.
- [x] #2 Créditos próprios, primários, legados e participação por outra faixa são preservados, sem contagem duplicada.
- [x] #3 Revisão pontual do legado documentada e testada com guardas contra cascata, sem limpeza automática.
- [x] #4 Testes de persistência, rollback, páginas, rankings, perfis e integridade aprovados em make check.
- [x] #5 Documentação e changelog atualizados.
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:39:00Z

---

Plano TuneTrail 1.0 (doc-032): incluída na Fase 0 · Base; corrigir antes de novos recursos de catálogo (ISSUE-096 depende desta).

---

author: Codex
created: 2026-11-14T02:04:00Z

---

Investigação Codex em 13/11/2026: defeito confirmado no código atual e presente na tag local v0.8. Reprodução com SQLite em memória: remover artista de faixa deixa vínculo residual de álbum e mantém álbum no ranking em dois perfis; crédito prévio de álbum e vínculo derivado produzem snapshots indistinguíveis; apagar último vínculo aciona trigger e exclui álbum/faixa/escutas dos dois perfis. Três testes de observação e três existentes passaram (não comprovam correção). Evidências e comando reproduzível em backlog/evidencias/issue-049/INVESTIGACAO.md. Advisor @Sol revisou e registrou ressalvas: justificar origens coexistentes, preservar legados, evitar cascata e coordenar migração com ISSUE-080; contrato ainda precisa dessas definições. Bootstrap recusa esquema acima de v3. Sem alteração funcional ou acesso ao banco do usuário; tarefa permanece Aberta, sem critérios de correção marcados.

---

author: Codex
created: 2026-11-14T02:07:00Z

---

Início autorizado pelo usuário. Branch fix/issue-049-associacao-artista-album criada manualmente, pois make issue.start recusou alterações preexistentes da ISSUE-048; preservadas sem inclusão nesta entrega.

---

author: Codex
created: 2026-11-14T02:18:00Z

---

Verificação: solução sem migração conforme escolha explícita do usuário; esquema permanece v3. Participação de álbum derivada por consulta das faixas, sem persistir crédito de álbum na edição ou ingestão. Créditos próprios e legados preservados; promoção explícita e merges cobertos. make check final aprovado (Go/vet/JS/build e 10/10 e2e), incluindo novos testes de páginas/rankings nos dois perfis, origem dupla, legado, rollback, integridade e guardas da revisão manual. Advisor @Sol aprovou; grafo atualizado. Evidência: backlog/evidencias/issue-049/ENTREGA.md. Limitação: vínculos antigos ambíguos precisam da revisão pontual documentada; nenhum dado real foi alterado. Aguarda aceite do usuário; sem publicação ou push.

---

author: claude
created: 2026-11-14T02:46:00Z

---

Verificação independente (Claude, 13/11/2026) antes do aceite: (1) diff revisado — vínculo artista↔álbum passa a ser derivado de artist_tracks (albumArtistRelations); edição de faixa não toca artist_releases, logo não aciona o trigger trg_delete_orphan_releases; função removida cleanOrphanedEntries não tinha chamadas; UpdateTrack mantém a verificação de faixa inexistente. (2) make check aprovado (10/10 e2e). (3) Mutação: reintroduzir a gravação do crédito em UpdateTrack reprova 5 testes. (4) Desempenho com 200 mil faixas e 400 mil escutas: busca de álbum por escuta 0,01→0,02 ms, ranking do artista 0,02→0,10 ms, artistas do álbum 0,03 ms; filtros usam índice nos dois ramos da UNION. (5) Caixa-preta com binários: no main, o álbum continua na página do artista após remover o artista da faixa; na correção, sai; o álbum continua existindo nos dois. (6) Banco gravado pela 0.8 com vínculo residual: a correção sozinha mantém o vínculo visível (limitação documentada); o SQL de editing.md, executado como escrito no sqlite3 3.43, remove exatamente 1 vínculo, preserva álbum e escutas, e a página deixa de mostrá-lo.

---

author: claude
created: 2026-11-14T02:46:00Z

---

Aceite: autorizado pelo usuário em 13/11/2026, condicionado à verificação acima. Atenção: o caso real registrado nesta issue (Engenheiros do Hawaii → álbum 1) permanece no banco do usuário até o reparo manual de docs/src/content/docs/guides/editing.md, com backup e aplicação parada.

---
<!-- COMMENTS:END -->
