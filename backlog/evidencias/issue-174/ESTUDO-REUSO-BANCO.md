# ISSUE-174 — estudo de reaproveitamento das tabelas existentes

Estudo de 24/11/2026. Nenhuma migração ou alteração de comportamento é proposta como já aprovada. Base de comparação: `main` no schema 23; implementação em revisão na branch da 174 chega ao schema 26.

## O que existe

- `artists` e `recordings` são entidades compartilhadas; cada linha possui um `musicbrainz_id` único ou nulo. Em `recordings`, `mbid_source` atualmente distingue `legacy_declared` e `validated`, não provedor nem perfil. `occurrences` vincula uma gravação a um lançamento e `occurrence_credits` liga seus nomes de crédito aos artistas; também são compartilhadas.
- `listens` contém o perfil e a ocorrência de cada escuta. `ingest_events` contém perfil, vínculos e o snapshot imutável do que foi recebido. Os recibos têm regras próprias de ciclo de vida; não são uma tabela permanente de metadados de catálogo.
- IDs recebidos já podem compor o catálogo ou ser preservados no recibo. A necessidade nova é guardar a afirmação adicional obtida após a escuta, com fonte, data e escopo, sem substituir IDs recebidos e sem transformar resultado externo em identidade validada.

## Alternativas

### A. Colunas em `artists` e `recordings`

Adicionar `mbid_profile_id`, `mbid_provider` e `mbid_observed_at` é simples para um único valor global. Não representa, numa única linha compartilhada, valores diferentes dos perfis A e B, fontes simultâneas ou um valor público e outro privado. Uma coluna de perfil também não muda automaticamente todas as consultas existentes que já usam `musicbrainz_id`, nem protege contra vazamento ou associações globais. Ao excluir o perfil dono, é necessário escolher como recuperar ou descartar o dado. Duplicar artistas e gravações por perfil exigiria rever unicidade de MBID, vínculos, deduplicação, fusão, rankings e migrações: é uma mudança arquitetural grande.

### B. Colunas ou JSON nos recibos existentes

`ingest_events` já tem `profile_id`, `occurrence_id`, `recording_id` e `original_snapshot`. É tecnicamente possível acrescentar campos separados para o resultado externo, ou usar `ingest_event_details` para uma observação tipada; não modificar o snapshot original. Uma estrutura multivalorada para artistas precisa manter nome e ID associados, origem, data e resultado ambíguo. Essa opção evita tabelas novas de declarações, mas repete resultado por recibo, torna retenção do dado dependente de recibos e exige definir o que ocorre ao remover ou purgar o recibo que o continha. Pesquisas por artista e por perfil podem exigir leituras JSON históricas ou índices/projeções; é necessário medir com histórico representativo antes de aceitá-la. A tabela de detalhes tem política própria de purga: não reutilizar sem revisar seu contrato e seus consumidores.

### C. Reutilizar escutas por perfil

Adicionar um ou mais campos de enriquecimento a `listens` dá escopo por perfil e mantém dados ligados à escuta, sem nova tabela. Porém uma música tocada milhares de vezes repetiria metadados, enquanto declarações por artista continuam multivaloradas. Definir qual escuta carrega o valor, transferi-lo na exclusão e manter exportação e consultas coerentes acrescenta complexidade. Colunas em `occurrence_credits` também não resolvem o escopo: o crédito é compartilhado, não pertence a um perfil.

### D. Uma tabela de declarações por perfil, além das existentes

Uma tabela única poderia registrar perfil, tipo de entidade, ID da entidade, MBID, fonte e instante, permitindo conflitos e vários valores. Reduziria as duas tabelas de declarações da branch a uma, mas referências polimórficas exigiriam validação e limpeza explícitas ou uma tabela comum de entidades: o SQLite não pode aplicar uma chave estrangeira condicional a `artists` ou `recordings`. Alternativamente, duas colunas de FK opcionais com `CHECK` exatamente uma não nula mantêm integridade, com índices parciais por tipo; medir fusão, exclusão e consultas. Uma tabela extra bem delimitada pode ser mais simples que duplicar entidades ou embutir listas em recibos.

