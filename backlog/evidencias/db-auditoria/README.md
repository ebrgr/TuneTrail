# Auditoria de uso e repetição dos dados do banco

Evidências do estudo doc-047 (`backlog/docs/estudos`). Medido em 23/11/2026 num banco novo da v0.74 (schema 23) que recebeu 5.000 escutas pela API.

- `colunas.csv`: uma linha por coluna (283), com a evidência e um veredito proposto. Filtre por `veredito` ou `tabela`.
- `audit_db_usage.py`: mede um banco (somente leitura) e escreve o JSON de evidências.
- `classify_columns.py`: transforma o JSON no CSV. As decisões manuais e seus motivos estão no topo do arquivo.

```
python3 audit_db_usage.py  BANCO.db  RAIZ_DO_REPOSITORIO  evidencias.json
python3 classify_columns.py BANCO.db evidencias.json colunas.csv
```

O CSV traz só contagens e tamanhos, nenhum dado de escuta. Rode sobre uma cópia do banco, nunca sobre o arquivo em uso.
