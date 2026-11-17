---
id: doc-040
title: Contrato de ingestão, retenção e deduplicação (ISSUE-079)
type: specification
created_date: '2026-11-17T15:28:00Z'
updated_date: '2026-11-17T15:32:00Z'
tags:
  - design
---
# Contrato de ingestão, retenção e deduplicação

ISSUE-079 · Advisor · 17/11/2026. Complementa o contrato de identidade aceito em doc-039/decision-008. As cinco escolhas de política foram decididas pelo usuário em comentário na ISSUE-079; a formalização permanece em revisão, com decision-009 em **proposed**. Este documento não implementa o pipeline nem autoriza publicação.

## 1. Escopo e força normativa

**Contrato:** as regras desta seção e das seções 2–9 são requisitos para implementação e aceite das tarefas seguintes. “Deve” indica obrigação; “não deve” indica proibição. Os valores de retenção, diagnóstico, exclusão e separação da ISSUE-055 refletem as decisões do usuário.

**Sugestão de implementação:** a seção 10 apresenta mecanismos, assinaturas, limites auxiliares e sequência sugeridos. Não transforma detalhes ainda não testados em escolhas do usuário. A seção 11 define verificações futuras; não afirma que testes de produção já foram executados nesta spike.

A ingestão deve preservar recebimentos antes de confirmar seu recebimento, resolver identidade e duplicidade separadamente, e permitir revisão autenticada por perfil. Nenhuma similaridade autoriza descarte silencioso. Doc-039 e decision-008 permanecem a autoridade sobre gravação, ocorrência e evento. Este contrato define a retenção que aqueles documentos deixaram em aberto.

## 2. Glossário e invariantes

| Termo | Significado |
| --- | --- |
| Gravação | Identidade musical interna estável; recording MBID validado quando conhecido. Não representa composição ou edição. |
| Lançamento | Edição ou release, distinta de release-group. |
| Ocorrência | Faixa no contexto de um lançamento, disco/posição/créditos; preserva os atuais track_id e URLs. Pode ser provisória sem lançamento. |
| Evento de escuta | Escuta aceita com event_id próprio e perfil, independente de gravação ou segundo; referencia a ocorrência ouvida. |
| Recebimento/recibo | Entrega lógica de uma fonte, com receipt_id e snapshot observado; ledger ingest_events. Vários recebimentos podem apontar para um evento. |
| Observação conflitante | Variante imutável recebida com a mesma chave confiável, mas conteúdo incompatível. Não sobrescreve o recibo original. |
| Snapshot | Representação estruturada, saneada e versionada dos dados necessários à resolução e ao reprocessamento suportado. Não é cópia irrestrita do corpo. |
| Revisão | Decisão por perfil, com evidência, autor e versão; não altera implicitamente o catálogo global. |
| Outbox | Intenção durável de entrega externa, persistida junto da decisão elegível. |
| Detalhes encerrados | Tentativas, erros saneados, candidatos intermediários e trilhas detalhadas que já não são necessárias à decisão vigente ou ao trabalho aberto. |

Uma pendência não deve criar escuta contável. Cada evento aceito conta uma vez por ocorrência e por gravação, e por lançamento incluindo o balde “sem lançamento”. MBID de gravação não identifica ocorrência nem evento. Duas repetições legítimas no mesmo segundo devem continuar possíveis. Migração não deve fundir por texto nem descartar eventos.

## 3. Envelope do recebimento

O formato abaixo é o contrato lógico; nomes físicos de colunas e codificação dos IDs podem ser definidos na implementação, preservando os tipos e invariantes.

