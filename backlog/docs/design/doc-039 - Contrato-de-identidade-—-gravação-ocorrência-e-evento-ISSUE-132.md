---
id: doc-039
title: Contrato de identidade — gravação, ocorrência e evento (ISSUE-132)
type: specification
created_date: '2026-11-16T16:28:00Z'
updated_date: '2026-11-16T16:30:00Z'
tags:
  - estudo
  - identidade
  - contrato
---
**ISSUE-132 · Advisor · 16/11/2026 · Base: `75b1aff` (ISSUE-131 aceita, ainda não publicada).**

Contrato aceito pelo usuário em 16/11/2026 e formalizado na decision-008. Este documento não autoriza publicação nem escolhe prazo de retenção ou guarda de payload bruto. Complementa o doc-036; não repete a investigação da ISSUE-116 nem a correção da ISSUE-131.

## Parecer e limite do estudo

**Parecer do Advisor: APROVADO para planejamento. Contrato aceito pelo usuário em 16/11/2026.** Separar gravação, ocorrência e evento é necessário para preservar o lançamento ouvido e reunir gravações no ranking. A implementação deve aguardar o contrato de retenção/payload da ISSUE-079, a coordenação de schema da ISSUE-080; o aceite deste contrato já foi registrado. A ISSUE-132 não resolve as ISSUE-079/092 por antecipação.

O estudo foi executado diretamente pelo Advisor, sem delegação. Artefatos limitados ao Backlog e fixtures conceituais; nenhuma alteração de produção. Não há mudança de versão ou CHANGELOG.

## Evidência atual e lacunas

| Ponto examinado | Consequência atual e impacto futuro |
| --- | --- |
| `internal/catalog/catalog.go:94–109,223–233` | Associação serializada só dentro do processo; remove o escopo para catálogo compartilhado, mas grava a escuta com ProfileID. O evento não guarda o lançamento recebido. Um mutex/singleflight não substitui unicidade e transação no banco. |
| `internal/catalog/associate_track.go:47–75` | A ISSUE-131 já impede MBID incompatível no fallback textual, mas o acerto global por recording MBID ainda devolve a faixa do primeiro álbum. Não é uma regressão da 131: é a limitação estrutural deste recorte. |
| `internal/db/sqlite/listen.go:22–49` e migração `003_listening_profiles.sql:52–70` | Chave `(profile_id, track_id, listened_at)` e ON CONFLICT DO NOTHING colapsam eventos distintos no mesmo segundo. Não serve como identidade definitiva de evento. |
| `engine/handlers/lbz_submit_listen.go:187–303` | Processa lote item a item, mistura metadado enviado e mbid_mapping, responde 500 ao erro de associação; itens anteriores podem já ter sido gravados. Relay usa o corpo original após sucesso, independente da identidade resolvida. |
| `internal/importer/lastfm.go:78–81,122–148` | Álbum vazio vira título da faixa. Erro de associação encerra a importação antes dos itens seguintes. |
| `internal/importer/upload.go:17,56–130` | Contadores não distinguem pendente; staging temporário é removido ao sair. Pendência não pode guardar só caminho desse arquivo. |
| `internal/listenbrainz/types.go:27–41` | Não representa track_mbid, disco ou posição. Esses campos não chegam ao resolvedor atual. |
| `internal/db/sqlite/exports.go:17–28` | Cursor `(listened_at,track_id)` não distingue dois eventos da mesma ocorrência no mesmo segundo. Evoluir para ID estável e versão de formato. |
| `internal/db/sqlite/track.go` (MergeTracks) | A fusão atual deduplica por perfil/faixa/segundo. Não reutilizar esse descarte ao unir gravações/ocorrências. |

Localizações referem-se à base examinada. Navegação feita com graphify antes das leituras; nenhuma base real foi aberta.

## Contrato de entidades e identidades

