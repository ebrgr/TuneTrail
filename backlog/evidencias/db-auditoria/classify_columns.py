#!/usr/bin/env python3
"""Turns the audit JSON into one CSV row per column with a class and a proposed verdict.

Usage: classify_columns.py DATABASE AUDIT_JSON OUTPUT_CSV
The verdict is a PROPOSAL for review, never an instruction: it combines the class derived from the
evidence with a short list of manual decisions, each with its reason, written below.
"""
import csv
import json
import sqlite3
import sys

db_path, audit_path, out_path = sys.argv[1:4]
r = json.load(open(audit_path))
db = sqlite3.connect(f'file:{db_path}?mode=ro', uri=True)
info = {t: {x[1]: x for x in db.execute(f'pragma table_xinfo("{t}")')} for t in r['tables']}
pk = {t: {c for c, x in info[t].items() if x[5]} for t in r['tables']}
fk = {t: {x[3] for x in db.execute(f'pragma foreign_key_list("{t}")')} for t in r['tables']}
repeat = {}
for t, pairs in r['repeats'].items():
    for a, b in pairs:
        repeat.setdefault((t, a), []).append(b)
        repeat.setdefault((t, b), []).append(a)

NODATA = 'Nenhuma referência no Go de produção nem em índice, gatilho ou view.'
LEGACY = 'Constante legada: padrão da época da sombra; todo recibo entra com um único valor.'
DETAIL = 'Tabela sem nenhum escritor no código de produção (só a rotina de retenção a cita).'
RESERVED = 'Constante hoje, mas reservada para outras fontes (ISSUE-086 a 091).'
STATE = 'Máquina de estados do recibo: constante enquanto nada falha, varia nos demais casos.'
OVERRIDE = {
    # ingest_events
    **{f'ingest_events.{c}': ('remover', NODATA) for c in ('uncertainty_ms', 'client_name', 'client_version', 'normalized_snapshot', 'normalizer_version')},
    **{f'ingest_events.{c}': ('remover', LEGACY) for c in ('mode', 'resolver_variant', 'delivery_policy')},
    'ingest_events.batch_id': ('revisar', 'Só escrita (36 B por recibo): expor no recibo ou remover.'),
    'ingest_events.listened_at': ('simplificar', 'Texto RFC3339 de 30 B que repete listens.listened_at (inteiro): guardar como inteiro.'),
    'ingest_events.closed_at': ('simplificar', 'Igual a resolved_at em todos os recibos aceitos: uma coluna de tempo basta (conferir os demais estados).'),
    **{f'ingest_events.{c}': ('simplificar', 'Resultado padrão do caminho normal (idêntico em todos os recibos): gravar só quando fugir do padrão.')
       for c in ('reason_codes', 'evidence', 'rule_version')},
    'ingest_events.api_key_id': ('manter', 'Atribuição; sustenta a coluna gerada inbox_key.'),
    **{f'ingest_events.{c}': ('manter', RESERVED) for c in ('schema_version', 'adapter_version', 'digest_version', 'time_precision', 'time_semantics', 'operation', 'source_type', 'external_id_trusted')},
    **{f'ingest_events.{c}': ('manter', STATE) for c in ('attempt_count', 'work_version', 'retry_attempt_base', 'claim_attempt_base', 'identity_status', 'duplicate_decision', 'state', 'resolution_version')},
    # listens
    'listens.review_distinct': ('remover', 'Era a exceção do escritor v9; sem leitura nem escrita desde a ISSUE-194.'),
    'listens.user_id': ('remover', 'Repete o dono já guardado em profile_id; vazia nas escutas novas. Só tem valor no histórico migrado do v2, onde profile_id já guarda o mesmo dono.'),
    # integrations
    **{f'integrations.{c}': ('revisar', 'Sem uso hoje; pode estar reservada para as ISSUE-086 a 091.') for c in ('token_ref', 'revoked_at', 'diagnostic_enabled')},
    'integrations.details_retention_days': ('remover', 'Só serve às tabelas de detalhes, que não têm escritor.'),
    'relay_sent_archive.sent_count': ('revisar', 'Só escrita por gatilho; ninguém lê. Mostrar a contagem ou remover.'),
    'sessions.created_at': ('revisar', 'Só escrita no Go; confirmar se alguma tela a usa.'),
    'sessions.persistent': ('revisar', 'Só escrita no Go; confirmar se alguma tela a usa.'),
}
TABLE_RULE = {  # whole tables
    'ingest_event_details': ('remover', DETAIL), 'outbox_details': ('remover', DETAIL),
    'legacy_relay_binding': ('remover', 'Marca a associação única do relay legado por variável de ambiente; sai com esse bootstrap (decisão do usuário).'),
    'relay_delivery_gate': ('remover', 'Constante: vale durable em todo banco após o primeiro start.'),
    'event_origins': ('simplificar', 'Tabela de 9 valores vira CHECK em listens.origin.'),
    'profile_ingest_review': ('simplificar', 'Vira uma coluna de listening_profiles.'),
    'profile_relays': ('simplificar', 'Relação 1:1 com relay_destinations: fundir as duas e eliminar 5 gatilhos de sincronia.'),
    'ingest_revisions': ('simplificar', 'Repete 100% a decisão do recibo e ninguém lê: gravar só decisões de usuário e de fusões.'),
    'listen_export_order': ('simplificar', 'Uma linha por escuta repetindo event_id (94 B por escuta): trocar por coluna de sequência.'),
    'tracks': ('simplificar', 'Ponte legada do Koito v2, repetida em occurrences e recordings (pacote C).'),
}
DEFAULT = {
    'chave': ('manter', 'Chave primária ou estrangeira.'),
    'gerada': ('manter', 'Coluna gerada que sustenta índice ou contagem.'),
    'ativa': ('manter', 'Lida e escrita, com valores variados.'),
    'ativa, constante': ('manter', 'Constante nestes dados, mas faz parte do contrato da tabela.'),
    'ativa, vazia': ('manter', 'Vazia nestes dados; depende do cliente ou da importação.'),
    'lida, nunca escrita': ('manter', 'Preenchida por padrão, por gatilho ou por migração.'),
    'só no banco': ('manter', 'Usada por índice ou gatilho.'),
    'só escrita': ('revisar', 'Recebe dados e nenhuma consulta lê.'),
    'morta': ('remover', NODATA),
    'revisar': ('revisar', 'Referenciada só por constantes de Go; conferir à mão.'),
}