| Campo | Tipo lógico e origem | Regra |
| --- | --- | --- |
| schema_version | Inteiro positivo do servidor | Versão do envelope/snapshot. |
| receipt_id | ID opaco estável do servidor | Independente de event_id e de identidade musical. |
| batch_id, item_ordinal | ID do lote e inteiro não negativo | Localizam o item; ordinal de arquivo é absoluto, antes de filtros. |
| profile_id | Inteiro do contexto autenticado | Nunca confiar no perfil declarado no corpo. |
| source_type, source_instance_id | Enum do adaptador e ID estável por fonte/perfil | Trocar token ou renomear upload não muda a instância. |
| adapter_version | Texto versionado | Permite explicar a interpretação do formato. |
| received_at | Instante UTC do servidor | Distinto do horário ouvido. |
| external_event_id, external_namespace | Textos opcionais da fonte | Usáveis para idempotência apenas quando confiáveis e com namespace documentado. |
| client_idempotency_key | Texto opcional do cliente | Escopo perfil/instância/operação; não obrigatório para clientes ListenBrainz. |
| file_digest, file_format, format_version | SHA-256 dos bytes exatos e identificação do formato | Arquivo pode ser removido após persistência; nome/caminho não são identidade. |
| listened_at, original_time | Instante UTC e valor original saneado | Preservar precisão recebida; não fabricar milissegundos nem substituir por received_at. |
| time_precision, time_semantics | Enum de precisão e start/end/unknown | Registrar estimativa, offset original quando enviado e uncertainty_ms quando conhecido. |
| client_name, client_version | Textos opcionais do cliente | Informativos, limitados, não autoridade de identidade. |
| original_snapshot | Objeto tipado imutável | Dados observados e respectivas origens, sem segredos. |
| normalized_snapshot, normalizer_version | Objeto e versão do resolvedor | Derivados; nunca sobrescrever o snapshot observado. |
| snapshot_digest, digest_version | Digest da representação canônica saneada | Separado do digest do arquivo; igualdade não prova identidade de evento sem recibo confiável. |
| state, identity_status | Enums do processamento | Separar classificação musical de falha operacional. |
| duplicate_decision | Igual/Diferente/Revisar ou ausente | Resultado independente da resolução musical. |
| reason_codes, evidence, rule_version | Razões estáveis, objeto de evidência e versão | Obrigatórios ao decidir; escopo do perfil. |
| resolution_version, resolved_at | Inteiro monotônico e instante opcional | Controle de concorrência e histórico da decisão. |
| event_id, occurrence_id, recording_id | Referências opcionais resolvidas | Accepted/duplicate devem apontar ao evento escolhido; pendência pode não ter referências. |

O snapshot deve conservar título, lista de artistas/créditos, lançamento recebido ou ausência, IDs declarados com namespace/fonte/confiança, track MBID, disco/posição quando informados, duração com unidade, duração efetiva observada separada e cliente. Tags e datas devem ser conservadas quando efetivamente recebidas e reconhecidas, com tipo, precisão parcial e procedência. Tags livres não devem ser convertidas indiscriminadamente em gênero. Data de edição não deve substituir ano original. Additional_info declarado deve ficar separado de enriquecimento mbid_mapping.

UUID válido sintaticamente não comprova namespace ou correção. Conflitos entre IDs não devem substituir identidade consolidada silenciosamente. Ausência de álbum não deve inventar álbum pelo título da faixa nem selecionar outro álbum da gravação. Reprocessamento deve funcionar sem staging temporário e preservar receipt_id; não promete reconstruir campos desconhecidos ou descartados do corpo original.

### Estados, recebimento e ACK

Fluxo lógico: received → accepted, pending_identity, pending_duplicate, duplicate ou invalid. Falhas operacionais mantêm trabalho retryable com tentativa, motivo e próximo agendamento, sem reclassificar uma ambiguidade como erro de transporte. Invalid classifica conteúdo já recebido e preservado com razão; não deve ser usado para confirmar lote estruturalmente inválido.

Formato, autorização e limites devem ser validados antes de persistir o lote. Erros correspondentes usam 400/401/413. Todos os recibos e observações do lote válido devem ser persistidos atomicamente antes de HTTP 200 no formato compatível; 200 significa recebido duravelmente, inclusive pendências. Falha temporária de persistência retorna 503 sem sucesso declarado. Resolução posterior não deve provocar novo ACK nem exigir que o cliente reenvie após um 200.