| Entidade | Identidade e invariantes propostos |
| --- | --- |
| Gravação | ID interno estável. Recording MBID validado e único quando conhecido. Sem MBID, identidade provisória; sem fusão automática por título, duração ou grafia. Não representa a composição. |
| Lançamento | ID atual preservado; release MBID identifica a edição. Release-group agrupa edições sem substituir sua identidade. |
| Ocorrência | ID estável da faixa naquele lançamento. Preservar o atual track_id e suas URLs como ocorrência; relacionar recording_id. Guardar track MBID, meio/disco, posição e créditos apresentados quando disponíveis. Não impor UNIQUE(recording_id,release_id) nem UNIQUE(title,release_id). |
| Recebimento | Entrega de uma fonte, perfil autenticado, origem, received_at, identificador externo/recibo e snapshot observado. Reenvios não são necessariamente novas escutas. Recebimento pode ser duplicado de outro ou ficar pendente. |
| Evento de escuta | ID estável independente de gravação/instante; profile_id, ocorrência resolvida, listened_at original e precisão conhecida, cliente e duração observada/efetiva com proveniência. Pode ter vários recebimentos de fontes diferentes associados após decisão explícita. |
| Revisão | Pertence ao perfil e ao recebimento; motivo, candidatos, versão da regra, autor, decisão e revisão monotônica. Não muda catálogo compartilhado implicitamente. |

O ledger ingest_events da ISSUE-080 guarda recebimentos e decisões; listens guarda eventos aceitos. Uma pendência existe duravelmente no ledger sem listen provisório contabilizado. Evitar duas tabelas independentes para a mesma fila. A relação recebimento→evento permite conciliar fontes sem perder a proveniência de nenhuma delas.

IDs externos devem indicar o namespace (recording, track, release), a fonte e seu nível de confiança. UUID sintaticamente válido não comprova o tipo nem a correção do ID. Se o tipo não puder ser validado, conservar a afirmação original e não usá-la para substituir identidade consolidada.

## Resolução de ocorrência

1. ID de ocorrência/track MBID validado e coerente com gravação e lançamento.
2. Lançamento identificado + disco/posição conhecidos; recording MBID valida a compatibilidade. Contradição vai para revisão, sem sobrescrita.
3. Recording MBID identifica a gravação; só identifica a ocorrência com contexto suficiente e candidato único no lançamento. Em A/B, criar/reutilizar ocorrências distintas ligadas à mesma gravação, preservando o lançamento recebido.
4. Aliases exatos + lançamento + inclusão dos artistas recebidos, sem depender de ordem/repetição. Múltiplas candidatas resultam em pending_identity; nenhum SELECT LIMIT 1 silencioso. Uma candidata com MBID incompatível também fica pendente.
5. Sem evidência externa nem candidatas, criar gravação/ocorrência provisórias; preservar o caráter provisório. Candidatas ambíguas não autorizam criar uma nova como se estivesse resolvido.

Comparação Unicode/caixa fica na ISSUE-136 e agrupamento remaster/deluxe nas ISSUE-137/138. Duração é sinal auxiliar, não identidade. Não assumir que MBIDs enviados têm prioridade sobre contradições; registrar ambos os lados e encaminhar à revisão.

**Álbum ausente:** não inventar álbum usando título da faixa. Propor ocorrência provisória sem lançamento (release_id nulo, origem unknown), ligada à gravação quando identificável. Nenhuma atribuição automática a um álbum conhecido da gravação. A migração deve adaptar views/INNER JOINs que hoje exigem release e apresentar “Lançamento desconhecido”. Escuta conta no total e na gravação, mas em nenhum álbum real; incluir o balde “sem lançamento” na reconciliação. Se surgir o álbum depois, revisão explícita pode reassociar o evento e preservar o snapshot anterior. Fonte com ID estável de item pode reutilizar a ocorrência provisória; sem contexto suficiente, não se presume que duas ocorrências provisórias são a mesma.

## Identidade do evento e colisões

Identidade musical e deduplicação são decisões independentes. Um mesmo segundo pode conter duas escutas legítimas, duas ocorrências distintas ou o mesmo evento recebido de duas fontes.

