---
id: ISSUE-172
title: >-
  172 — Remover o escritor legado e o interruptor de ativação da ingestão após o
  piloto
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-19T19:58:00Z'
updated_date: '2026-11-23T00:26:00Z'
labels:
  - ingestao
milestone: m-11
dependencies:
  - ISSUE-081
  - ISSUE-135
references:
  - backlog/evidencias/issue-172/ENSAIO.md
  - backlog/evidencias/issue-172/INVENTARIO-FASE2.md
  - backlog/evidencias/issue-172/FASE2.md
  - backlog/evidencias/issue-172/LIMPEZA.md
priority: low
type: chore
ordinal: 185000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: decisão do usuário em 2026-11-19 sobre a ISSUE-081 (parte 2), item (a) do parecer do Advisor. A 081b entrega o pipeline atrás de um interruptor (mestre mais flag por perfil), desligado por padrão, mantendo o escritor legado como fonte da verdade até a medição de algumas semanas de modo sombra em produção e o piloto. Depois de um piloto bem-sucedido e do corte coordenado com a ISSUE-135 (leitores por ocorrência), esta tarefa remove o escritor legado de listens, o interruptor e as flags por perfil, e a tripla legada idx_listens_legacy_identity se a 135 já tiver substituído os leitores. Escopo a refinar com o Advisor (effort high, concorrência e migração de dados): migração que remove o que a 081b deixou como compatibilidade, reversão só por backup, ensaio sintético e, se houver, validação em cópia de snapshot autorizado. Pré-requisitos: ISSUE-081 e ISSUE-135 concluídas e piloto avaliado.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 O escritor legado de listens e o interruptor de ativação são removidos sem perda de escutas, com ensaio sintético e o Advisor (effort high) registrado
- [ ] #2 Contagens por perfil e digests iguais antes e depois, reversão por backup comprovada
- [ ] #3 Os limiares aprovados (p95 de ACK < 200 ms, visibilidade < 10 s, < 0,1% de 503, até 15 escutas/s) são medidos em ensaio sintético com carga, no lugar das 3 semanas de sombra, e registrados
- [ ] #4 Backup verificado e restauração comprovada antes do corte; ISSUE-181 e ISSUE-182 concluídas antes de ligar a ingestão ativa
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-21T23:22:00Z

---

Decisão do usuário (21/11/2026): mudança de plano. Depois de concluir a ISSUE-181 (e a 182), ligar o código novo e remover o legado sem esperar as 3 semanas de sombra. Consequências registradas: (1) os limiares aprovados passam a ser comprovados por ensaio sintético com carga e por cópia de snapshot autorizado, não por produção; (2) o corte só é reversível por backup, então o backup verificado e a restauração testada são pré-requisito; (3) Advisor (effort high, concorrência e migração de dados) obrigatório antes do aceite; (4) a janela de manutenção de 10 minutos continua. Risco aceito pelo usuário: sem o período de sombra, problemas de dados reais só aparecem depois do corte.

---