Playing_now permanece efêmero, sem ledger histórico, evento, estatística ou outbox histórica. Importação deve persistir o item antes de associar, continuar após conflito musical e retomar com cursor seguro. Processed = Imported + Duplicates + Pending + Ignored + Errors; itens pendentes não devem entrar em outra categoria. Falha operacional pode interromper com progresso parcial durável.

## 4. Idempotência e recebimentos conflitantes

| Canal | Chave lógica confiável | Efeito |
| --- | --- | --- |
| Fonte com ID estável | perfil + instância estável + namespace + external_event_id | Reenvio equivalente reutiliza recibo/resultado enquanto existir. |
| API com chave opcional | perfil + instância + operação + chave; ordinal por item no lote | Digest canônico do lote detecta reutilização incompatível da chave. |
| Arquivo | perfil + instância estável de importação + formato/versão + digest dos bytes exatos + ordinal absoluto | Mesmo conteúdo com outro nome reutiliza os itens; repetições em ordinais diferentes são preservadas. |
| API sem ID ou chave confiável | Nenhuma garantia automática de exatamente uma vez | Nova entrega indistinguível gera revisão de duplicidade. |

Constraints no banco devem sustentar as chaves, com tratamento explícito de valores ausentes. Rotação de token não reinicia o namespace. IDs externos reciclados exigem namespace/época documentados pelo adaptador; não presumir estabilidade não demonstrada.

Mesma chave e snapshot equivalente reutilizam o recibo. Mesma chave com conteúdo incompatível deve gerar observação imutável durável antes do ACK, mantendo snapshot anterior e abrindo revisão por perfil. Repetições da mesma variante podem ser compactadas por digest/contador sem perder variantes distintas. Alteração de tag não autoriza criar outra escuta nem sobrescrever identidade. O resolvedor deve explicar a equivalência escolhida.

Arquivo reordenado, parcialmente sobreposto ou convertido é novo recebimento e passa pela comparação; hash do item não deve apagar repetições legítimas. Não usar track_id + segundo ou recording MBID + segundo como chave de evento.

### Exclusão e reimportação: decisão do usuário

**Não conservar tombstones nem chaves de reimportação após excluir uma escuta. A exclusão nunca deve bloquear uma importação futura.** A idempotência por recibo vale somente enquanto o recibo/evento correspondente existir. Depois da exclusão, a mesma escuta recebida novamente deve ser tratada como nova entrada e, quando resolvida/aceita, receber **novo event_id**, sem ressuscitar o evento excluído.

A exclusão deve remover os recibos associados exclusivamente ao evento, snapshots, variantes, vínculos de revisão e chaves que poderiam bloquear sua recriação, inclusive em índices auxiliares, caches e resultados de comparação. Observações ou pendências autônomas que ainda existam não devem carregar uma marca oculta que proíba a futura importação. Ao excluir/descartar um recebimento pendente explicitamente, sua chave não deve continuar como bloqueio. Estatísticas operacionais agregadas podem permanecer sem identidade ou conteúdo que permita bloquear a escuta.

Outbox não enviada do evento excluído deve ser cancelada e removida com suas chaves locais vinculadas, sem virar tombstone. Chave de entrega de evento novo deve derivar da nova identidade do evento, sem reaproveitar a chave antiga. Exclusão local não desfaz efeito já entregue remotamente e não garante que o destino aceite uma repetição idêntica segundo as regras dele. Uma entrega em voo pode já ter produzido efeito externo: o worker deve detectar ausência do evento ao concluir, sem recriar registros excluídos.

Uma chave de lote/arquivo não deve funcionar como bloqueio global quando apenas alguns itens foram excluídos: reimportar deve reutilizar os recibos dos itens que ainda existem e criar novos recibos/eventos para os excluídos. Resultado de lote em cache não pode responder sucesso antigo sem reavaliar esses itens. Dedup ainda considera outros eventos vivos: se houver outra escuta remanescente indistinguível, aplica-se revisão conservadora; o evento excluído nunca participa como candidato ou impedimento. Reenvio automático da fonte também pode recriar uma escuta excluída; isso é consequência intencional da decisão, sem filtro oculto para impedir.

## 5. Deduplicação: Igual, Diferente e Revisar

