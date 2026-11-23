# Plano ISSUE-172 — fase 2: remover o escritor legado e o interruptor

## Decisões e premissas (usuário, 21–22/11/2026)
- Sem as 3 semanas de sombra; corte direto depois do ensaio (fase 1, `ENSAIO.md` na branch).
- **Produção sai do schema v2 direto para a última versão**; versões intermediárias só existem em desenvolvimento. Não projetar tratamento de estados intermediários para produção; o teste de aceite do corte é o salto v2 -> última.
- Escuta sem álbum = ocorrência sem lançamento (ISSUE-186, v0.61). Recibos `release_missing` antigos: ver ISSUE-187 (não são pré-requisito).
- **Revisar (duplicatas) fica LIGADO por padrão** (decisão "2: ligado"). **Manter o reagendamento do timeout do MusicBrainz** (decisão "3: reagendar").
- Advisor (effort high) obrigatório antes do aceite; Reviewer após a entrega. Quem conversa com o usuário roda os dois; o Codex não cria PARECER-*.md.

## Escopo
Remover: escritor legado de `listens` (origem `legacy`), o modo sombra, o interruptor mestre `TUNETRAIL_INGEST_ACTIVE` (`cfg.ActiveIngestEnabled`), a tabela `profile_ingest_flags` com seus gatilhos (migrações 009–012, 014, 016), o índice legado `idx_listens_legacy_identity` se nenhum leitor depender dele, a UI/rota de ativação (`engine/handlers/ingest_activation.go`) e a validação condicionada ao interruptor em `relay_management.go`. A ingestão por ocorrência passa a ser o único caminho: handler `lbz_submit_listen`, importadores (`importer/shadow.go`, `listenbrainz.go`, `koito.go`), escuta manual (`internal/admin/listens.go`) e restauração de arquivo.
Manter: o que a flag `review` guardava (Revisar por perfil) em estrutura própria, padrão LIGADO para todo perfil (existente e novo); linhas históricas `origin='legacy'` continuam legíveis.

## Cortes (cada um termina com `make check` verde e commit `ISSUE-172: ...`)
0. **Base e inventário.** Mesclar `main` na branch. Escrever `backlog/evidencias/issue-172/INVENTARIO-FASE2.md`: todo chamador de `SaveListen`, `catalog.SubmitListen`, `BeginImportIngest/AdmitIngest(Policy)` com `master`, modo sombra, gatilhos, índice, `origin='legacy'`, e a decisão para cada um (remover, migrar para ocorrência, manter). Se uma decisão de produto/contrato não estiver neste plano, PARE e relate; não invente.
1. **Infra de testes.** Helper de fixture que cria escutas pelo escritor de ocorrência e espera a resolução (admissão ≠ resolução); converter os grupos de falhas do ensaio (212 fixtures que gravavam pelo legado, 18 de flag duplicada, 18 de importação síncrona, 14 de guardião de conversão, 11 de catálogo assíncrono, relay 409 por destino inválido, 6 jornadas E2E, etc.) para o helper, mantendo a suíte verde com o interruptor ligado por padrão no `Makefile` ANTES de remover o código. Sem enfraquecer asserções; as expectativas síncronas passam a aguardar a resolução.
2. **Remoção do código legado** (itens do Escopo), com as rotas de escuta manual e importação nativa migradas para o escritor de ocorrência segundo o contrato doc-039/decisão 008. Comentários em inglês.
3. **Migração 021** (goose, Down protegido como as anteriores): remove gatilhos, `profile_ingest_flags` e o índice legado; cria a estrutura de Revisar com padrão 1 para todos os perfis; trata recibos de sombra/abertos de bancos de desenvolvimento de forma explícita e documentada (recusar com mensagem acionável ou descartar com auditoria; justificar). Testes: (a) salto v2 -> última (ampliar `TestBootstrapKoitoV2WithoutUpstreamIndexes`: 200 escutas preservadas, digests e contagens por perfil iguais, ingestão ativa e Revisar ligados, escuta nova visível); (b) cópia de banco v20 de desenvolvimento -> última; (c) reversão só por backup.
4. **Docs e configuração.** Remover a variável e o interruptor de `docs/.../active-ingestion.md`, exemplos, Docker, Makefile e scripts; CHANGELOG [Unreleased] em inglês; runbook curto do corte (backup `.backup`, janela de 10 min, restauração e `quick_check`).
5. **Medição e prova.** Rodar `scripts/rehearse_issue172.py` na cópia (sem rede externa): p95 de ACK < 200 ms, visibilidade < 10 s, < 0,1% de 503, 15 escutas/s no perfil piloto (re-medir: há escritas extras de gatilhos das ISSUE-181/184/185/186). Suíte Go completa com `-race -timeout 30m` (o prazo padrão de 10 min é curto) e os 13 E2E. Backup verificado e restauração comprovada (contagens e digests). Registrar em `backlog/evidencias/issue-172/FASE2.md` sem dados pessoais.

