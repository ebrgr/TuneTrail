---
id: ISSUE-218
title: '218 — Navidrome como provedor de metadados: gêneros, ano, datas, ISRC e BPM'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-24T19:53:00Z'
updated_date: '2026-11-28T02:03:00Z'
labels:
  - catalogo
  - ingestao
milestone: m-13
dependencies:
  - ISSUE-214
references:
  - backlog/evidencias/issue-055/ESTUDO-COMPLEMENTAR.md
  - backlog/evidencias/issue-218/MEDICAO.md
  - backlog/evidencias/issue-218/HANDOFF.md
  - backlog/evidencias/issue-218/PARECER-REVIEWER.md
  - backlog/evidencias/issue-218/INSTRUCOES-CORRECOES.md
  - backlog/evidencias/issue-218/PLANOS-GATILHOS.txt
  - backlog/evidencias/issue-218/MUTACOES.txt
  - backlog/evidencias/issue-218/INSTRUCOES-CORRECOES-2.md
  - backlog/evidencias/issue-218/PARECER-ADVISOR.md
  - backlog/evidencias/issue-218/INSTRUCOES-CORRECOES-3.md
documentation:
  - doc-048
priority: medium
type: feature
ordinal: 231000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 24/11/2026 (subsistema de enriquecimento compartilhado, começando pelas ISSUE-174 e 175) e o fluxo que ele descreveu: banco já salvo, Navidrome, MusicBrainz, depois ListenBrainz e os serviços comerciais. Cadeia B, etapa Navidrome. Amostra aleatória do servidor do usuário em 24/11/2026 (presença de campo, não qualidade nem censo): em 100 músicas, 78 tinham genres, 98 year, 41 musicBrainzId, 26 isrc e 14 bpm; em 15 álbuns, releaseDate e originalReleaseDate em 15, genres em 11, recordLabels em 11, musicBrainzId em 5 e releaseTypes em 5. O adaptador de escuta do Navidrome 0.64.2 não envia gêneros, tags, ano, datas, ISRC nem BPM, então só a consulta à API Subsonic traz esses dados. Usar o ID nativo do servidor apenas quando comprovadamente associado a ele; caso contrário, buscar candidatos por search3 e exigir correspondência inequívoca (já feito na ISSUE-174). Depois de uma associação segura, getSong e getAlbum fornecem os campos adicionais, guardando a origem por entidade em metadata_values. Confirmar a presença de originalReleaseDate na resposta de álbum, que não consta no tipo Child. O ISRC alimenta a ponte por ISRC da ISSUE-216. Não registrar credenciais nem URLs autenticadas em evidências, logs ou testes.

**Correção de 27/11/2026 (conferida no código da v0.79).** (a) `metadata_values` (migração 024) só aceita os tipos `mbid`, `genre`, `tag`, `recording_year`, `release_date` e `artist_origin_country`: ISRC e BPM exigem tipos novos e uma migração, que no SQLite recria a tabela. O texto anterior e o doc-048 diziam que a tabela não mudaria, e isso estava errado. Decisão do usuário em 27/11/2026: uma única migração cobrindo os tipos desta issue e da ISSUE-217, entregue antes da ISSUE-195. (b) Hoje só `mbid` é gravado (`writeMBIDDeclaration` em `internal/db/sqlite/enrichment.go` ignora os demais campos): a primeira entre a 217 e a 218 constrói a escrita dos outros tipos, com o vínculo à entidade certa; a leitura e a tela são da ISSUE-212. (c) A cadeia de identidade roda antes da de metadados. Um ISRC colhido na cadeia de metadados só chega à busca do MusicBrainz nas escutas seguintes, a menos que a etapa do Navidrome na cadeia de identidade também o colha. (d) A consulta por ISRC da ISSUE-216 (`lookupISRC`) existe, testada, e sem chamador: o critério 3 daquela issue ficou aberto à espera do ISRC.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Gêneros, ano e datas do Navidrome gravados com fonte e instância, só depois de associação segura da faixa e do álbum
- [x] #2 ISRC e BPM guardados quando existirem, na tabela de metadados por entidade criada por migração (decision-012), e o ISRC disponível para a ponte com o MusicBrainz
- [x] #3 Campos ausentes ou vazios não geram pendência nem apagam dado anterior
- [x] #4 Testes com respostas simuladas, sem credenciais, cobrindo versões repetidas e resultados truncados
- [x] #5 Escrita dos metadados descritivos na tabela por entidade, com a fonte no cadastro de fontes e vínculo à entidade certa (se a ISSUE-217 não a tiver construído antes), e migração coordenada com a ISSUE-195
- [x] #6 A consulta por ISRC da ISSUE-216 ligada à cadeia de identidade, usando o ISRC colhido do Navidrome, com ISRC que leva a mais de uma gravação tratado como ambíguo
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-27T15:00:00Z

