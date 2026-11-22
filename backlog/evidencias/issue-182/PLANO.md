# Plano ISSUE-182 — teto vitalício e reprocessamento em massa

Origem: parecer do Advisor da ISSUE-173 (backlog/evidencias/issue-173/PARECER-ADVISOR.md, melhorias 1, 2, 4 e 5). Pré-requisito do corte da ISSUE-172 (decisão do usuário de 21/11/2026).

## Objetivo
1. Recibo cuja resolução sempre excede a lease de 120 s não repete para sempre: ao atingir um teto vitalício de claims, é estacionado (`parked_at`, `park_reason`) com motivo próprio.
2. A central de saúde reprocessa todos os recibos estacionados do perfil de uma vez.
3. O reprocessamento registra quem e quando.
4. O comentário de `warnParkedIngest` descreve a janela real de 1 minuto.

## Estado atual (conferir)
- `claimNextIngest` (`internal/db/sqlite/ingest_resolve.go`): lease expirada incrementa `retry_attempt_base`, então a tentativa devolvida não cresce; `attempt_count` é a contagem vitalícia.
- `retryIngest`/`ReprocessIngest`/`warnParkedIngest` em `internal/db/sqlite/ingest_retry.go`; `maxIngestAttempts=5`; esperas 1/2/4/8 min; estacionar = `parked_at` + `park_reason`, `state` continua `retryable`.
- Central: seção "Falhas permanentes" com reprocessamento unitário (`/admin/health/reprocess`), 25 listados.
- A 181 trouxe um padrão de ação em massa presa a digest (`ingest_review_batch.go`) e o ID da conta em `evidence.decided_by`; reaproveite o estilo, sem copiar o que não se aplica.

## Decisões fixadas
- **Teto vitalício:** estacionar no claim (ou na recuperação da lease) quando `attempt_count` desde o último reprocessamento passar de 5x o orçamento (25 claims), com `park_reason='lease_exhausted'` e alerta WARN limitado, sem perder snapshot, chave ou contagem. O contador recomeça no reprocessamento (`retry_attempt_base`/coluna aditiva, o que for mais simples e correto).
- **Reprocessar todos:** botão por perfil, dono do perfil e root, POST + CSRF, idempotente; reprocessa só os estacionados do perfil, em lotes limitados dentro de uma transação por lote ou uma única transação se barato; repetição não duplica trabalho nem altera `work_version` além do necessário; perfil diferente = no-op.
- **Auditoria:** gravar quem (ID da conta) e quando em estrutura aditiva (coluna em `ingest_events` ou tabela `ingest_reprocess_audit`); o reprocessamento unitário também registra. Sem dados pessoais além do ID.
- **Transitório x permanente:** NÃO mudar a classificação de falhas nesta issue (fora do escopo); registrar em comentário se a análise sugerir uma issue futura.
- Migração aditiva (schema atual v17 → v18) com backup/guarda de downgrade como as anteriores.

## Restrições
- Visual da central: reaproveitar a seção "Falhas permanentes" (1/2/3 colunas só com `min-width` 40/64/90rem, `health.css` ≤ 400 linhas, hover padrão, i18n en/pt-BR); regenerar a referência ISSUE-075 se a UI mudar.
- Não alterar o backoff 1/2/4/8 min, a semântica de `retryable` nem a fila de revisão da 181.
- Testes: teto vitalício (receipt cuja resolução sempre excede a lease), reprocessar todos, repetição, concorrência com o resolvedor, isolamento entre perfis, auditoria; `TUNETRAIL_WEB_DIR=$PWD/assets` nos testes de engine; medir o custo do reprocessamento em massa com fila grande (ex.: 10 mil estacionados).
- Advisor: recomendado depois da implementação (lease/concorrência); acionado pelo usuário.
