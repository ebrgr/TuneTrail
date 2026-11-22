# Plano ISSUE-184 — central de saúde com Inbox (API key) e Outbox (relay)

Pedido do usuário em 21/11/2026 (veja a tarefa). Depende da ISSUE-181 (v0.57) e da 182 (v0.58). Schema atual v18; esta issue usa a migração v19.

## Objetivo
`/admin/health` em duas colunas no desktop: **Inbox** à esquerda (o que o TuneTrail recebe de outros serviços, como o Navidrome, e das importações) e **Outbox** à direita (o que vai para o relay). O Inbox é filtrado pela **API key** que o serviço usa para enviar (a chave é fornecida pelo TuneTrail), e mostra recebimentos por chave e o que chegou com dados faltantes ou metadados incompletos.

## Estado do código (conferir)
- O recibo (`ingest_events`) não guarda a chave: o lote do ListenBrainz sai com `source_instance_id` fixo; `api_keys` tem `label` e `profile_id` (migração 003). Confirmar no middleware de autenticação como a chave é resolvida e como passá-la ao lote (`ingest.Batch`).
- A 181 criou `ingest_source_summary` (perfil, fonte, cliente, último recibo) com gatilhos de inserção/exclusão; foi projetada para receber a dimensão da chave sem acoplar credenciais.
- A central hoje tem: fontes e última entrega, decisões, relay (fila, entregas, pausa), falhas permanentes (173/182) e pendências de revisão (181).

## Decisões fixadas
- **Chave no recibo:** migração aditiva v19 com `api_key_id` (nullable, sem FK que bloqueie a exclusão da chave) em `ingest_events`, preenchido no handler de submissão com o ID interno da chave autenticada. Recibos antigos e importações por arquivo ficam sem chave: no Inbox aparecem como "Chave desconhecida" e "Importações" (origem própria). Chave revogada/excluída aparece como "Chave removida". **Nunca gravar nem exibir a chave bruta; só o rótulo e o ID interno.**
- **Colunas:** Inbox (esquerda): recebimentos por chave/cliente, metadados faltantes e incompletos, fontes e última entrega, pendências de revisão (181) e falhas permanentes (173/182), pois são o processamento do que foi recebido. Outbox (direita): fila do relay, entregas, pausa e falhas de entrega. Duas colunas só a partir de `min-width: 64rem`; empilhado abaixo disso (Inbox primeiro).
- **Contagens do Inbox:** por chave (rótulo) e cliente: total recebido, aceitos, duplicados, inválidos e pendentes, em janelas de 24 h, 7 d e 30 d. Escolher entre consulta indexada e contadores mantidos na admissão com base na medição (limite de referência: ≤ 50 ms com 300 mil recibos); backfill dos recibos existentes.
- **Metadados:** duas faixas. "Faltante" = título ou artista ausente (bloqueia a identidade); "Incompleto" = sem álbum, sem MBID (gravação, lançamento ou artista) ou sem duração. Calcular na admissão (bitmask ou colunas), com backfill, evitando `json_extract` sobre toda a janela a cada abertura. Para cada tipo, 1 recibo de exemplo com só campos seguros do snapshot (título, artista, horário, cliente), sem corpo bruto nem tokens.
- **Acesso e escopo:** dono do perfil e root; isolamento por perfil; GET só leitura.
- Fora do escopo: descartar recibos (ISSUE-185), mudar o relay, alterar a política de retry, indicadores da ISSUE-097.

## Restrições
- Reaproveitar o resumo de fontes da 181 e as seções existentes; não quebrar as ações de reprocessar (182) e de revisão (181).
- UI: só `min-width` 40/64/90rem, `health.css` ≤ 400 linhas por arquivo (dividir em novo arquivo + manifesto se passar), hover igual ao padrão, i18n en/pt-BR; regenerar a referência ISSUE-075 (`make ui.reference`).
- Migração v19 aditiva, com guarda de downgrade e backup, como as anteriores; o backfill não pode travar o escritor por muito tempo (medir).
- Medir as consultas da central (EXPLAIN QUERY PLAN) com 300 mil recibos e registrar em `backlog/evidencias/issue-184/`.
- Testes por classe: chave nova/antiga/removida, importação, isolamento entre perfis, janelas de tempo, faltante x incompleto, ausência de segredos na saída (nenhuma chave bruta), empilhamento responsivo. `TUNETRAIL_WEB_DIR=$PWD/assets` nos testes de engine.
- Docs (guia de ingestão) e CHANGELOG [Unreleased] em inglês.
- Advisor: recomendado depois da implementação (migração e exposição de dados); acionado pelo usuário.
