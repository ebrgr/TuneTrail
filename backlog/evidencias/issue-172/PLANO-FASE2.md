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