- ID externo estável e confiável: unicidade de recebimento por `(profile_id, source_instance_id, external_event_id)`. O mesmo ID com payload incompatível é conflito de entrega, não sobrescrita silenciosa. Mudança de token não muda source_instance_id. A chave identifica o recibo lógico; tentativas com conteúdo divergente precisam de observações anexas imutáveis, preservadas antes do ACK, sem violar a unicidade nem apagar o snapshot anterior.
- Arquivo: identidade por perfil + formato + digest do conteúdo exato + ordinal absoluto do item. source_instance_id identifica a fonte de importação de forma estável, não o UUID temporário do upload. Preservar repetições dentro do arquivo. Reenviar o mesmo conteúdo, mesmo com outro nome, reutiliza recebimentos. Arquivo reordenado ou parcialmente sobreposto é novo recebimento e passa por deduplicação explicável.
- API sem ID: aceitar chave de idempotência como extensão opcional, mas não exigir dos clientes ListenBrainz. Igualdade de perfil, segundo e metadados sem recibo confiável é insuficiente para garantir exatamente uma vez. Propor recebimento durável e estado pending_duplicate para a segunda entrega indistinguível; não criar a segunda contagem nem apagar os dados. Revisão pode concluir Igual (link ao evento existente) ou Diferente (novo evento).
- IDs externos distintos na mesma fonte permitem eventos distintos no mesmo segundo, salvo evidência explícita contrária. Fontes diferentes podem atribuir IDs diferentes ao mesmo evento: conciliar sem presumir nem igualdade nem diferença apenas por isso.

Estados propostos: received → accepted | pending_identity | pending_duplicate | duplicate | invalid; falhas operacionais ficam retryable com tentativa e motivo. Erros de formato rejeitados antes de receber o lote não exigem recibo; invalid é uma classificação de conteúdo já preservado, não um ACK falso de um lote inválido. Accepted e duplicate referenciam o evento escolhido; invalid nunca vira descarte sem razão. Reprocessamento preserva receipt_id e é idempotente. Uma decisão nova aumenta resolution_version e pode substituir a associação, conservando a trilha; não cria outro evento apenas por tentar novamente.

Resolver uma pendência é transação curta: conferir perfil e versão → validar ocorrência/candidatos → criar ou ligar evento → atualizar decisão → gravar intenção de outbox quando elegível → commit. Usar CAS/versionamento para dois revisores; o perdedor recebe conflito e recarrega. Nenhuma chamada HTTP externa dentro da transação. Em erro, não deixar listen sem decisão nem duas intenções de relay. Constraints no banco sustentam o contrato entre processos.

## API ListenBrainz e importação Last.fm

A API oficial documenta HTTP 200 para recebimento aceito e trata playing_now como temporário. Propomos manter `200 {"status":"ok"}` após a persistência durável de todos os itens válidos, mesmo quando algum aguarda revisão; 200 significa recebido, não associação concluída. Isso é uma escolha do TuneTrail compatível com o formato de resposta, não uma garantia externa sobre resolução. Não usar 202/409 para conflito musical de um item já preservado. Pendências aparecem por consulta/gestão autenticada do próprio perfil (ISSUE-083); não mudar o envelope obrigatório dos clientes.

Validar formato/autorização/limites do lote antes de gravar: 400/401/413 para os respectivos erros; falha de persistência temporária retorna 503 sem declarar sucesso. Persistir o lote de recebimentos atomicamente antes do ACK; se a resposta se perder após commit, aplicar as regras de recibo/idempotência acima. Não prometer ausência de ambiguidade para cliente sem identificador. Uma falha posterior de associação é uma tarefa retryable, não um novo 500 para o cliente que já recebeu ACK.

`playing_now` permanece efêmero, sem evento, contagem ou fila histórica durável; exibir snapshot recebido quando não resolvido. A chegada de single/import cria seu próprio recebimento.

Last.fm: persistir o item antes de tentar associar. Conflito musical aumenta Pending e segue ao item seguinte; não aumenta Imported, Duplicates ou Ignored. Contadores de término satisfazem Processed = Imported + Duplicates + Pending + Ignored + Errors (itens inválidos explicitamente classificados). Interrupção operacional pode parar o lote com resultado parcial e cursor seguro; cursor só avança após cada recebimento durável. Retomada usa digest/ordinal e reprocessa pendências existentes, sem duplicar eventos aceitos. Não depender do staging já removido.

Relay fica condicionado à decisão aceita conforme ISSUE-082; pendência não deve disparar relay do corpo inteiro. Quando aceita, a outbox deve preservar os metadados observados compatíveis e sua proveniência. Remover/rever uma escuta já enviada não permite prometer desfazer um efeito remoto: reconciliar pelo contrato da integração.

## Estatísticas e compatibilidade

