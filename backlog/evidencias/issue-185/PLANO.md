# Plano ISSUE-185 — descartar recibos que não deram certo

Pedido do usuário em 22/11/2026 (veja a tarefa). Depende da 182 (v0.58) e usa a central reorganizada pela 184 (v0.59, schema v19). Esta issue usa a migração v20, se precisar de uma.

## Objetivo
Ação **Descartar** na central de saúde, individual e em massa, para recibos que não viraram escuta: estacionados (173/182), pendentes de revisão (181) e inválidos. Com rastro mínimo e sem apagar escutas já aceitas.

## Estado do código (conferir)
- `DiscardIngestPending` (`internal/db/sqlite/ingest_delete.go`) remove um recibo `received/retryable/pending_identity/pending_duplicate`, modo active, resolvedor por ocorrência, sem evento e sem `closed_at`, **sem rastro**, e não tem rota nem botão.
- Não há descarte para recibos estacionados (`parked_at`) nem inválidos; não há descarte em massa.
- A exclusão de escuta aceita é pela página de escutas (`DeleteListen`: remove escuta, recibos e itens abertos do relay).
- A 181 criou a fila de revisão e o padrão de ação em massa presa a digest (`ingest_review_batch.go`); a 182 criou reprocessar todos e o histórico em `ingest_revisions` (autor user, `evidence.decided_by`); a 184 criou o Inbox (resumo diário por gatilhos em `ingest_events`: o descarte deve manter `ingest_inbox_daily` e `ingest_source_summary` consistentes pelos gatilhos existentes).

## Decisões fixadas (defaults recomendados; o usuário pediu para iniciar)
- **Rastro mínimo:** o descarte não é remoção silenciosa. Gravar um registro do descarte (quem = ID da conta, quando, motivo/estado de origem, identificador do recibo e `reason_codes`), sem o corpo bruto nem o snapshot. Preferir estrutura aditiva (tabela `ingest_discards` ou linha em `ingest_revisions` mantida após a exclusão do recibo; decidir com base nas FKs de `ingest_revisions` que têm CASCADE) e documentar a escolha em comentário de código. Retenção alinhada à retenção atual de recibos.
- **Escopo:** só recibos **sem escuta** (`event_id IS NULL`), estados estacionado, pendente (identity/duplicate/retryable/received sem lease ativa) e inválido. **Recusar** recibos ligados a uma escuta, com orientação para a exclusão de escuta.
- **Em trânsito/relay:** um recibo que nunca virou escuta não tem intenção de relay; se existir qualquer intenção aberta ligada ao recibo, recusar. O que já foi enviado ao relay (ListenBrainz) não é apagado no destino, e isso fica explícito na tela (aviso junto da confirmação) e no guia.
- **Segurança e concorrência:** dono do perfil e root; POST + CSRF; confirmação na tela (passo explícito, não só um clique); transação única sob `withIngestWriter` com CAS por `resolution_version`/`work_version` e estado esperado; recibo com lease ativa (em resolução) não pode ser descartado; ação em massa presa a digest como a 181, em lotes limitados; idempotente; perfil diferente = no-op/404; sem expor segredos.
- **Coerência das projeções:** os gatilhos da 181/184 (`ingest_source_summary`, `ingest_inbox_daily`) devem permanecer corretos após o DELETE; testar.

## Restrições
- Reaproveitar a seção "Falhas permanentes" e "Pendências de revisão"; UI só com `min-width` 40/64/90rem, `health.css` ≤ 400 linhas (dividir + manifesto se passar), hover do padrão, i18n en/pt-BR; regenerar a referência ISSUE-075 (`make ui.reference`) e conferir no Orca (390/1100/1500 px).
- Não alterar a política de retry, o relay nem a fila de revisão; não implementar "desfazer descarte".
- Migração só se necessária, aditiva, com guarda de downgrade como as anteriores.
- Testes por classe: cada estado elegível, estado inelegível (aceito, com escuta, com lease ativa), isolamento entre perfis, repetição, concorrência com o resolvedor e com a decisão de revisão (181) e com o reprocessar (182), consistência das projeções (resumo de fontes e Inbox), rastro gravado sem corpo bruto, em massa em lotes. `TUNETRAIL_WEB_DIR=$PWD/assets` nos testes de engine. Medir o custo do descarte em massa (10 mil).
- Docs (guia de ingestão) e CHANGELOG [Unreleased] em inglês.
- Advisor: recomendado depois da implementação (exclusão de dados); acionado pelo usuário.
