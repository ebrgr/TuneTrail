---
id: doc-043
title: Estudo ISSUE-081 — pipeline de ingestão com deduplicação explicável (Advisor)
type: other
created_date: '2026-11-19T16:15:00Z'
updated_date: '2026-11-19T16:15:00Z'
tags:
  - estudo
---
# Estudo ISSUE-081 — pipeline de ingestão com deduplicação explicável

Advisor (Opus, effort medium), 2026-11-19, só leitura, sobre a main v0.39. Fontes: ISSUE-081 e relacionadas, doc-040 (contrato de ingestão), `004_ingest_infrastructure.sql`, RELATORIO da 080 e o código citado.

## A. Como funciona hoje

```
Cliente LBZ ──POST /apis/listenbrainz/1/submit-listens──► LbzSubmitListenHandler (engine/handlers/lbz_submit_listen.go)
  valida corpo ≤16 MiB, perfil do token, artista/título
  └─► por item: singleflight ─► catalog.SubmitListen (internal/catalog/catalog.go)
         trava global (submitListenGate) → resolve artista/álbum/faixa (MBID→alias→nome; pode chamar MusicBrainz)
         playing_now: só em memória (memkv) e para aí
         senão: SaveListen → INSERT … ON CONFLICT (profile_id,track_id,listened_at) DO NOTHING
  erro em qualquer item → HTTP 500 (itens anteriores do lote já gravados)
  tudo OK → go doLbzRelay(bytes do pedido), sem nova tentativa → HTTP 200
Upload de arquivo (internal/importer/*) ─► mesmo catalog.SubmitListen ─► contadores Imported/Duplicates
```

- Duplicata hoje: chave primária `(profile_id, track_id, listened_at)` em segundos (003_listening_profiles.sql:58; listen.go:38-39); sem janela de tempo; decidida em silêncio pelo SQLite (`DO NOTHING`); no upload só soma `Duplicates++` (upload.go:40-49).
- Falhas silenciosas: (1) repetição legítima no mesmo segundo perdida; (2) mesma escuta contada duas vezes se a grafia/álbum muda o `track_id`; (3) horário ausente vira `time.Now()` (lbz_submit_listen.go:241-244), proibido pelo contrato; (4) lote via API falha pela metade com 500; (5) relay em goroutine sem nova tentativa, repassa o pedido bruto (inclusive playing_now) e não ocorre se algum item falhar; (6) importador aborta o arquivo no primeiro erro (listenbrainz.go:214-217) e o log conta linhas; (7) bug lateral: lbz_submit_listen.go:203 descarta o resultado de `ParseUUIDSlice(mbid_mapping.ArtistMBIDs)`.
- v4 (ISSUE-080): criou `integrations`, `ingest_events`, `ingest_event_details`, `ingest_observations`, `ingest_revisions`, `outbox`, `outbox_details` e índices únicos de idempotência; hoje nada escreve nelas (só ingest_retention.go e testes). Guardas: estados só received/pending_*/invalid, `event_id IS NULL`, `resolution_version=0`, triggers bloqueando revisões e outbox até a 133.

## B. O que a 081 muda

```
Fonte ─► validar ─► [TX] gravar recibos do lote ─► 200 = "guardado" (503 se o disco falhar)
recibo ─► resolver identidade ─► dedup: Igual | Diferente | Revisar (+ reason_code, evidência, rule_version)
  Diferente ─► cria evento/escuta + intenção na outbox (082)
  Igual     ─► liga o recibo ao evento existente, sem contar de novo
  Revisar   ─► pendente; não conta até alguém revisar (tela na 083)
playing_now ─► só em memória, sem recibo
```

Exemplo: "Song X" às 21:03:15 (celular) e às 21:03:16 (multi-scrobbler, álbum grafado diferente). Depois da 081 o segundo recibo fica pendente com `cross_source_candidate` (Δ=1 s, álbum divergente, regra v1); a janela de ±2 s só seleciona candidatos, não prova identidade. Com ID externo confiável e mesmo conteúdo, o resultado é "Igual" automático.

Impactos: dados (grava em ingest_events/observations; a 133 remove as guardas numa migração v7+), código (novos internal/ingest e internal/dedup; alterados o handler LBZ, catalog.go, importadores e listen.go), API (mesmo JSON; 200 = guardado; 503; sem 500 em conflito), UI (a 081 só grava os motivos; tela na 083), desempenho (~1-2 KiB por escuta, ~50 MB em 24 mil escutas; MusicBrainz fora da trava), relay/outbox (081 produz a intenção; envio durável é a 082), multiusuário (perfil sempre do token; 158-161 não mudam a ingestão).

## Riscos
Perda de escutas (200 só após o commit do recibo); contagem duplicada na transição (modo sombra); pendência sem tela sem a 083; concorrência entre processos e revisores (`resolution_version`); fuso/horário de verão só ampliam candidatos; reenvios dos clientes após 500; volume de importação (lotes curtos e cursor por ordinal); migração e reversão (backup VACUUM INTO; binário antigo recusa o schema novo); exclusão sem tombstones (doc-040 §4).

## Ordem
doc-040 §10: 133 → 081/134 → 082/083; 085/086 por último. A v4 proíbe `accepted`/`event_id` até a 133.

## Plano
1. `internal/ingest/envelope.go` (tipos, snapshot, digest, precisão de tempo). 2. `ingest_receipts.go` (ReceiveBatch em transação curta). 3. `internal/dedup` (Compare → Igual/Diferente/Revisar). 4. API em modo sombra (200 só após commit; 503). 5. Importadores em sombra (digest + ordinal, Pending, cursor). 6. Ativação (081b) após a 133/134: evento + outbox na mesma transação; Advisor effort=high obrigatório sobre o diff. 7. CHANGELOG e evidências.