## Fila e observações

A escolha sobre onde guardar MBIDs é independente da execução assíncrona. A fila proposta na branch usa `enrichment_jobs` e `enrichment_configuration`. Antes de preservá-las, avaliar se a infraestrutura existente de `ingest_events` ou `outbox` já oferece fila separada, recuperação e limites sem confundir uma falha de provedor com falha de ingestão ou de relay; reutilização só é válida se preservar essas semânticas. Um worker em memória elimina tabela de fila, mas perde trabalho após reinício; recuperação exige varredura ou outra fonte durável.

A tabela `enrichment_received_artist_credits` é uma projeção para acelerar consulta de precedência; não é necessariamente indispensável. Remover a necessidade da consulta histórica, limitar a verificação ao evento, ou usar índice/estrutura existente deve ser avaliado com provas de correção e plano de consulta, sem perder IDs de artistas individuais.

## Recomendação e ensaios

Não adicionar `profile_id` diretamente a `artists` e `recordings` como solução de conflito: elas são compartilhadas e suas chaves atuais não representam várias declarações. Antes de aceitar as cinco tabelas propostas, testar um protótipo de **recibos existentes** com retenção explicitada e compará-lo a **uma única tabela de declarações com duas FKs opcionais**. Preservar a fila existente apenas se a alternativa for demonstravelmente equivalente. Não concluir que uma abordagem é mais barata pela contagem de tabelas: medir migração e consultas, exclusão e reimportação.

Casos obrigatórios: dois perfis com IDs distintos; dois artistas com um ID ausente; ID recebido depois do enriquecimento; gravação em vários lançamentos; conflito e edição manual; exclusão de recibo, escuta, perfil, artista ou ocorrência; fusão; reinício durante busca; reimportação; importação de banco v23 e atualização desde Koito v2. Medir leitura de artista, exportação e exclusão em lote com dados na escala de produção. Manter a ordem da ISSUE-195 e não mudar as migrações existentes sem uma estratégia de equivalência aprovada.

## Limites deste estudo

Não foi implementado protótipo nem medido desempenho das alternativas. A branch existente permanece intacta e sem aceite; os critérios de retenção, o papel exato de `ingest_event_details` e as implicações de reuso da outbox exigem inspeção e testes antes de escolher um modelo definitivo.

## Direção proposta pelo usuário: uma tabela de metadados

O usuário propõe uma tabela única para dados produzidos pelo subsistema, com MBIDs, gêneros, tags, ano/data, possível país, relações com artistas e músicas e proveniência. Avaliar também lançamento, que é uma entidade distinta de gravação. Esta direção substitui a preferência preliminar por tabelas de declarações separadas, mas o desenho técnico ainda precisa ser validado.

Modelo candidato: uma linha por valor observado, com `profile_id` opcional para dado público ou obrigatório para dado privado; exatamente uma referência real a artista, gravação ou lançamento; `kind` controlado; valor tipado ou validado, fonte/instância e instante. Listas como gêneros são várias linhas, não texto separado por ponto e vírgula. Dados recebidos e dados de provedores devem ter precedência definida; não sobrescrever o MBID canônico existente nem alterar snapshot original. Ter um único campo de referência polimórfica sem FK perde integridade; colunas de FK opcionais com CHECK de exatamente uma preenchida são uma alternativa. Ao incluir novas entidades o schema precisará de migração; comparar esse custo com uma tabela de entidades comum.

Semântica: distinguir MBID de artista/gravação/lançamento; data específica do lançamento e primeiro lançamento com precisão parcial; gênero de tag livre; país de origem do artista do país de lançamento. `US`, `BR` e `GB` são códigos ISO; `UK` é uso comum, mas não o código ISO alpha-2 oficial do Reino Unido. Não inferir país de artista a partir da edição. Índices por perfil, entidade e tipo devem ser escolhidos a partir dos gráficos e dos testes; valores conflitantes ficam identificáveis.

