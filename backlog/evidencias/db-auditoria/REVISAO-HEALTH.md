# Central de Saúde · revisão de Inbox, ingestão e Outbox

Revisão sobre a v0.75, schema 24, a partir do doc-047 (v0.74, schema 23). As cifras do estudo anterior descrevem 5.000 escutas normais pela API, sem carga representativa de falhas ou relay. Nenhuma alteração de comportamento ou migração nesta revisão.

| Parte | Fonte de verdade | Funções a preservar |
|---|---|---|
| Inbox | `ingest_events`, resumo diário `ingest_inbox_daily`, último recibo por fonte em `ingest_source_summary` | Períodos de 1, 7 e 30 dias, chave e cliente (inclusive chave excluída), totais por estado, qualidade de metadados, exemplos e última submissão |
| Ingestão | `ingest_events`, `listens`, decisões e trilha de revisão | Deduplicação, revisão humana, explicação por escuta, falhas estacionadas, reprocessamento e descarte confirmado sem escuta |
| Outbox | `outbox`, `relay_destinations`, `profile_relays`, `relay_allowlist` | Entrega durável, retries, pausa, expiração, fila por perfil, destino aprovado e proteção contra resultados de uma configuração anterior |

## Simplificações por ordem de execução

1. **Limitar a leitura ao que a página pede.** O handler em `internal/adminui/ingest_health.go` carrega toda a Central antes de validar e aplicar os filtros da Inbox. `IngestInbox` já recebe filtro de chave, mas o handler passa `nil`; o store busca até dois exemplos para cada grupo e só depois a página descarta os outros grupos. Separar as opções do seletor e a consulta de exemplos, preservando a seleção de chaves excluídas, o período e a contagem filtrada. Dividir a leitura da Central por seção, mantendo links diretos, formulários, acesso por perfil e contadores. Nesta etapa o schema pode continuar igual. Provar que a página e os atalhos preservam todas as funções e que uma consulta filtrada deixa de buscar exemplos das outras chaves.
2. **Enxugar a escrita do recibo pelas ISSUE-203 e 204.** Remover estruturas de detalhes sem escritor de produção e o legado de sombra, após confirmar os leitores e gatilhos no schema final. Na v0.75, o gatilho `trg_enrichment_enqueue` também usa `mode='active'` e precisa entrar no inventário. O doc-047 mediu cerca de 194 B por escuta em revisões que repetiam o recibo. Parar de duplicar a revisão automática do caminho comum, mantendo a decisão no recibo, a revisão humana e a trilha de fusões. Cobrir também rejeição, repetição confiável e correção administrativa. Tratar a unificação de horários e a gravação de valores padrão como cortes separados até confirmar suas semânticas nos estados excepcionais.
3. **Simplificar a configuração da Outbox pela ISSUE-205.** Unir `profile_relays` e `relay_destinations` quando o modelo 1:1 for preservado. Os cinco gatilhos de sincronização são candidatos a sair. Manter itens de entrega em `outbox`, autorização de rede em `relay_allowlist`, geração que invalida reservas antigas, limites, pausa, confirmação de lote e recuperação. `relay_sent_archive` é escrito na exclusão de envios e limpo na retenção, mas a página de entregas exclui `sent` e não lê esse total. Tratar a retenção dessa auditoria como decisão separada da fusão.
4. **Medir índices com as consultas finais pela ISSUE-207.** `idx_ingest_inbox_example` é exigido explicitamente pela busca de exemplos. `idx_ingest_inbox_edges` é exigido para as partes de dias nas bordas do período. `idx_ingest_health_decisions` atende a ordenação das últimas decisões. `idx_ingest_source_receipt` participa da manutenção do último recibo por fonte, inclusive quando o atual é excluído. Medir cada candidato com `EXPLAIN QUERY PLAN` antes e depois, incluindo escrita, filtros, exclusão do último recibo e distribuição por perfil. A ausência de referência nominal de um índice em Go não prova que o SQLite não o usa. Reavaliar `idx_ingest_health_received` com os planos finais.
5. **Validar retenção pela ISSUE-208 e consolidar pela ISSUE-195.** O `original_snapshot` é consumido também pelo resolvedor, pela admissão/repetição e pela exportação com procedência da ISSUE-174. O digest participa do tratamento de reenvio. Compactar o snapshot depende de definir quais evidências continuam disponíveis e testar as rotas que usam o recibo. Antes da 195, medir o caminho da API em banco descartável e verificar a migração com contagens e digest por perfil, incluindo histórico e relay.

A Central é a camada de leitura e ação sobre esses serviços. Inbox, resolução de escutas, entrega e enriquecimento têm ciclos de vida próprios. Para a ISSUE-213, mostrar o estado do enriquecimento na Central usando a fila já existente, sem converter falha de metadados em pendência de ingestão.

## Da API à escuta no SQLite