author: ebrgr
created: 2026-11-22T16:17:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-172` na branch `chore/issue-172-remover-o-escritor-legado-e` criada a partir de `main`.

---

author: Codex
created: 2026-11-22T16:31:00Z

---

Fase 1 executada em cópia isolada. make check e E2E com mestre ligado passaram. A ocorrência conservou 205 ACKs sem álbum como pendências que o legado materializou. Lote completo a 15/s com 900 escutas materializadas. E2E com ocorrência por padrão: 7/13. Prévia local em 4120. Métricas e limitações em backlog/evidencias/issue-172/ENSAIO.md.

---

author: ebrgr
created: 2026-11-22T17:02:00Z

---

Decisões do usuário após o ensaio da fase 1 (22/11/2026): (1) escutas sem álbum são aceitas como Lançamento desconhecido (nova ISSUE-186, pré-requisito do corte); (2) o padrão da flag Revisar é LIGADO: a ativação da ingestão por perfil e a migração do corte passam a definir review=1 por padrão; (3) o timeout de 20 s da prova do MusicBrainz continua reagendando o recibo. Pendências da fase 2: ISSUE-186; reescrever as fixtures que gravam pelo escritor legado (291 falhas com ocorrência por padrão no ensaio) e as expectativas síncronas e de relay; remover escritor legado, interruptor, flags e índice legado; Advisor obrigatório antes do aceite.

---

author: claude
created: 2026-11-22T19:34:00Z

---

Premissa registrada (22/11/2026): produção migra de v2 para a última versão em um salto; migrations intermediárias só importam ao desenvolvimento. Na fase 2: (a) não é preciso tratar recibos intermediários em produção; (b) o teste de aceite do corte deve ser o salto v2 -> última (TestBootstrapKoitoV2WithoutUpstreamIndexes cobre 200 escutas até currentSchemaVersion; ampliar para conferir contagens, ingestão ativa e Revisar) em vez de migrações incrementais; (c) avaliar se faz sentido consolidar as migrations 013+ depois do corte.

---

author: codex
created: 2026-11-22T20:11:00Z

---

Fase 2, corte 0: inventário concluído, make check verde (Go com timeout de 30 min, E2E 13/13). Cortes seguintes pausados por decisão de contrato: como a escuta manual deve tratar o mesmo perfil, ocorrência e segundo quando já existe escuta aceita de outra fonte. Evidência em backlog/evidencias/issue-172/INVENTARIO-FASE2.md.

---

author: claude
created: 2026-11-22T20:12:00Z

---

Decisão do usuário (22/11/2026) sobre a escuta manual: se já existe escuta aceita de outra fonte no mesmo perfil, ocorrência e segundo, informar que já existe e manter a contagem (sem duplicata e sem pendência). Registrado no PLANO-FASE2 para os cortes seguintes.

---

author: codex
created: 2026-11-22T20:18:00Z

---

Fase 2, corte 1 parcial: helper real de ocorrência e espera de resolução nas fixtures convertidas; make check verde com mestre ligado, 13/13 E2E. Comparação isolada de arquivo Koito v1 confirmou 1 escuta nos dois modos, mas aliases artista/álbum/faixa 2/2/2 no legado e 1/1/1 por ocorrência. Pausa na decisão de contrato sobre preservar aliases secundários explícitos de Koito v1.

---

author: claude
created: 2026-11-22T20:19:00Z

---

Decisão (claude, default conservador; usuário pode reverter): o importador Koito v1 por ocorrência PRESERVA os aliases secundários explícitos do arquivo (artista, álbum, faixa), para que exportar e reimportar não perca dados que o legado preservava. Alias já pertencente a outra entidade do mesmo perfil/catálogo é ignorado e contado no relatório da importação, sem abrir pendência e sem fundir entidades.

---

author: codex
created: 2026-11-22T20:26:00Z

---

Fase 2, corte 1: Koito v1 preserva aliases secundários explícitos por ocorrência, ignora colisões e registra contagem durável, exibida no resumo ao repetir o mesmo arquivo. Teste cobre artista, álbum e faixa. make check verde com mestre ligado, 13/13 E2E. Comparação isolada de 450 requisições por modo mostrou 409 ACKs integralmente materializados, paridade de projeção, p95 de ACK ativo 3,64 ms e 0% 503.

---

author: codex
created: 2026-11-22T20:28:00Z

---

Corte 2 em andamento: ação manual já grava pela ocorrência escolhida. Repetição do mesmo perfil, ocorrência e segundo retorna sucesso sem novo evento ou Inbox. Origem manual e nenhum relay, preservando o contrato local. Make check verde, inclusive percurso público com ocorrência ativa.

---

author: codex
created: 2026-11-22T20:32:00Z

---

Corte 2 parcial, ponto de parada: escuta manual por ocorrência e separação do now-playing passaram no make check (13/13 E2E). Comparação isolada da rota experimental obrigatória reproduziu regressão ISSUE-170: 1 escuta no perfil em ambos os binários, 1→0 para o artista do MBID no ranking e 1→2 entidades no catálogo. A rota experimental foi retirada da worktree; os cortes 3 a 5 não iniciaram.

---

author: claude
created: 2026-11-22T20:32:00Z

---

Decisão (claude, default conservador = paridade com o legado; usuário pode reverter): o resolvedor por ocorrência deve honrar MBIDs de artista recebidos (artist_mbids e mbid_mapping.artist_mbids, ISSUE-170): se EXATAMENTE um artista do catálogo do perfil já tem esse MBID, o crédito aponta para ele (a grafia observada vira alias/nome observado, sem criar um segundo artista); MBID com 2+ artistas no catálogo = pending_identity (ambiguidade, como as demais); sem artista com o MBID, segue o comportamento atual por nome. O resultado deve igualar o do escritor legado em TestMappedArtistMBIDResolution170 antes de retirar o fallback. Toca a identidade: Advisor effort high obrigatório ao final.

---

author: codex
created: 2026-11-22T20:36:00Z

---

Corte 2: aplicada a regra da ISSUE-170 à ocorrência. MBID único vincula artista existente, duplicidade de caixa normalizada gera pending_identity, ausência usa nome. Créditos sem correspondência posicional inequívoca seguem por nome; additional_info prevalece sobre mbid_mapping. Teste de paridade e casos de borda passaram, make check verde.

---

author: codex
created: 2026-11-22T20:50:00Z

---

Corte 2 parcial: rota HTTP histórica ficou verde em make check (13/13 E2E), com paridade de atribuição ISSUE-170 e todos os 409 ACKs do ensaio a 15/s materializados em ambos os modos. Pausa por perda confirmada no candidato do importador: Koito v1 válido com 220 aliases, 18.345 bytes, registra 1 escuta no legado e 0 no novo caminho (HTTP 400, sem recibo) por limite de 10 KiB no snapshot. Troca do importador retirada; cortes 3 a 5 pendentes.

---

author: claude
created: 2026-11-22T20:51:00Z

---

Decisão (claude, default conservador; usuário pode reverter): nunca perder a escuta por volume de aliases. No importador Koito v1 por ocorrência, os aliases secundários explícitos entram no snapshot só até o orçamento do limite de 10 KiB por item (em ordem de arquivo, mantendo os campos obrigatórios); os excedentes são ignorados e contados no relatório da importação como 'alias ignorado por limite' (mesma contagem dos conflitos), sem rejeitar a escuta e sem aumentar o limite público. Substitui, só no excesso, a preservação integral de aliases decidida antes.

---

author: codex
created: 2026-11-22T20:56:00Z

---

Corte 2: orçamento Koito v1 aplicado e make check verde, E2E 13/13. Arquivo sintético de 18.345 bytes com 220 aliases resultou em 1 escuta aceita, 143 aliases preservados e 77 ignorados por limite. Nova perda confirmada na conversão experimental: arquivo válido de 10.807 bytes sem aliases secundários e client de 10.500 bytes gerou 1 escuta no legado e HTTP 400, nenhuma escuta ou recibo, no candidato. Conversão experimental retirada. Cortes seguintes continuam pendentes.

---

author: claude
created: 2026-11-22T20:57:00Z

---

Decisão (claude, default conservador; usuário pode reverter): escutas de arquivo não se perdem por campo opcional gigante. Nos importadores de arquivo por ocorrência, campos de texto OPCIONAIS (client, versão do cliente, release/álbum, aliases, etc.) são truncados em limite UTF-8 seguro (client 255 bytes; demais opcionais 1 KiB; aliases pelo orçamento já decidido) ANTES de medir o limite de 10 KiB por item, e a truncagem é contada no relatório da importação ('campo truncado'). Título e artista (obrigatórios) acima de 4 KiB continuam sendo recusados com erro claro (item abusivo), sem mudar o limite público de 10 KiB. Caso de teste: Koito v1 válido com client de 10.500 bytes gera 1 escuta com client truncado.

---

author: codex
created: 2026-11-22T21:02:00Z

---

Corte 2: normalização UTF-8 de opcionais nos cinco adaptadores, contagem e erro explícito para artista/título >4 KiB. make check verde, 13/13 E2E. Cliente de 10.500 B passou com 1 escuta, 255 B e 1 campo truncado. Parada por perda comprovada com campos obrigatórios de 3.121 e 3.141 B que ultrapassam 10 KiB ao serializar em JSON: legado 204 e 1 escuta, candidato 400 sem escuta/recibo. Troca experimental retirada. Cortes 3 a 5 pendentes.

---

author: claude
created: 2026-11-22T21:03:00Z

---

Decisão (claude, default conservador; usuário pode reverter): o limite público de 10 KiB por item é parte do contrato novo e NÃO será perseguido caso a caso. Item de arquivo cujo título+artista (obrigatórios, após escape JSON) ainda passa de 10 KiB depois de truncar os opcionais é RECUSADO INDIVIDUALMENTE: o item é contado no relatório da importação como 'item recusado (acima de 10 KiB)' e o restante do arquivo segue; não se recusa o arquivo inteiro. Registrado como divergência aceita e documentada em relação ao legado (que gravava títulos de milhares de bytes). Vale para os cinco importadores. Sem mais ensaios de tamanho: faça uma única varredura em tabela e siga.

---

author: codex
created: 2026-11-22T21:07:00Z

---

Corte 2 em andamento. A recusa individual acima de 10 KiB passou em teste de tabela nos cinco formatos, com duas escutas normais materializadas por arquivo. Após drenar o resolvedor, as fixtures de Last.fm, ListenBrainz e Koito v1 ainda falham nas consultas de catálogo por MBID. O legado promovia MBIDs declarados no arquivo ao catálogo sem validação online. O resolvedor por ocorrência preserva evidência sem promover MBID de lançamento sem confirmação e a regra da ISSUE-170 cobre artista já identificado no catálogo. A decisão pendente é se a importação deve promover esses IDs declarados offline, e sob que proveniência, ou manter apenas a evidência no recibo. Cortes 2 a 5 pendentes.

---

author: claude
created: 2026-11-22T21:07:00Z

---

Decisão (claude, default conservador = paridade com o legado; usuário pode reverter): importações de arquivo por ocorrência VINCULAM os MBIDs declarados no arquivo (lançamento e artistas novos), mesmo offline, com proveniência 'declarada' (campos já existentes mbid_source e mbid_declared_at do snapshot: origem = formato da importação, data = momento da importação), nunca como 'verificada'. Nunca sobrescrever um MBID diferente já existente na entidade: conflito é ignorado e contado no relatório (sem pendência e sem fusão), e MBID já pertencente a OUTRA entidade segue a regra da ISSUE-170 (um dono = aponta para ele; 2+ = pending_identity). Mesma regra de MBID declarado que o caminho de API confiável já usa; ISSUE-187 item 4 trata validar depois pelo MusicBrainz.

---

author: codex
created: 2026-11-22T21:16:00Z

---

Corte 2, importadores convertidos: cinco adaptadores por ocorrência, recusa individual de item acima de 10 KiB e preservação offline dos MBIDs declarados. Last.fm, ListenBrainz e Koito v1 passam após resolução. Conflitos registrados no recibo e reportados no reenvio. make check verde, 13/13 E2E. Restam a retirada dos caminhos de compatibilidade vinculados ao schema 20 e a migração 021.

---

author: codex
created: 2026-11-22T21:20:00Z

---

Corte 3 em andamento. Testes focados do salto v2→v21 (200 escutas com digest preservado, nova escuta visível) e v20→v21 (recusa transacional com sombra não vinculada, retry preservando digest e Down bloqueado) passaram. A aprovação de relay pela interface depende de salvar antes o destino sem permissão; ao tornar todos os perfis ativos, a validação do caminho por ocorrência devolve 409 nesse passo e impede o fluxo de aprovação. O E2E de administração e o de migração reproduzem o 409. Precisa decidir entre preservar rascunho sem entrega até aprovação e mover a aprovação de rede para antes do salvamento. Corte 3 sem commit; make check completo pendente, cortes 4 e 5 pendentes.

---

author: claude
created: 2026-11-22T21:21:00Z

---

Decisão (claude, default conservador; usuário pode reverter): relay em perfil por ocorrência permite SALVAR O DESTINO COMO RASCUNHO ('aguardando aprovação de rede'), sem entrega e sem criar intenções no outbox, mantendo o fluxo atual salvar -> aprovar na interface. Nenhuma saída de rede acontece antes da aprovação; escutas anteriores à aprovação não são reenviadas retroativamente (documentar na tela e nos docs). Após a aprovação o destino ativa e as novas escutas geram intenções normalmente. O salvamento deixa de devolver 409 por falta de aprovação; as demais validações (esquema, host, faixa de IP, allowlist) continuam exigidas na aprovação e na ativação.

---

author: codex
created: 2026-11-22T21:33:00Z

---

Corte 3 concluído. Schema 021 retira flags e índice legados e liga Revisar por padrão. Testes v2→v21 e v20→v21 preservaram contagens e digests e confirmaram bloqueio do Down. Relay salvo fica em rascunho, sem rede nem outbox. A geração registrada na admissão impede retroativo, inclusive para recibos resolvidos só depois da aprovação. IP proibido continua bloqueado; host DNS só é resolvido durante a confirmação, com allowlist existente para faixas necessárias. make check verde, E2E 13/13.

---

author: codex
created: 2026-11-22T21:36:00Z

---

Corte 4 concluído. Removidos o interruptor de ingestão e as rotas de ativação, com ocorrência em todos os perfis. Guia atualizado com Revisar ligado, rascunho e aprovação do relay sem retroativo, janela de 10 minutos, .backup, quick_check e restauração. make check verde, E2E 13/13.

---

author: codex
created: 2026-11-22T22:01:00Z

---

Corte 5 concluído. make check e suíte completa com -race -timeout 30m verdes, E2E 13/13. A 15 escutas/s, p95 de ACK 11,43 ms no lote misto e 11,60 ms no lote completo, visibilidade máxima 1,07 s, 0% 503 e nenhuma escuta aceita ausente. Paridade legado/ocorrência para 2.045 escutas e mesmo digest. Relay isolado entregou 2.947 escutas e zerou o outbox. Prova v20→v21 e restauração para v20 preservaram 2.826 escutas, contagens e digests. Evidência em FASE2.md. Status Em andamento.

---

author: advisor
created: 2026-11-22T22:22:00Z

---

Reviewer: APROVADO com débito de manutenção (código morto schema<21), sem BUG; make check 13/13 e -race sqlite/importer/ingest/catalog/relay ok. Advisor (effort high): APROVADO COM MELHORIAS: sem falhas críticas; migração, relay em rascunho, identidade e medições ok; follow-up para remover a compatibilidade com schema<21; acompanhar Revisar ligado por padrão. Decisões conservadoras do Claude listadas em PARECER-ADVISOR.md.

---

author: claude
created: 2026-11-22T22:36:00Z

---

Pedido do usuário (22/11/2026): antes de fechar, limpar o código legado que não está mais em uso (compatibilidade com schema < 21, resolvedor v9, ativação, SaveListen antigo, catalog.SubmitListen) e reescrever os testes de atualização com SQL direto; evidência em LIMPEZA.md. Rebase sobre a main (conflitos em CHANGELOG, Makefile, ISSUE-075/reference e arquivo da tarefa) fica para depois da limpeza, por squash em poucos commits.

---

author: codex
created: 2026-11-22T22:38:00Z

---

Limpeza, etapa de ativação: removidos Preview/SetIngestActivation e ConvertOccurrence, que perderam as rotas. Os testes de confirmação dessas ações foram retirados por testarem exclusivamente as operações removidas. Mantidos os testes das migrações e os controles atuais de sessão de root e de relay. make check verde, 13/13 E2E.

---

author: codex
created: 2026-11-22T22:42:00Z

---

Limpeza, etapa catálogo e escuta manual: removidos catalog.SubmitListen, SaveListen e SaveListenWithResult. As asserções de associação foram portadas para a entrada PublishNowPlaying, com gravação manual apenas quando a asserção precisa de histórico. Os testes de migração e dos gatilhos antigos semeiam escutas por SQL. Os cinco testes de escuta ao vivo e o roteiro HTTP na porta descartável 4127 passaram.

---

author: codex
created: 2026-11-22T22:49:00Z

---

Limpeza, etapa de admissão e resolução: a admissão de produção chama apenas o escritor de ocorrência. O worker deixou de despachar para v9, e o CLI de replay de sombra foi removido. Fixtures de atividade e isolamento executam no schema 21. Os testes de atualização de retry/reprocessamento semeiam recibos históricos por INSERT. A carga curta deu ACK p95 de 10,30 ms, 109/109 escutas aceitas materializadas e digest igual ao escritor anterior. Ainda há auxiliares históricos nos testes e consultas de compatibilidade a eliminar.

---

author: codex
created: 2026-11-22T23:05:00Z

---

Limpeza de saúde, Inbox e fixtures de sombra em ce6de58. O make check passou com 13/13 E2E e a suíte Go completa passou com os caminhos dos assets e idiomas. Os cinco testes da escuta ao vivo passaram. Na porta temporária 4127, playing_now retornou 200 e apareceu na API e na home sem criar histórico. A escuta single retornou 200 e criou a única entrada. A instância foi encerrada. Na carga isolada, ACK p95 de 4,43 ms, 109 aceitas e materializadas, zero 503 e digest igual ao legado. Os ramos de revisão e outbox para schema antigo e algumas fixtures v9 ainda precisam ser retirados.

---

author: codex
created: 2026-11-22T23:11:00Z

---

bc838ea porta a admissão usada pelas fixtures de ocorrência para o schema 21, preserva as guardas históricas por SQL e verifica a recuperação de lease com o resolvedor atual. make check 13/13, suíte Go completa, go vet e roteiro da porta 4127 passaram. A carga final teve ACK p95 de 7,52 ms no lote misto e 45,80 ms no piloto de 900, zero 503 e todas as escutas aceitas materializadas. A limpeza fica em andamento enquanto restarem a fixture de finalização v9 e os ramos de revisão e relay para schemas antigos.

---

author: claude
created: 2026-11-22T23:39:00Z

---

Reviewer (limpeza do código legado): APROVADO, sem BUG; limpeza parcial por escolha explícita (resto em LIMPEZA.md). make check 13/13 e -race ok em sqlite, importer, ingest, catalog, relay, admin e webui; escuta ao vivo preservada; carga com ACK p95 7,5 ms e 45,8 ms. Débito: ramos schema<21 em revisão/saúde/relay e fixtures históricas do resolvedor v9.

---

author: ebrgr
created: 2026-11-23T00:23:00Z

---

Verificação: Reviewer e Advisor (effort high) APROVADOS; limpeza do legado aprovada pelo Reviewer; branch comprimida e rebaseada sobre a main com a 125 e a 126; -race adminui/webui/admin/engine ok; make check 13/13 E2E

---

author: evertonberger
created: 2026-11-23T00:23:00Z

---

Aceite do usuário em 22/11/2026 ('aceito a 172 e já faça uma issue de acompanhamento'). Follow-up: ISSUE-194.

---

author: ebrgr
created: 2026-11-23T00:26:00Z

---

Publicação: versão 0.65 (tag `v0.65`), branch `chore/issue-172-remover-o-escritor-legado-e` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