| Evidência | Decisão e ação |
| --- | --- |
| Mesmo recibo confiável e conteúdo equivalente | Igual: reutilizar resultado, sem novo evento ou intenção de entrega. Sem janela de expiração enquanto o recibo existir. |
| Mesma chave e conteúdo incompatível | Revisar: preservar observação e conflito; não sobrescrever nem contar nova escuta automaticamente. |
| IDs confiáveis distintos na mesma fonte | Diferente, inclusive no mesmo segundo, salvo relação explícita de correção/duplicata declarada pela fonte. |
| Mesmo arquivo e ordinal | Igual enquanto recibo existir; após exclusão vale a regra de nova entrada. |
| Ordinais diferentes | Preservar a possibilidade de repetição legítima; não colapsar por igualdade do item. |
| Fontes diferentes coincidentes, ou entrega sem ID indistinguível | Revisar/pending_duplicate; nenhuma segunda contagem antes da resolução. |
| Evidência confiável compartilhada de um evento ou revisão explícita | Igual: ligar recebimentos ao evento existente sem perder proveniência. |
| Identidade de eventos comprovadamente distinta | Diferente, respeitando resolução de ocorrência e gravação. |
| Conflito musical/alias ambíguo/MBID incompatível no próprio item | pending_identity; não escolher candidato silenciosamente. |

Sem ID confiável, os mesmos perfil, instante e metadados não provam exatamente uma vez. O primeiro recebimento coerente pode criar evento; o concorrente indistinguível deve encontrar o estado atualizado e ficar pendente. Ocorrências de lançamentos diferentes preservam seu contexto; versão ao vivo ou remaster não se funde por texto/duração. Igual não deve apagar o snapshot de origem. Diferente não deve ignorar uma contradição interna não resolvida.

### Candidatos e tolerâncias

É normativo que **janelas só selecionem candidatos**, nunca comprovem Igual nem autorizem descarte. A política por adaptador deve ser versionada e declarar precisão e semântica do tempo. Perfil e listened_at delimitam a busca; received_at não limita a reconciliação de importações antigas.

Os valores iniciais abaixo são **sugestões de implementação a validar com fixtures**, não novos prazos decididos pelo usuário:

| Tempo da fonte | Janela inicial sugerida |
| --- | --- |
| Início exato | ±2 segundos |
| Arredondado ao minuto | ±60 segundos |
| Estimado com incerteza conhecida | Soma das incertezas +2 segundos, teto de 120 segundos |
| Final da reprodução | Converter para início apenas com duração observada confiável; registrar transformação |
| Sem semântica/incerteza conhecida | Não assumir início exato ou ampliar janela arbitrariamente; revisar quando a ambiguidade impedir decisão |

Duração é evidência auxiliar: tolerância sugerida min(5 s, max(2 s, 2% da duração)); ultrapassá-la enfraquece correspondência, sem comprovar diferença sozinha. Texto normalizado serve à candidatura; Unicode/caixa e agrupamento de remasters permanecem nas ISSUE-136/137/138. Fora da janela, somente aceitar novo evento se identidade estiver resolvida e não houver outra evidência conflitante.

A explicação persistida deve conter reason_code estável, rule_version, atributos comparados, deltas temporais, precisão/transformação usada, candidatos, decisão, autor automático/humano e resolution_version. Deve permitir responder “por que esta escuta está aqui?” após expirar detalhes: conservar a evidência mínima da decisão vigente. Pontuação opcional não deve ser apresentada como probabilidade sem calibração nem substituir a regra.

## 6. Revisão por perfil e reprocessamento

Revisão deve consultar somente recibos/candidatos daquele perfil, validar perfil e versão esperada, conferir coerência da ocorrência e executar em transação curta: criar/ligar evento → atualizar decisão/versionamento → persistir outbox elegível → commit. Dois revisores concorrentes não podem aplicar duas decisões; o perdedor recebe conflito e recarrega. Reprocessar mantém receipt_id e não cria evento/envio adicional por mera tentativa.