1. A API compatível com ListenBrainz autentica a chave e determina o perfil. O handler limita o corpo e o lote, valida a operação e monta o snapshot com os campos que o adaptador reconhece. `playing_now` segue o caminho de exibição temporária. Escutas históricas passam por `AdmitOccurrence` em `internal/db/sqlite/ingest_receipts.go`.
2. A admissão grava o recibo em `ingest_events`, incluindo perfil, origem, chave de API, snapshot e digest. Reenvios podem reutilizar o recibo ou deixar evidência de conflito, conforme a identidade e a chave de idempotência. A API responde com sucesso após a admissão durável. Esse sucesso confirma a recepção do recibo. A resolução acontece depois.
3. O trabalhador de ingestão reserva um recibo com prazo e versão. O resolvedor confronta os créditos, a gravação, o lançamento, a posição e a evidência de identidade disponível. As consultas remotas de verificação acontecem fora da transação de escrita. A conclusão revalida a reserva e examina candidatos à duplicação no perfil. Identidade contraditória ou ambígua pode ficar pendente, e uma colisão de tempo pode exigir revisão conforme a configuração do perfil. Erros transitórios têm retentativas limitadas e podem deixar o recibo estacionado.
4. Se for uma escuta distinta, a transação resolve ou cria o vínculo de catálogo, insere `listens` com `event_id`, ocorrência, horário, perfil e origem, e registra o recibo como aceito. Um duplicado aponta para a escuta já existente. A mesma transação grava a decisão e, quando elegível, a intenção de relay. A Inbox consulta recibos e seus resumos, e a página de procedência liga a escuta ao recibo aceito pelo evento e pelo perfil.

## Da Outbox à entrega

1. Na admissão, um recibo elegível captura a geração do destino aprovado do perfil. Na aceitação, `insertRelayIntent` confirma que a geração continua válida e insere o trabalho em `outbox` junto da escuta. Uma entrega que excede a capacidade fica registrada com esse motivo. O trabalhador de relay opera separadamente da API e do resolvedor.
2. `ClaimRelayBatch` reserva um destino e itens devidos com prazo, versão, perfil e geração. `RelayTarget` recupera o destino, credencial e permissões. O trabalhador monta payloads versionados, invalida itens cuja fonte deixou de ser válida e revalida a reserva imediatamente antes do POST a `/submit-listens`. Envia `single` para um item e `import` para um lote.
3. O resultado é persistido depois de cada POST. Sucesso conclui os itens. Respostas de lote inválido podem levar à divisão do lote para isolar o item rejeitado. Limite do destino, falhas transitórias e problemas de autenticação ou configuração seguem as políticas de espera ou pausa. Troca de geração impede que uma reserva antiga atualize a fila atual. Uma queda entre o POST remoto aceito e a gravação local pode exigir repetir a entrega, por isso a fila durável não garante entrega única.
4. A Outbox na Central mostra trabalho aberto, pausa e resultados recentes de entregas ainda fora de `sent`. O relay e a Central usam os mesmos dados operacionais por perfil. A data de tentativa, a reserva e a geração são informações do envio. A decisão de ingestão continua ligada ao recibo e à escuta.

## Verificações

Os testes de contagem e migração da Inbox, planos de atualização da Outbox e isolamento de reservas pela geração do destino passaram. A versão anterior do `TestIngestHealthQueryPlans` parava no schema 15 porque a consulta atual da fila precisava de `profile_ingest_review`, criada na migração 021. O ensaio agora roda no schema corrente e passou com 300 mil recibos sintéticos, 250 mil no perfil medido e 250 pendências de revisão nesse perfil.

| Consulta com dados sintéticos | Mediana após aquecimento |
|---|---:|
| Inbox de 1 / 7 / 30 dias | 8,25 / 8,35 / 6,43 ms |
| Última submissão por fonte | 0,10 ms |
| Últimas decisões | 0,11 ms |
| Contagem, página e candidatos de revisão | 3,14 ms |

O plano usou o resumo diário e `idx_ingest_inbox_edges` nas bordas do período, `idx_ingest_source_summary_recent` em fontes, `idx_ingest_health_decisions` nas decisões e `idx_ingest_health_review` na fila. O ensaio mede consultas SQLite sem renderização da página, com poucas fontes e sem volume de relay representativo. Serve de referência para a ISSUE-207, que ainda precisa comparar candidatos a remoção de índice com e sem cada índice.

O `make check` passou com 13 de 13 percursos de ponta a ponta. Em 24/11/2026 o usuário concordou com a avaliação e pediu que os dados e a sequência fossem registrados no estudo.

## Sequência de execução

1. Registrar o recorte de leitura da Central e da Inbox no backlog. Aplicar o filtro de chave antes de buscar exemplos e carregar as seções conforme a navegação. Validar contagens, opções do seletor, links diretos, ações e permissões. Coordenar com a ISSUE-192, cujo escopo atual é filtro por origem.
2. ISSUE-203. Retirar tabelas sem escritor e constantes legadas do recibo. Incluir o gatilho de enriquecimento no inventário de dependências de `mode`.
3. ISSUE-204. Começar pela revisão automática que repete o recibo aceito. Verificar separadamente horários e valores padrão nos caminhos de rejeição, repetição e revisão.
4. ISSUE-205. Reunir configuração e estado do destino do relay. Testar reservas em andamento, mudanças de configuração, pausas, limites e reenvios.
5. ISSUE-206. Executar as simplificações pendentes de revisão por perfil, origens e ordem de exportação antes da consolidação.
6. ISSUE-207. Usar o ensaio de 300 mil recibos para comparar planos e custos com e sem cada índice candidato após as mudanças de consulta. A ISSUE-208 pode avançar em paralelo como estudo da retenção e de seus efeitos sobre a procedência.
7. ISSUE-195. Consolidar após as mudanças de esquema previstas, incluindo as dependências já registradas nas ISSUE-055 e 193. Validar o salto do Koito v2 com backup, integridade, digest, contagens por perfil e relay configurado.