Ranking padrão por gravação, com chave por lançamento já decidida pelo usuário. Histórico e página de álbum usam ocorrência. Perfil P1 ouve R em A e B: 2 eventos em R, 1 em A, 1 em B. Perfil P2 ouve R em A: 1 evento em R e A, 0 em B. Consultas sempre partem dos eventos do perfil, nunca de todas as ocorrências da gravação.

Pendências não entram em ranking ou tempo ouvido; mostrar quantidade separada. Após resolução podem alterar contagens, com trilha de revisão. Gravações provisórias sem equivalência comprovada ficam separadas. Identidade global e preferências de agrupamento por perfil permanecem distintas.

Reconciliação: total de eventos aceitos = soma por ocorrência = soma por gravação = soma por lançamento **incluindo sem lançamento**. Um evento conta uma vez em cada dimensão. Créditos múltiplos não autorizam somar totais de artistas para inferir total de eventos (ISSUE-092). Primeira escuta por gravação usa MIN do perfil; por ocorrência preserva estreia naquele lançamento. Médias, desempates e benchmarks continuam na ISSUE-092. Duração efetiva histórica não deve mudar quando a duração do catálogo é editada (ISSUE-097).

Migração 1:1 preserva IDs de ocorrência, perfis, timestamps, clientes e eventos existentes, sem deduplicação adicional. Dados legacy sem álbum verdadeiro preservam o álbum conhecido e marca de qualidade; o álbum perdido no modelo antigo não é recuperável sem fonte externa. Não converter automaticamente álbuns cujo título coincide com o da faixa em “desconhecido”.

Exportação evolui para formato versionado contendo event_id/namespace de origem, ocorrência, gravação, proveniência e revisão; exportar pendências em seção explícita de backup, sem representá-las como escutas aceitas. Cursor passa a `(listened_at,event_id)` por perfil. Importar IDs de outra instalação exige namespace/mapeamento, não copiar PK globalmente. Exclusão por event_id evita apagar duas escutas no mesmo segundo; rota legada ambígua não pode escolher uma. Merge de gravações não deduplica eventos; exclusão de ocorrência/release precisa de prévia dos eventos afetados, sem cascata para outras ocorrências/gravações.

## Fluxo e assinaturas conceituais

Assinaturas propostas para planejar ISSUE-081/134, não API implementada:

```go
ReceiveBatch(ctx context.Context, envelope BatchEnvelope) ([]Receipt, error)
ResolveReceipt(ctx context.Context, profileID int32, receiptID ReceiptID) (Resolution, error)
ReviewReceipt(ctx context.Context, profileID int32, receiptID ReceiptID,
    expectedVersion int64, decision ReviewDecision) (Resolution, error)
```

BatchEnvelope leva perfil derivado da autenticação, source_instance_id, referência de entrega, instante/precisão, tipo e snapshot original saneado. Resolution diferencia identidade e deduplicação, ocorrência/gravação/evento opcionais, razão e rule_version. Contratos públicos não expõem tokens, caminhos de staging ou IDs de outro perfil. Recibo aceito não pode ser reatribuído por trocar o ProfileID da requisição.

## Ordem de execução e arquivos

