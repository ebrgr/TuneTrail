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

## Migração 026 proposta e registros de execução

Em 27/11, com o modelo D e timestamps RFC3339Nano fictícios (precisão de nanossegundos), o script `medir_modelo_d.py` mede a migração 026 proposta e inclui uma linha do cadastro de fontes, seus índices e os índices da tabela descritiva. São 10.000 escutas, três gêneros, ano, duas datas, ISRC e BPM, com um lançamento para cada dez faixas. `VACUUM` é executado antes e depois. As chaves estrangeiras e os gatilhos de escopo não são ativados no ensaio isolado; a medição compara somente páginas de tabelas e índices.

| Escutas por faixa | Metadados e fonte, bytes/escuta |
| --- | ---: |
| 1 | 637,75 |
| 3 | 210,12 |
| 10 | 62,67 |

A tabela isolada fica abaixo de 650 B. Porém, o executor atual grava uma linha em `enrichment_provider_runs` para cada resposta `found`, em cada cadeia. Com a implementação parcial, uma resposta com novos metadados pode ser `found` tanto na identidade quanto na cadeia de metadados, inclusive quando os mesmos valores já existem. O script `medir_modelo_d_com_execucoes.py` inclui duas linhas fictícias por escuta nessa tabela existente, com suas chaves e índices:

| Escutas por faixa | Metadados, fonte e duas execuções, bytes/escuta |
| --- | ---: |
| 1 | **941,26** |
| 3 | 513,64 |
| 10 | 366,18 |

Este é um cenário de pior caso plausível da implementação parcial, não uma medição de um banco de produção. É uma comparação contra nenhuma linha anterior para essas duas execuções; onde a identidade já registraria uma execução antes da mudança, o acréscimo líquido será menor. O ensaio não mede jobs, outras tabelas nem tráfego de rede. A soma supera o limite de parada de aproximadamente 650 B no cenário de uma escuta por faixa. A implementação está interrompida para decisão; evitar gravações de execuções por escuta exige estudar e validar a recuperação dos jobs, não apenas omitir as linhas.

## Resultado com remoção dos sucessos

Após a decisão da opção 1, `python3 backlog/evidencias/issue-218/medir_modelo_d_final.py` usa a definição da migração 026 e a tabela real de execuções da 025, insere a fonte e, por escuta, confirma a inserção de um registro `found`, grava os valores, apaga o sucesso e confirma a remoção. **Não executa VACUUM após as remoções.** Usa dados sintéticos e timestamp com nanossegundos, em 10.000 escutas, com um lançamento por dez faixas; a tabela e os índices são isolados das outras tabelas do aplicativo. A execução mede o arquivo SQLite principal, sem WAL, jobs ou backups.

| Escutas por faixa | Bytes por escuta |
| --- | ---: |
| 1 | **671,33** |
| 3 | 220,36 |
| 10 | 65,54 |

O pior cenário fica abaixo do limite de aproximadamente 700 bytes. Os valores são estimativas de armazenamento, não medições em dados reais. Uma busca bem-sucedida do Navidrome já coleta metadados durante a identidade e evita outra consulta do mesmo provedor na cadeia de metadados na mesma passagem; o script mantém o registro de sucesso até o fechamento simulado.

## Correções do parecer — gatilhos e armazenamento

A migração 026 foi medida novamente após separar as consultas por entidade. O script `medir_gatilhos.py` monta tabelas e índices reais em SQLite em memória, com perfis e valores fictícios, desativa as chaves estrangeiras para isolar os gatilhos e mede inserção de metadados, inserção repetida, exclusões individuais e exclusão em lote. Cada grupo de dez faixas compartilha artista e lançamento; o lote contém metadados para todas as faixas. Tempos são médias de uma execução local, não garantias para produção.

| Escutas | Inserir ms | Repetir ms | Excluir escuta ms | Excluir ocorrência ms | Excluir faixa ms | Lote ms/escuta |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1.000 | 0,061 | 0,049 | 0,039 | 0,031 | 0,017 | 0,013 |
| 10.000 | 0,068 | 0,058 | 0,056 | 0,040 | 0,019 | 0,015 |
| 100.000 | 0,068 | 0,053 | 0,069 | 0,050 | 0,028 | 0,020 |

Com `--shared`, um artista e um lançamento concentram toda a biblioteca: a exclusão em lote levou 0,011 / 0,016 / 0,016 ms por escuta para 1.000 / 10.000 / 100.000 escutas. Com `--shared --selective`, 16 escutas de outro artista permanecem durante a exclusão parcial; o lote levou 0,011 / 0,011 / 0,013 ms por escuta. A busca curta perto da faixa afetada e no início do perfil evita repetir uma varredura longa nesses casos; quando não encontra suporte, há consulta de fallback por artista ou lançamento, que ainda pode depender da quantidade de faixas dessa entidade. Os planos, inclusive dos caminhos limitados a 16, estão em `PLANOS-GATILHOS.txt`.

A tabela `metadata_retries` usa chave primária por perfil, ocorrência e fonte. A medição de bytes foi repetida com a migração final, inserindo e removendo o marcador de falha parcial e a execução bem-sucedida em transações separadas e sem VACUUM após a remoção: **671,33 / 220,36 / 65,54 B por escuta** para 1 / 3 / 10 reproduções por faixa, abaixo do limite de 700 B no primeiro cenário. Os números abrangem as tabelas isoladas, a fonte, os índices e os registros transitórios exercitados; não incluem o banco completo ou dados reais.
