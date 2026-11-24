# ISSUE-174 — início do subsistema de enriquecimento

Plano inicial, 24/11/2026. A implementação e os critérios de aceite permanecem pendentes.

## Contrato

- Preservar os MBIDs recebidos e procurar apenas identificadores ausentes de gravação e de cada artista. Não usar campos disponíveis de uma entidade como justificativa para substituir identificadores de outra.
- Admissão e confirmação de escuta não esperam rede; chamadas com contexto e timeout ocorrem fora do lock de escrita. Uma falha ou ambiguidade de provedor não gera pendência de ingestão nem retry infinito.
- Dados externos são declarações com fonte e data, não prova validada. Não reescrever snapshot original, histórico ou identificadores existentes; revalidar identidade e escopo em transação após retorno da rede.
- O contrato central precisa comportar ISSUE-175 e ISSUE-055 sem acoplar o provedor ao cliente de imagens. Escopo por perfil e instância Subsonic é parte da chave de cache e de trabalho; nenhuma credencial em log.

## Evidências e pontos de integração

- `internal/ingest/listenbrainz.go:14` seleciona os campos recebidos e `internal/ingest/envelope.go:29` define o snapshot; ambos preservam somente informação explicitamente escolhida.
- `internal/db/sqlite/ingest_resolve.go:63` faz a resolução fora da transação de escrita e `engine/ingest_recovery.go:16` recupera recibos pendentes; não misturar falhas de enriquecimento com retentativas da ingestão.
- `internal/db/sqlite/ingest_occurrence_catalog.go:24` e `:59` tratam MBIDs declarados e créditos; não associar IDs de artistas por posição quando a lista não for inequívoca.
- `internal/db/sqlite/bootstrap.go:22` indica schema 23. `recordings.mbid_source` restringe os valores a `legacy_declared` e `validated`; registrar origem e data de maneira aditiva e compatível com identidade, exportação e migração, sem elevar declaração a validação.
- `internal/images/subsonic.go` contém configuração e transporte de imagens; examinar extração segura de infraestrutura comum sem alterar comportamento existente. O Navidrome testado aceitou POST e a escuta não fornece ID interno; `search3` precisa filtrar candidatos por identidade e detectar truncamento.

## Sequência e verificações

1. Projetar persistência e propriedade da fila, política de reconciliação e estados `encontrado`, `ausente`, `ambíguo`, `desabilitado` e falha transitória, respeitando exclusão, merges, reinício e perfil. Diferenciar trabalho para escuta nova da futura varredura histórica (213).
2. Implementar cliente e correspondência Subsonic com testes de múltiplos candidatos, versões, duração divergente, resultados truncados, dados incompletos, credenciais e cancelamento. Reutilizar respostas sanitizadas; não incluir dados da biblioteca ou credenciais reais.
3. Ligar a fila ao fluxo existente, sem caminho de rede no ACK e sem segurar o lock SQLite. Testar corrida entre resultado remoto e edição, exclusão, merge ou recebimento de MBID e reinício com trabalho em andamento.
4. Testar schema novo e atualização do v23, preservação de dados, origem por entidade, isolamento de perfis, provedor indisponível e desligado, repetição e recuperação. Executar `make check` e revisão prevista no projeto.

A data e forma da migração devem ser coordenadas com a ISSUE-195: o plano antigo colocava 174 depois da consolidação, mas o usuário pediu começar por 174. Não alterar dependências automaticamente sem conferir as demais tarefas e o plano de upgrade.

## Adaptação aprovada para a tabela única

A gravação de valores passa a usar `metadata_values` a partir do schema 27. A estratégia mantém uma FK de entidade por observação, valida no banco a origem dentro do perfil e usa índices separados por entidade. A projeção de créditos de entrada e a fila com lease permanecem estruturas de apoio. O schema 28 exige vínculo ativo com escuta comprovada. A exclusão da escuta remove o valor e a migração mantém valores antigos sem prova em resguardo separado. A validação cobre os caminhos de upgrade v23 e de preservação de declarações da branch, a privacidade, a retenção, a reconciliação, a exportação v2 e a recuperação da fila. A política futura de resolução explícita está descrita no estudo de reuso do banco.