## Restrições
- Não tocar o banco `.local/koito/tunetrail.db` nem a instância da 4110; usar cópias em `~/Documents/TuneTrail-ensaio-172/`. Nenhuma rede externa nos ensaios.
- Sem push, merge, publish ou Done. Commits com prefixo `ISSUE-172: `. Sem delegar a outro agente.
- Parar e relatar se: houver perda de escuta em qualquer comparação; o ACK p95 passar de 200 ms; uma decisão de contrato faltar.

## Decisão aberta (padrão adotado se o usuário não disser outra)
Reenvio com ID externo confiável quando Revisar está ligado: manter o comportamento atual (abre pendência, caso f da ISSUE-181).

## Decisão de contrato: escuta manual (22/11/2026)
Escuta manual que coincide com escuta aceita de outra fonte no mesmo perfil, ocorrência e segundo: informar que já existe e manter a contagem, sem duplicata e sem pendência (resposta e interface como hoje: inseriu ou já existia).

## Decisão de contrato: aliases secundários na importação Koito v1 (22/11/2026)
Preservar os aliases secundários explícitos do arquivo (artista, álbum, faixa) no caminho por ocorrência. Alias já pertencente a outra entidade é ignorado e contado no relatório da importação, sem pendência e sem fusão. Default conservador; o usuário pode reverter.


Como a resolução do Koito v1 é assíncrona, a contagem de conflitos fica no
resultado durável por item. Reenviar o mesmo arquivo apenas lê o resultado
apurado e apresenta a contagem no resumo, sem nova escuta. Alias novo mantém
proveniência do perfil importador e não entra como alias canônico global.


A escuta manual seleciona uma ocorrência já conhecida e tem resultado
síncrono. Registra origem manual, sem mensagem de relay nem recibo de
admissão, preservando as características da ação administrativa anterior.

## Decisão de contrato: MBID de artista recebido na ocorrência (22/11/2026)
Paridade com o legado (ISSUE-170): se exatamente um artista do catálogo do perfil tem o MBID recebido (artist_mbids ou mbid_mapping.artist_mbids), o crédito da ocorrência aponta para ele e a grafia observada fica como nome observado/alias, sem criar segundo artista. 2+ artistas com o MBID = pending_identity. Sem artista com o MBID, comportamento atual por nome. Default conservador; o usuário pode reverter.


Os campos posicionais são usados quando há um MBID válido por crédito
observado. Sem correspondência posicional inequívoca, resolver por nome,
sem adivinhar vínculo. IDs válidos de `additional_info` têm prioridade
sobre `mbid_mapping`, conforme ISSUE-170.

A fusão de álbuns conserva a proteção da ISSUE-134 para lançamentos de
ocorrências vinculadas. Recusa a mudança de lançamento com 409 e conserva
as escutas nos dois álbuns. A fusão de catálogo legado continua coberta
separadamente. A ordem de execução do worker não atribui significado a
IDs sequenciais de catálogo.