| Etapa | Responsável e alvos previstos |
| --- | --- |
| Aceitar identidade e preparar fixture | ISSUE-132: este doc-039 e evidências. ADR accepted registrada como decision-008 após aceite; critério #4 cumprido. |
| Fechar retenção e payload | ISSUE-079: envelope/razões/retention em backlog/docs/design. Sem prazo de retenção ou armazenamento indiscriminado de bruto decidido aqui. |
| Persistir ledger e outbox | ISSUE-080: db/migrations_sqlite, internal/db/db.go, opts.go e sqlite/bootstrap.go; versão coordenada, backup e restauração. |
| Evoluir identidade musical | ISSUE-133: migração incremental, models/track.go, sqlite/track.go, aliases e FKs; preservar IDs. Aplicação precisa continuar utilizável entre 133 e 135, com camada de compatibilidade ou plano de corte transacional documentado. |
| Ingestão e recibos | ISSUE-081 + 134: novos internal/ingest e internal/dedup conforme tarefas existentes; catalog/catalog.go e associate_track.go, listenbrainz/types.go, handlers/lbz_submit_listen.go, importer/*.go e upload.go. Testar lote e restart. |
| Relay e gestão de revisão | ISSUE-082/083: outbox elegível e consulta/resolução por perfil. Não declarar substituição funcional completa enquanto houver fila sem mecanismo de revisão. |
| Leituras e operações | ISSUE-135: sqlite/listen.go, track.go, exports.go, scoped_reads.go, internal/listening/now_playing.go, webquery/UI e handlers de edição/exclusão; benchmark coordenado com 092. |
| Normalização e agrupamentos | ISSUE-136/137/138, depois dos contratos e dependências já registrados. |

Não adicionar dependência circular entre 079 e 132: são contratos coordenados. 133 já depende de 080/132; 134 já depende de 081/133. A 081 exige comportamento observável preservado, mas pendências e ACK após recebimento mudam a semântica do erro atual: ajustar explicitamente seu critério ao aceitar o contrato, sem fingir neutralidade. A 135 precisa reconhecer o balde sem lançamento na igualdade dos totais.

## Riscos e limites do contrato aceito

1. **Retenção/payload (079):** propor mínimo estruturado suficiente para reprocessar (textos recebidos, IDs declarados e origem, data/precisão, cliente, referência da fonte e razões), sem tokens/cookies. Bruto completo e prazos exigem decisão do usuário. Recebimentos não resolvidos não podem ser podados como auditoria expirada antes dessa decisão; nenhum prazo é fixado neste estudo. Purga/exclusão deve impedir ressurreição por replay sem guardar payload sensível indefinidamente.
2. **Sem ID confiável:** impossível distinguir automaticamente reenvio idêntico de repetição legítima indistinguível. Proposta conservadora: pending_duplicate; nunca afirmar exatamente uma vez nesse cenário.
3. **Álbum ausente:** ocorrência sem lançamento e balde explícito exigem adaptar leituras legadas. Evitar o atalho de inventar um álbum ou escolher o primeiro conhecido.
4. **Catálogo compartilhado:** correção de identidade global repercute em todos os perfis. Uma revisão de evento muda a associação daquele perfil; mudança global exige ação administrativa com prévia e revisão de referências, fora de uma preferência pessoal.
5. **Fila durável:** limites de armazenamento e backpressure precisam impedir ACK sem durabilidade. Reinício deve localizar received/retryable sem depender de goroutine viva. Contexto cancelado não avança cursor nem confirma lote não commitado.
6. **Entrega incremental:** não pôr em produção schema que permite ocorrências duplicadas de MBID com leitores ainda assumindo uma faixa global por MBID. 133 precisa documentar ponte; 134 deve habilitar escrita nova apenas com leituras essenciais compatíveis.

## Fixtures e validação

`backlog/evidencias/issue-132/fixture-identidade.sql` materializa apenas o contrato conceitual em SQLite temporário, não uma migração do TuneTrail. `validar-fixture.py` verifica contagens A/B por perfil, duas repetições no mesmo segundo, pendências fora das estatísticas, álbum ausente, recibo duplicado, resolução idempotente, rollback e isolamento de revisão. Invariantes de esquema/contagem não provam implementação do resolvedor, transporte HTTP, concorrência entre processos, desempenho, migração ou política de retenção; essas verificações pertencem às tarefas acima.

## Fontes primárias

Consultadas em 16/11/2026. A distinção proposta coincide com MusicBrainz: uma gravação pode aparecer em várias tracks; track é sua representação em um lançamento/meio. [Recording](https://musicbrainz.org/doc/Recording), [Track](https://musicbrainz.org/doc/Track).

ListenBrainz documenta HTTP 200 para listens aceitos e playing_now temporário. A proposta de ACK após persistência local é uma inferência de design do TuneTrail, não uma promessa de processamento síncrono do serviço externo. [Core API](https://listenbrainz.readthedocs.io/en/latest/users/api/core.html), [Submission JSON](https://listenbrainz.readthedocs.io/en/latest/users/json.html).

Na documentação atual, additional_info contém afirmações do cliente não validadas; mbid_mapping é enriquecimento retornado pelo servidor e não pertence ao formato de envio. Portanto, guardar separadamente os dois e não aceitar o mapping recebido como autoridade resolve uma lacuna real do adaptador atual. [MBID Mapping](https://listenbrainz.readthedocs.io/en/latest/users/json.html#mbid-mapping).