def classify(t, c):
    rd, wr, tk, du = r['reads'][t][c], r['writes'][t][c], r['token_refs'][t][c], r['dbuse'][t][c]
    pop = r['population'][t][c]
    n = r['rowcount'][t]
    if c in pk[t] or c in fk[t]:
        return 'chave'
    if c in r['generated'].get(t, []):
        return 'gerada'
    if rd == 0 and wr == 0 and tk == 0:
        return 'só no banco' if du else 'morta'
    if rd == 0 and wr == 0:
        return 'revisar'
    if wr > 0 and rd == 0:
        return 'só escrita'
    if rd > 0 and wr == 0:
        return 'lida, nunca escrita'
    if pop and n > 1 and pop['nonnull'] == 0:
        return 'ativa, vazia'
    if pop and n > 1 and pop['distinct'] == 1:
        return 'ativa, constante'
    return 'ativa'


rows = []
for t in r['tables']:
    if t == 'goose_db_version':
        continue
    for c in r['columns'][t]:
        cls = classify(t, c)
        # A column decision wins; otherwise a whole-table decision applies to every column, keys included.
        verdict, why = OVERRIDE.get(f'{t}.{c}') or TABLE_RULE.get(t) or DEFAULT[cls]
        pop = r['population'][t][c]
        rows.append(dict(
            tabela=t, coluna=c, tipo=info[t][c][2] or '', classe=cls,
            leituras_go=r['reads'][t][c], escritas_go=r['writes'][t][c], usada_pelo_banco=' '.join(r['dbuse'][t][c]),
            linhas_nao_nulas='' if pop is None else pop['nonnull'], valores_distintos='' if pop is None else pop['distinct'],
            tamanho_medio_bytes='' if not pop or pop['avg_len'] is None else pop['avg_len'],
            repete_outra_coluna=' '.join(repeat.get((t, c), [])), veredito=verdict, motivo=why))
with open(out_path, 'w', newline='') as f:
    w = csv.DictWriter(f, fieldnames=list(rows[0]))
    w.writeheader()
    w.writerows(rows)
from collections import Counter
print(len(rows), 'colunas ->', out_path)
print(dict(Counter(x['veredito'] for x in rows)))