## Decisão de contrato: limite de aliases na importação Koito v1 (22/11/2026)
A escuta nunca é perdida por volume de aliases: os aliases secundários entram no snapshot até o orçamento de 10 KiB por item (ordem do arquivo; campos obrigatórios têm prioridade); o excedente é ignorado e contado no relatório como alias ignorado por limite, sem rejeitar a escuta e sem aumentar o limite público. Caso de teste: 1 escuta com 220 aliases (18.345 bytes) deve gerar 1 escuta, aliases até o orçamento e o restante contado. Default conservador; o usuário pode reverter.

## Decisão de contrato: campos opcionais gigantes em importações de arquivo (22/11/2026)
Campos opcionais (client, versão do cliente, álbum/release, aliases etc.) são truncados em limite UTF-8 seguro (client 255 bytes; demais opcionais 1 KiB; aliases pelo orçamento já decidido) antes da medição de 10 KiB por item; a truncagem é contada no relatório como campo truncado. Título e artista obrigatórios acima de 4 KiB seguem recusados com erro claro. O limite público de 10 KiB não muda. Vale para TODOS os importadores de arquivo por ocorrência, não só o Koito v1. Default conservador; o usuário pode reverter.

Nos cinco adaptadores de arquivo, truncar texto opcional em UTF-8 seguro
antes da medição (cliente 255 B, demais 1 KiB), contabilizar no resumo e
usar o orçamento restante para aliases. Recusar artista ou título acima
de 4 KiB com erro explícito. Se ainda for preciso liberar espaço antes dos
aliases, ceder listas opcionais e depois textos de cliente e álbum, com
contagem. Manter o limite serializado de 10 KiB.

## Decisão de contrato: item de arquivo acima de 10 KiB (22/11/2026)
O limite de 10 KiB por item é contrato novo e não é perseguido caso a caso. Item cujo título+artista (após escape JSON) passa de 10 KiB mesmo com opcionais truncados é recusado individualmente, contado no relatório como item recusado (acima de 10 KiB), e o restante do arquivo segue (não se recusa o arquivo inteiro). Divergência aceita em relação ao legado. Vale para os cinco importadores.

## Decisão de contrato: MBIDs declarados em importações offline (22/11/2026)
Importações de arquivo vinculam MBIDs declarados (lançamento e artistas novos) mesmo offline, com proveniência declarada (mbid_source = formato da importação, mbid_declared_at = momento), nunca verificada. Não sobrescrever MBID diferente já existente (conflito ignorado e contado no relatório). MBID que já pertence a outra entidade segue a regra da ISSUE-170. Paridade com o legado; validação posterior fica na ISSUE-187 item 4. Default conservador; o usuário pode reverter.

O resumo do reenvio consulta os conflitos apurados no recibo original, como nos
aliases secundários. O relógio local da admissão fica na evidência e não muda
a identidade do item de arquivo. No ListenBrainz, um release MBID explícito
sem título de lançamento usa o título da faixa como rótulo, preservando a
exportação legada. Sem título e sem release MBID, a ocorrência segue sem
lançamento. As referências de capa do Koito v1 permanecem restritas à rede
pública e só são instaladas na criação da entidade.

## Decisão de contrato: relay como rascunho até a aprovação de rede (22/11/2026)
Salvar o destino do relay em perfil por ocorrência cria um rascunho (aguardando aprovação de rede): sem entrega e sem intenções no outbox; nenhuma saída de rede antes da aprovação; escutas anteriores não são reenviadas retroativamente (informar na tela e nos docs). Aprovar ativa o destino; as validações de esquema, host, faixa de IP e allowlist continuam na aprovação e na ativação. Fluxo da interface preservado (salvar e depois aprovar). Default conservador; o usuário pode reverter.