Mudanças de associação devem preservar o snapshot anterior e a explicação vigente enquanto o evento existir. Revisão de perfil não deve alterar catálogo global implicitamente; correção administrativa global requer ação própria. Pendências devem ser consultáveis e resolvíveis antes da ativação geral do pipeline, coordenando ISSUE-083. Nenhum item pendente deve disparar relay indiscriminado do lote original.

## 7. Retenção por classe

| Classe | Prazo/política final | Configuração e exclusão |
| --- | --- | --- |
| Escutas aceitas e identidade musical necessária | Até exclusão explícita | Sem TTL; purga operacional não apaga histórico nem catálogo necessário. |
| Pendências e conflitos abertos | Até resolução/descarte explícito | Sem TTL automático; alertas e controle de capacidade. |
| Núcleo do recibo, chave confiável, vínculo e decisão vigente | Enquanto recibo/evento existir | Não expira pelo prazo de detalhes. Exclusão da escuta remove o que poderia impedir reimportação. |
| Snapshot estruturado vigente e proveniência mínima | Vida do evento; pendência até resolução/descarte | Suficiente ao reprocessamento suportado, limitado e saneado. Exclusão elimina os dados associados. |
| Tentativas encerradas, erros saneados, candidatos intermediários, revisões antigas detalhadas e conflitos encerrados | **90 dias após encerramento** | Configurável de **7 a 3650 dias**; nunca apagar evidência ainda necessária à decisão vigente. |
| Outbox pendente/retryable/falha não descartada | Até sucesso ou descarte explícito | Sem TTL que perca trabalho; exclusão do evento cancela e remove suas intenções locais. |
| Detalhes de outbox enviada/descartada | 90 dias após estado terminal | Mesma configuração de detalhes; resultado/chave mínimos só enquanto o evento existir. Exclusão prevalece sobre o prazo. |
| Corpo diagnóstico saneado opcional | **Desativado por padrão; 7 dias quando habilitado** | Opt-in por perfil e fonte, quota de **100 MiB por perfil**, máximo de **1 MiB por item**. Pode expirar mesmo em pendência, pois snapshot é obrigatório. |
| Integração e credenciais | Até revogar/excluir configuração | Segredo separado por token_ref; retirada da integração não apaga proveniência das escutas vivas. |
| Playing_now | Efêmero | Sem retenção histórica. |

Não existe classe de tombstones. Prazo de detalhes não deve apagar núcleo/snapshot/evidência vigente por engano; exclusão explícita de escuta, por sua vez, não deve ser neutralizada pela retenção. Não guardar chave em outra tabela para contornar a escolha do usuário.

### Purga e capacidade

A purga deve ser retomável, executar em lotes transacionais curtos e revalidar estado/versão e dependências antes de apagar detalhes elegíveis. Reabertura suspende elegibilidade; mera tentativa/releitura não reinicia artificialmente o prazo. Trabalho aberto ou com lease ativo não deve perder dados necessários. Métricas de purga devem ser agregadas e saneadas. Frequência diária e lote inicial de 500 IDs são sugestões operacionais.

