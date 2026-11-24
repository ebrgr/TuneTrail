# ISSUE-174 — fluxo e decisões de implementação

## Avaliação e consulta

1. A resolução da escuta termina normalmente. Apenas recibos ListenBrainz aceitos ou duplicados, enquanto o Subsonic está configurado, criam trabalho de enriquecimento na mesma transação; recibos históricos não são varridos.
2. O trabalhador próprio reserva o recibo com lease e limite de duas reservas, incluindo recuperações após interrupção. Serviço desligado não cria trabalho novo; ao desligar, o trabalho anterior pendente é encerrado. Falha externa não altera a decisão de ingestão nem provoca repetição infinita.
3. O pedido é filtrado por entidade: IDs recebidos ou já conhecidos no escopo do perfil e da instância não são substituídos. Créditos com IDs parciais sem associação inequívoca não são atribuídos por posição.
4. O provedor procura pelo título via `search3`, requer correspondência de título, artista e álbum, confronta duração, posição e IDs disponíveis e recusa múltiplos candidatos ou uma página possivelmente truncada. A consulta de artista usa o ID nativo do artista ligado à música, com conferência do nome. O MBID da música requer `mediaType=song`.
5. As chamadas usam POST, prazo, limite de taxa e cache limitado por perfil e configuração. Credenciais não são registradas nem incluídas nos arquivos de exportação.

## Gravação e consumo

Depois da rede, uma transação curta revalida o recibo, perfil, versão, ocorrência, gravação, créditos e metadados do catálogo. MBIDs externos são declarações com fonte e instante. O snapshot e os IDs recebidos mantêm seus valores. Declarações conflitantes ficam no perfil sem escolha automática.

Cada valor pertence à escuta que comprova o perfil e a associação à entidade. Apagar apenas o recibo elimina sua evidência de precedência e seu trabalho, mas preserva a escuta e os valores. Apagar a escuta remove seus valores por FK. Fusões podem transferir valores quando a mesma escuta continua sustentando a entidade de destino.

A exportação v2 consulta declarações por página e as inclui junto da respectiva escuta, com procedência e conflitos. Ordinais distinguem instâncias sem expor o hash de credenciais. A importação revalida a associação da entidade à escuta restaurada no perfil de destino. O v1 omite declarações por não representar procedência.

## Tabelas da migração 024

- `metadata_values` guarda valores por perfil, fonte, instância, data e escuta. Três FKs de catálogo com um CHECK exigem exatamente uma entidade por valor. A âncora de ocorrência auxilia a reconciliação de gravações.
- `enrichment_received_artist_credits` projeta nomes de créditos com ID recebido, a partir dos recibos. Os nomes indexados evitam decodificar o histórico ao abrir a página de um artista. A projeção é preenchida uma vez no upgrade.
- `enrichment_jobs` guarda um trabalho por recibo, com lease, teto de duas reservas e resultado terminal. É independente da fila de ingestão.
- `enrichment_configuration` guarda a chave que habilita a criação de trabalho quando o provedor está configurado.

A migração parte do schema 23 sem enfileirar o histórico. A cadeia de desenvolvimento 024 a 028 e seu resguardo intermediário foram retirados antes da publicação. A comparação em cópias descartáveis do mesmo v23 encontrou 177 objetos equivalentes no `sqlite_schema`, excluindo o controle de versões do Goose e a antiga tabela de resguardo com seu índice. O upgrade suportado testa backup, integridade e preservação dos dados v23. O bootstrap recusa as prévias de desenvolvimento 24 a 28. O [estudo de reuso](ESTUDO-REUSO-BANCO.md) detalha os tipos reservados e as decisões de conflito ainda abertas.
