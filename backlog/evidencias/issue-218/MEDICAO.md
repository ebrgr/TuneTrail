# ISSUE-218 — medição preliminar de armazenamento

A medição sintética de `python3 backlog/evidencias/issue-218/medir_metadados.py` cria uma tabela `metadata_values` isolada com a definição e todos os índices da migração 024, permitindo os novos tipos, e compara o tamanho do arquivo SQLite após `VACUUM`, antes e depois de inserir dados.

Resultado local: **10.000 escutas, oito valores por escuta, 3.806,82 bytes adicionais por escuta** (arquivo inicial 45.056 bytes, final 38.113.280 bytes). Cada escuta tem três gêneros, ano, duas datas, ISRC e BPM; cada linha tem uma impressão digital fictícia de 64 caracteres, fonte `navidrome-subsonic`, data de observação, ID de evento de 36 caracteres e vínculos fictícios. Há 1.000 gravações e 100 lançamentos repetidos. O cenário usa uma data de primeiro lançamento hipoteticamente vinculada à edição apenas para estimar tamanho, sem decidir sua modelagem.

A medição isola tabela e índices existentes: não mede outros dados nem jobs de enriquecimento, não inclui dados reais e não demonstra o custo exato de uma implementação final. Os índices usam o vínculo por evento atual; o custo inclui dados e índices e está muito acima do limite aproximado de 500 bytes definido para parada.

## Alternativas medidas e decisão

`python3 backlog/evidencias/issue-218/medir_alternativas.py` (coordenador, 27/11/2026), com os mesmos oito valores e 10.000 escutas. Bytes por escuta, conforme quantas vezes cada faixa é ouvida:

| Modelo | 1 vez | 3 vezes | 10 vezes |
| --- | --- | --- | --- |
| A. Por escuta (migração 024) | 3.816 | 3.815 | 3.807 |
| B. Por escuta, origem por referência | 2.004 | 1.998 | 1.993 |
| C. Por entidade | 1.648 | 544 | 163 |
| **D. Por entidade, origem por referência (aprovado)** | **573** | **189** | **56** |

São estimativas sintéticas, só da tabela de metadados e dos índices; o cadastro de fontes não entra na conta por ter poucas linhas. O usuário escolheu o modelo D em 27/11/2026 (decision-012).