Recibos de escuta existentes mantêm o evento e o snapshot de entrada. Pode-se guardar referência opcional ao recibo como evidência, mas com retenção independente e regra explícita de `ON DELETE`; não copiar recibos completos à tabela de metadados nem perder valores ao expirar recibos. A fila de requisições e o estado de saúde são conceitos diferentes dos valores musicais; avaliar reaproveitamento de infraestrutura existente sem presumir que uma tabela de dados substitua uma fila durável. A tabela de projeção de créditos da branch deve ser reavaliada para eliminação, medindo custo e preservando precedência por artista.

Ensaios antes de reestruturar: restrições e unicidade para NULLs no SQLite (índices parciais); perfis divergentes e dados públicos; artista e gravação com mesmo ID numérico; migração sem perda dos MBIDs atuais; exclusão, mesclagem, backup/exportação/reimportação e permissões; desempenho de gráficos e recuperação da fila. Não misturar a origem da observação com identidade validada. Determinar como país, ano e tags serão obtidos de cada fonte antes de prometê-los na 174.

## Decisão aplicada na branch

A migração 027 reúne as duas tabelas de declarações em `metadata_values`. Há três FKs opcionais de catálogo e um CHECK que exige exatamente uma delas. Cada linha tem perfil obrigatório, tipo, valor, fonte, instância e instante. Não há escopo público implícito. A 174 só produz e consome MBIDs. O schema reserva gêneros, tags, ano de gravação, data de lançamento e país de origem do artista como valores individuais. Os próximos provedores ainda precisam definir validação, precisão e representação na exportação de cada tipo.

Cada novo valor de provedor aponta para a escuta resolvida, com FK para o evento e verificação de perfil e vínculo com a entidade. Para artista, a gravação exige também um crédito inequívoco antes e depois da rede. Identificadores de escuta do banco de origem não são copiados pela exportação. A restauração vincula o valor a uma escuta local que sustenta a associação. A migração mantém valores antigos sem origem quando a versão anterior já havia perdido esse vínculo, sem escolher uma escuta arbitrária.

Apagar o recibo elimina a projeção de precedência e o trabalho pendente, mas a escuta e as declarações podem sobreviver. Apagar a escuta de origem limpa o ponteiro. Declarações ficam enquanto o perfil tem uma escuta que sustente a entidade, e saem com o perfil ou a entidade. A gravação tem uma ocorrência de apoio além da FK de identidade. Isso permite acompanhar fusão e desfazer fusão sem perder a edição em que a observação nasceu. Se a ocorrência de apoio for excluída, só outras ocorrências da mesma gravação ouvidas pelo perfil recebem o valor, com vínculo de origem em branco.

O catálogo mantém a precedência do ID já salvo e o snapshot conserva o que foi recebido. IDs presentes no recibo suprimem sugestões externas da mesma entidade. Valores contraditórios ficam no escopo do perfil e não são promovidos a IDs de catálogo. A exportação v2 preserva as declarações e suas fontes e datas, inclusive conflitos. Arquivos v1 omitem essas declarações.

A projeção de nomes por recibo foi mantida porque registra evidência de precedência de entrada, sem copiar o valor musical. Ela evita decodificar o histórico na página de artista e permite remover a evidência precisamente quando o recibo sai. Um índice parcial cumpre o papel equivalente para gravações. As filas de ingestão e relay têm ciclos de vida diferentes. A fila de enriquecimento continua durável, com lease e teto de tentativas, para retomar o trabalho após interrupção sem varrer o histórico nem reabrir uma decisão de ingestão.

### Decisões pendentes

Conflitos permanecem armazenados e não criam pendência de ingestão. Uma futura revisão pode reunir candidatos por perfil, entidade e tipo, apresentar as evidências e gravar uma escolha explícita e auditável. Ainda é necessário decidir quem pode resolver conflitos, se uma escolha é privada ou global e como a escolha aparece nos consumidores. Nenhuma interface, promoção automática ou política pública foi definida nesta etapa. A integração de metadados adicionais e eventual consolidação das migrações ficam com as issues correspondentes.
