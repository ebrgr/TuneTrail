# Plano ISSUE-181 — fila de revisão de duplicatas

Origem: ISSUE-083 (central de saúde somente leitura), doc-043 (081b), doc-039 (contrato: revisão "Igual" liga ao evento existente; "Diferente" cria novo evento). Risco 1 do parecer da 083 (consultas da central) herdado aqui.

## Objetivo
Seção "Pendências de revisão" em `/admin/health`: o dono do perfil (e o root) decide cada recibo `pending_duplicate` com **Aceitar como duplicata** (liga ao evento existente, sem nova escuta) ou **Manter como escuta separada** (cria o evento, com intenção na outbox), com **Ver evidência**. Flag "Revisar" por perfil, desligada por padrão.

## Estado do código (conferir; AC #1 é verificação)
- `finalizeActiveReceipt` (`internal/db/sqlite/ingest_resolve.go`) e `writeOccurrenceDecision` (`ingest_occurrence.go`) já gravam `state='pending_duplicate'`, `duplicate_decision='review'`, `evidence` com `candidate_event_id`/`candidates`, quando `profile_ingest_flags.review=1`.
- `insertRelayIntent` cria a intenção da outbox só em `accepted`. Não há rota, ação nem tela que resolva pendências.
- Verificar se a flag `review` pode ser ativada hoje (UI/rota/CLI) e se os dois resolvedores (legado e por ocorrência) produzem os casos.

## Decisões fixadas
- **Aceitar como duplicata:** `state='duplicate'`, `duplicate_decision='same'`, `event_id` = candidato escolhido (deve pertencer ao mesmo perfil), `closed_at`, revisão com `author='user'`; nenhuma nova escuta, nenhuma intenção na outbox.
- **Manter separada:** insere a escuta (mesma regra do caminho `accepted`: `distinct_occurrence_event` no resolvedor por ocorrência; `event_id` novo), `state='accepted'`, `duplicate_decision='different'`, revisão `author='user'`, e `insertRelayIntent` na **mesma transação** (sem dual write).
- Idempotência e concorrência: CAS por `resolution_version` (e `work_version`), em writer exclusivo (`withIngestWriter`); decisão repetida ou concorrente não duplica escuta nem intenção; segunda chamada é no-op/erro claro.
- Autorização: dono do perfil e root, mesmo gate da central; POST + CSRF; isolamento por perfil (recibo de outro perfil = no-op/404). Não expor tokens nem corpo bruto; a evidência exibe só campos seguros.
- Flag de Revisar por perfil: desligada por padrão; ligar/desligar pela central (dono/root) ou ponto já existente, o que for coerente com o código.
- Fora do escopo: ISSUE-097 (indicadores), ISSUE-112 (logs), reprocessar em massa (ISSUE-182).

## Restrições
- Reaproveite a seção/colunas da central (ISSUE-173): 1/2/3 colunas com só `min-width` 40/64/90rem, `health.css` (≤ 400 linhas), hover igual ao padrão, i18n en/pt-BR, regenerar a referência ISSUE-075.
- Não alterar a semântica dos estados já publicados nem a política de retry da 173.
- Medir o custo das consultas da central (`healthSources`, `healthDecisions`, pendências) com volume realista (ex.: 300 mil eventos) via EXPLAIN QUERY PLAN; se necessário limitar a janela ou indexar (migração aditiva; schema atual v15). Registrar em `backlog/evidencias/issue-181/`.
- Testes por classe: isolamento entre perfis, decisão repetida, concorrência (duas decisões opostas), candidato de outro perfil, flag desligada, outbox exatamente uma vez.
- Advisor: recomendado depois da implementação (decisão que cria escuta + outbox); acionado pelo usuário.
