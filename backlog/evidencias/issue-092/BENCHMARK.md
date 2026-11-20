# ISSUE-092 — Histórico grande

## Ocorrência autônoma na v0.47

Em banco descartável inicializado pelo binário v0.47, sem outras escutas, executar via SQLite com `foreign_keys=ON`:

```sql
INSERT INTO recordings(id) VALUES(10);
INSERT INTO occurrences(id,recording_id,presented_title) VALUES(20,10,'Standalone fixture');
INSERT INTO occurrence_credits(occurrence_id,position,name) VALUES(20,0,'Standalone artist');
INSERT INTO listens(profile_id,track_id,listened_at,event_id)
VALUES(1,20,1672531200,'probe');
```

No prefixo `/apis/web/v1/profiles/default`, consultar com `?year=2023&tz=UTC`:

| Endpoint | Resultado observado |
| --- | --- |
| `/stats` | Uma escuta, uma faixa, um dia ativo, sequência 1, zero álbuns, zero artistas e zero minutos. |
| `/listens` | Total 1, itens vazios, próxima página indicada. |
| `/top/tracks` | Total 0 e itens vazios. |

A inserção SQL verifica o leitor num estado que o schema permite. A admissão sem lançamento pelo binário está bloqueada por padrão em `pending_identity/release_missing`.

## ESTADO

- Feito: ensaio HTTP preliminar e reprodução isolada da projeção autônoma. Script de carga versionado.
- Pendente: resultados de 300 mil e 1 milhão, análise de planos e custo de escrita, propostas e registro no Backlog. AC1 depende do usuário.