---

27/11/2026: texto corrigido. ISRC e BPM não são aceitos por metadata_values (exigem migração), a referência «issue C» era um resíduo e passa a ser a ISSUE-216, e a ordem das cadeias atrasa o uso do ISRC. Critério 2 ajustado; acrescentados o de escrita e migração e o de ligar a consulta por ISRC, pendência herdada do critério 3 da ISSUE-216.

---

author: ebrgr
created: 2026-11-27T15:21:00Z

---

Decisão do usuário em 27/11/2026 (A5, A7): migração única de metadados antes da 195, com a ISSUE-217; a ligação da consulta por ISRC fica nesta issue.

---

author: ebrgr
created: 2026-11-27T21:26:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-218` na branch `feat/issue-218-navidrome-como-provedor-de-metadados` criada a partir de `main`.

---

author: codex
created: 2026-11-27T21:26:00Z

---

Plano: medir primeiro o custo por escuta; em seguida, se dentro do limite, preparar a migração compartilhada, associar metadados e ISRC com segurança, validar preservação e executar os testes exigidos.

---

author: codex
created: 2026-11-27T21:27:00Z

---

Dúvida para decisão: a medição sintética com os índices atuais e oito valores por escuta resultou em 3.806,82 bytes/escuta, acima do limite aproximado de 500 B. Parada antes da implementação conforme instrução. Aceitar esse custo ou estudar outra modelagem? Qualquer mudança no vínculo por escuta exige decisão expressa. Método e limites em backlog/evidencias/issue-218/MEDICAO.md.

---

author: claude
created: 2026-11-27T21:34:00Z

---

Decisão do usuário em 27/11/2026: modelo D (metadados descritivos por entidade, com cadastro de fontes; MBIDs continuam por escuta), registrado na decision-012. Critérios 2 e 5 ajustados. Retoma a implementação.

---

author: codex
created: 2026-11-27T21:43:00Z

---

Nova parada: modelo D isolado com cadastro de fontes e timestamps completos mede 637,75 B/escuta no cenário de uma escuta por faixa; incluindo duas execuções found por escuta que a implementação parcial pode registrar, o cenário sintético chega a 941,26 B/escuta. O limite de aproximadamente 650 B foi excedido. É aceitável estudar mudança no registro/recuperação das execuções para reduzir o custo, ou aceita-se o custo maior? Método e limitações em MEDICAO.md; código parcial sem commit.

---

author: claude
created: 2026-11-27T21:49:00Z

---

Decisão do usuário em 27/11/2026: «opção 1» — apagar os registros de sucesso de enrichment_provider_runs quando o trabalho terminar. A limitação de exportação foi aberta como ISSUE-234. Retoma a implementação.

---

author: codex
created: 2026-11-27T21:59:00Z

---

Implementação verificada: migração 026 e modelo D; associação conservadora e metadados Navidrome; ISRC na identidade MusicBrainz; preservação e ciclo de vida; recuperação protegida pelo lease. Medição final de 671,33/220,36/65,54 B por escuta para 1/3/10 reproduções por faixa. make check aprovado, incluindo 13 jornadas; testes selecionados com race aprovados e testes de mutação detectaram as alterações. Exportação e restauração descritiva permanecem na ISSUE-234. Detalhes no HANDOFF.

---

author: claude
created: 2026-11-27T22:02:00Z

---

Parecer do Reviewer sobre 3e1ab90: REPROVADO, cinco itens obrigatórios (quatro na busca de identidade e um de desempenho nos gatilhos da 026). Detalhes e reproduções em PARECER-REVIEWER.md. Aguarda decisão do usuário sobre as correções.

---

author: claude
created: 2026-11-28T01:02:00Z

---

Reviewer reprovou (5 itens, PARECER-REVIEWER.md). A pedido do usuário, a tarefa volta ao Codex para avaliar e corrigir, incluindo a regra nova de consultar o Navidrome só quando faltar ID ou metadado. Instruções: backlog/evidencias/issue-218/INSTRUCOES-CORRECOES.md.

---

author: codex
created: 2026-11-28T01:22:00Z

---

Correções do parecer: identidade preservada nas falhas de detalhes; fallback e agregação de ISRCs com créditos de artista; gatilhos por entidade e consultas Navidrome condicionadas. Reproduções e mutações dos seis itens; make check aprovado (13 jornadas); detector de corrida em execução. Medição e riscos em backlog/evidencias/issue-218/MEDICAO.md e HANDOFF.md; exportação permanece na ISSUE-234.

---

author: codex
created: 2026-11-28T01:23:00Z

---

Verificação final das correções: detector de corrida aprovado nos testes de enriquecimento/migração e no teste de instância ampliado; make check aprovado, 13 jornadas. As seis mutações compilaram e foram detectadas; cenários de gatilhos, custo e riscos documentados no HANDOFF e MEDICAO.

---

author: claude
created: 2026-11-28T01:33:00Z

---

Segunda revisão (commit 7aef45c): os cinco itens anteriores foram resolvidos; REPROVADO por dois itens novos (consultas repetidas quando há divergência de dados; cache do Navidrome ignora o Prior). make check aprovado. Detalhes em PARECER-REVIEWER.md.

---

author: claude
created: 2026-11-28T01:37:00Z

---

A pedido do usuário, volta ao Codex para os dois itens da segunda revisão, rodando somente os testes dessas duas alterações. Instruções: backlog/evidencias/issue-218/INSTRUCOES-CORRECOES-2.md.

---

author: codex
created: 2026-11-28T01:40:00Z

---

Segunda rodada: divergências definitivas não ficam repetíveis e limpam marcador anterior; chave do cache distingue gravação e artistas pendentes após Prior. Reproduções e testes direcionados passaram; guarda de prévia e índice permanecem fora do escopo desta rodada.

---

author: claude
created: 2026-11-28T01:41:00Z

---

Terceira revisão (commit 6916f01): APROVADO para os dois itens. Rodados somente os testes dessas alterações, a pedido do usuário; make check pendente sobre este commit. Dois ajustes pequenos seguem abertos (PARECER-REVIEWER.md).

---

author: advisor
created: 2026-11-28T01:48:00Z

---

Advisor (Opus 5.5) sobre a branch: um bloqueio (a consulta por ISRC declara a gravação mesmo com conflito de artista conhecido, reproduzido) e três pontos para decisão do usuário. Detalhes em PARECER-ADVISOR.md.

---

author: claude
created: 2026-11-28T01:53:00Z

---

Decisão do usuário em 27/11/2026: o MBID sempre tem preferência sobre o ISRC. Volta ao Codex para o bloqueio do Advisor sob essa regra e para a guarda de prévia da versão 26. Instruções: backlog/evidencias/issue-218/INSTRUCOES-CORRECOES-3.md.

---

author: codex
created: 2026-11-28T01:56:00Z

---

Terceira rodada: MBIDs conhecidos prevalecem sobre ISRC, com descarte integral e busca por nome em caso de conflito ou ausência de artista; a guarda da versão 26 exige as duas tabelas. Reproduções e testes direcionados passaram; detalhes no HANDOFF.

---

author: claude
created: 2026-11-28T01:57:00Z

---

Quarta revisão (commit c00b09e): APROVADO. O MBID tem preferência sobre o ISRC; guarda do esquema 26 completa. Pontos restantes do Advisor abertos nas ISSUE-235, 236 e 237.

---

author: ebrgr
created: 2026-11-28T02:00:00Z

---

Reviewer aprovado na quarta rodada (c00b09e); Advisor atendido, com os demais pontos nas ISSUE-235, 236 e 237. make check executado neste envio. Para o aceite: no critério 1, quando só o álbum diverge, os valores da faixa são gravados (decisão pendente na ISSUE-236). A suíte completa de sqlite com detector de corridas não foi rodada nesta branch.

---

author: ebrgr
created: 2026-11-28T02:00:00Z

---

Aceite do usuário em 27/11/2026: «aceito e publique».

---

author: ebrgr
created: 2026-11-28T02:03:00Z

---

Publicação: versão 0.82 (tag `v0.82`), branch `feat/issue-218-navidrome-como-provedor-de-metadados` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