DELETE normalmente libera páginas para reutilização sem reduzir imediatamente o arquivo. Checkpoints WAL devem ser monitorados; leitores longos podem atrasá-los. VACUUM é manutenção fora da transação de purga, com espaço livre e backup verificados; não deve rodar a cada lote. A política deve distinguir exclusão lógica de remoção de vestígios em WAL, backups e armazenamento. Expiração local não apaga automaticamente cópias antigas. Ver [SQLite VACUUM](https://sqlite.org/lang_vacuum.html) e [WAL](https://sqlite.org/wal.html).

Snapshots e pendências crescem com histórico e fontes; retenção sem TTL não significa capacidade ilimitada. Medir bytes/recibo, filas, idade e espaço livre; impor limites de entrada e backpressure antes do ACK quando sem capacidade, sem apagar pendências para liberar espaço. Como estimativa, 1 milhão de recibos de 2 KiB ocupa cerca de 2 GB antes de índices/WAL/backups; 100 mil detalhes/dia de 1 KiB por 90 dias ocupa cerca de 9 GB. São exemplos de planejamento, não medições.

## 8. Payload e diagnóstico

Corpo bruto completo não deve ser armazenado por padrão. A captura opcional é **saneada**, não uma cópia byte a byte: retirar tokens, Authorization, cookies, credenciais em URLs, IP/localização/dispositivo desnecessários e extensões pessoais fora da lista permitida. Não copiar segredos para digest, logs, outbox ou explicações. Hash não torna dados musicais anônimos.

Snapshot obrigatório e diagnóstico opcional são independentes. Falta de quota do diagnóstico não deve impedir recibo se o snapshot puder ser persistido: registrar ausência da captura, sem ultrapassar quota nem truncar silenciosamente identidade. A configuração vale por perfil/fonte; a soma não deve ultrapassar 100 MiB por perfil. O corpo acima de 1 MiB fica sem captura diagnóstica; limites próprios de ingestão continuam sujeitos à validação antes do ACK. A captura não deve ser necessária à retomada/revisão nem transformar tags pessoais em catálogo compartilhado. Não promete recuperar campos desconhecidos após sete dias ou reexecutar bytes originais.

## 9. Concorrência, durabilidade e outbox

SQLite WAL permite leituras concorrentes com escrita, mas mantém **um escritor por vez**. O projeto deve usar transações curtas e constraints no banco; mutex local não sustenta idempotência entre processos. Decisão e criação/associação do evento devem reconsultar candidatos sob a serialização de escrita, não confiar apenas em comparação feita antes da transação. Ao conflito/retry, revalidar perfil, versão e estado atual, inclusive exclusão concorrente. Fonte externa/enriquecimento/HTTP não devem ocorrer dentro do lock de escrita. [SQLite WAL](https://sqlite.org/wal.html).

Persistência do lote antes do ACK, resolução versionada e intenção de outbox devem sobreviver a reinício. A configuração de sincronização deve sustentar a promessa de durabilidade, incluindo falha de energia conforme garantias do SQLite/armazenamento; WAL sozinho não basta. Erro de disco cheio nunca permite confirmar sucesso sem commit.

A outbox deve ter unicidade lógica por perfil + instância de destino + event_id + tipo de operação. Retry reutiliza chave; reprocessamento não cria novo relay. Worker deve reclamar trabalho por lease/versionamento em transação curta, enviar fora dela e concluir por comparação de versão/posse. Crash depois da aceitação remota e antes da confirmação local permite reentrega: garantia de entrega é pelo menos uma vez; exatamente uma vez remoto depende de suporte real do destino à idempotência. Respeitar Retry-After, backoff/jitter, limites e shutdown. Revalidar elegibilidade antes de enviar; efeito já em voo não pode ser desfeito por promessa local.

Exclusão deve competir com resolução/outbox na mesma disciplina transacional. Nenhum worker pode recriar evento, recibo ou chave excluídos ao finalizar resposta atrasada. Tentativas futuras são canceladas; reimportação explícita ou automática posterior segue o fluxo de nova entrada, sem bloqueio pelo histórico excluído.

## 10. Plano de implementação e migrações

Esta seção é um plano técnico; DDL, assinaturas finais e tuning dependem da implementação. A separação da 055 da v4 é decisão normativa.

### ISSUE-080: v4 somente de infraestrutura

Criar integrations com identidade estável por perfil; ingest_events como ledger único com snapshots/detalhes separáveis, observações conflitantes e revisão versionada; outbox, índices de fila/purga e constraints de isolamento/idempotência. **Não incluir tabelas finais de tags/datas da ISSUE-055 e não criar tombstones.**

Enquanto a 133 não introduzir evento estável, a v4 não deve apontar FK a coluna inexistente nem tratar track/segundo como event_id. Sugestão: criar estruturas inativas, referências futuras opcionais e impedir enfileiramento/resolução funcional até a 133 completar vínculo e constraints. A aplicação atual deve continuar utilizável entre etapas; migração da infraestrutura não substitui silenciosamente o relay existente.

Atualizar schemaVersion/checkDatabase/bootstrap e validadores de layout/triggers, fluxos tunetrail.db e koito.db, backup e restauração. Não editar retroativamente SQL 001–003. Rejeitar versões desconhecidas e schema incompleto.

### ISSUE-133: identidade posterior à v4

Coordenar a próxima versão disponível (v5 é sugestão, não reserva irrevogável). Tracks atuais viram ocorrências com IDs/URLs preservados; backfill de eventos 1:1 preserva perfis, timestamps e clientes, sem fusão ou descarte. Permitir ocorrência sem lançamento e várias ocorrências por gravação/lançamento. Introduzir event_id próprio e relacionar ledger/outbox antes de ativar escritores novos.

A mudança é aditiva no sentido de preservar dados; remover a chave composta de listens pode exigir reconstrução controlada da tabela. Definir ponte de compatibilidade ou corte coordenado com leitores essenciais da 135. Nunca ativar escritores novos com leitores assumindo gravação única por álbum ou evento único por segundo. Exclusão por event_id e cursor estável devem acompanhar a ponte.

### ISSUE-134: ingestão comum e reimportação

Adaptar ListenBrainz, Last.fm e todos os importadores ao envelope e resolvedor comuns (junto da 081). Manter ocorrência ouvida, digest/ordinal, precisão temporal, Pending e retomada independente do staging. Testar idempotência enquanto recibos existem e nova importação após exclusão, inclusive arquivos/lotes parcialmente excluídos. Chave de lote não deve bloquear os itens excluídos. Coordenar revisão por perfil com 083 e outbox elegível com 082, sem ativação geral antes de ser possível resolver pendências.

### ISSUE-055: migração posterior

Investigar fixtures efetivas dos clientes e fallback MusicBrainz; validar nomes, tipos e unidades antes de definir schema de metadados. Distinguir tag pessoal por perfil, gênero confiável e metadado editorial compartilhado, preservar edição manual e datas parciais (YYYY, YYYY-MM, YYYY-MM-DD), sem fabricar dia/mês. Coordenar as entidades com a 133. A v4 conserva apenas campos conhecidos efetivamente recebidos em snapshot; a projeção futura não deve criar outra escuta/outbox. Não prometer recuperar campos históricos que nunca foram recebidos/conservados.

### Backup, rollback e ordem de habilitação

Executar expandir → validar → ativar. Backup consistente antes de migrar via SQLite Backup API ou VACUUM INTO; não copiar apenas .db de uma base ativa ignorando WAL. Cópia de banco real permanece local/privada fora de Git, com resultados agregados sem dados pessoais. [SQLite Backup API](https://sqlite.org/backup.html).

Rollback transacional é exigido para falha antes do commit. Após novas escritas, não prometer Down sem perda: preferir correção adiante ou restauração com escritores parados e impacto explícito sobre escritas posteriores. Binário antigo deve recusar schema novo. Restaurar backup antigo pode trazer escutas excluídas e intenções já enviadas: exigir reconciliação antes de reativar workers, sem introduzir tombstones para contornar a decisão de exclusão.

Ordem sugerida: contrato 079 → infraestrutura 080 → identidade/ponte 133 → pipeline/adaptadores 081/134 com revisão/outbox 082/083 → ativação gradual → purga após testes → projeção 055 em migração posterior. Investigação das fixtures da 055 pode ocorrer em paralelo.

Alvos previstos: db/migrations_sqlite (SQL novos); internal/db/sqlite/{bootstrap,sqlite,listen,track,exports}.go; internal/db/{db,opts}.go; internal/models/track.go; internal/catalog/{catalog,associate_track}.go; internal/listenbrainz/types.go; engine/handlers/lbz_submit_listen.go; internal/importer/*.go. Novos módulos previstos: internal/ingest, internal/dedup e internal/outbox, persistência e testes correspondentes. Não são arquivos alterados nesta spike.

Sugestões: executor local de escrita limitado, BEGIN IMMEDIATE em conexão dedicada, timeout e retries limitados com contexto; reiniciar toda a transação em conflito de snapshot, não somente o último INSERT. Avaliar synchronous=FULL para o writer de recibos. Limites iniciais de snapshot de 32 KiB/item, strings de 4 KiB e 100 tags de 256 bytes são propostas a validar com fixtures; nunca truncar identidade silenciosamente.

Assinaturas conceituais, sem compromisso de API Go final:

```text
ReceiveBatch(ctx, BatchEnvelope) -> Receipts
ResolveReceipt(ctx, profileID, receiptID) -> Resolution
Compare(ctx, PolicyVersion, Receipt, Candidate) -> DecisionWithEvidence
ReviewReceipt(ctx, profileID, receiptID, expectedVersion, ReviewDecision) -> Resolution
PurgeClosedDetails(ctx, cutoff, batchLimit) -> PurgeResult
ClaimOutbox(ctx, owner, leaseUntil) -> Deliveries
```

## 11. Testes de aceite e evidências exigidas

Estes são critérios para a implementação nas tarefas seguintes. A ISSUE-079 entrega contrato e plano, não código/testes de runtime.

| Grupo | Cenários e resultado exigido |
| --- | --- |
| Envelope/isolamento | Perfil forjado não muda escopo; tokens não aparecem em snapshot/log; campos ausentes e precisão parcial permanecem distintos; limites inválidos rejeitam antes de ACK. |
| ACK durável | Crash antes/depois do commit, resposta perdida, disco cheio e lote estruturalmente inválido; nenhum 200 sem lote válido durável. Reinício recupera trabalho sem goroutine anterior. |
| Identidade | Mesma gravação em A/B, posições distintas de “Intro”, álbum ausente, aliases ambíguos e MBIDs conflitantes; preservar ocorrência e não contar pendências. |
| Idempotência viva | Mesmo ID/snapshot, chave com variante conflitante, token rotacionado, arquivo renomeado, ordinais repetidos legítimos, arquivo reordenado/sobreposto; não duplicar recibo confiável nem perder variantes. |
| Deduplicação | Repetições no mesmo segundo, fontes cruzadas, sem ID, horário start/end, ±limites de janela, fuso/DST e clock skew; janelas não concluem Igual. |
| Revisão/corridas | Dois revisores, duas fontes, exclusão de perfil e evento durante resolução; só uma versão vence e ninguém acessa outro perfil. |
| Exclusão/reimportação | Excluir escuta com vários recibos/variantes/outbox, reenviar mesmo ID/chave/arquivo: evento novo; itens vivos do mesmo lote reutilizados; nenhum cache/índice bloqueia excluídos. Candidato vivo independente ainda pode gerar Revisar. |
| Outbox | Crash após remoto aceitar, lease vencida, Retry-After, falha terminal, exclusão durante envio; reentrega explicada, sem recriação local por resposta tardia. |
| Retenção | Padrão 90, limites 7/3650, configuração fora da faixa, cutoff exato, reabertura concorrente, detalhes versus mínimo vigente; histórico/pendências preservados e exclusão explícita efetiva. |
| Diagnóstico | Desativado por padrão, opt-in por perfil/fonte, sete dias, 1 MiB/item, soma de 100 MiB/perfil, saneamento; falta de quota não invalida snapshot recebido. |
| Migração real copiada | Banco novo, v3 multi-perfil real e legado koito; integrity_check/foreign_key_check, contagens/digests por perfil, IDs/URLs, aliases/MBIDs nulos; falhas por fase, reinício e restore comprovado. Sem dados pessoais nas evidências. |
| Compatibilidade/capacidade | Ponte entre versões, binário antigo/versão desconhecida, leitores sem lançamento, dois eventos/segundo, tamanho/latência/WAL/disco e purga concorrente; nenhuma escuta descartada. |

## 12. Registro da spike

As políticas vieram do comentário do usuário na ISSUE-079 em 17/11/2026. A recomendação anterior do Advisor de guardar tombstones foi **rejeitada** e substituída pela seção 4; não deve ser retomada por leitura de memória antiga. Decision-009 registra a formalização como proposed, aguardando revisão do usuário. Este documento não marca implementações futuras como verificadas nem altera as ISSUE-080/133/134.
