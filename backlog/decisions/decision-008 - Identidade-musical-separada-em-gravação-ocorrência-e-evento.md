---
id: decision-008
title: Identidade musical separada em gravação, ocorrência e evento
date: '2026-11-16T17:05:00Z'
status: accepted
---
## Context

A ISSUE-131 corrigiu aliases secundários e passou a rejeitar ambiguidades e MBIDs conflitantes. O modelo atual ainda associa uma gravação a um único lançamento e identifica escutas por perfil/faixa/segundo. Isso perde a distinção entre ocorrências em álbuns diferentes e entre repetições legítimas no mesmo segundo.

Contrato detalhado: doc-039, `backlog/docs/design/doc-039 - Contrato-de-identidade-—-gravação-ocorrência-e-evento-ISSUE-132.md`. Evidências conceituais: `backlog/evidencias/issue-132/`; 21 verificações SQLite passaram. Aceite explícito do usuário em 16/11/2026: “aceito vamos resolver a issue 132”.

## Decision

Separar gravação, ocorrência num lançamento, evento de escuta e recebimento de fonte. Preservar IDs e URLs das faixas atuais como ocorrências; migrar eventos existentes 1:1, sem fusão textual nem descarte adicional.

Ranking padrão por gravação, com opção por lançamento. Histórico e páginas de álbum usam a ocorrência ouvida e o escopo do perfil. A mesma gravação ouvida em A e B soma dois eventos na gravação e um em cada álbum.

Álbum ausente resulta em ocorrência provisória sem lançamento e balde explícito “sem lançamento”; não inventar álbum pelo título da faixa nem escolher outro álbum da gravação. Conservar a atribuição histórica conhecida na migração sem reconstrução especulativa.

Eventos têm identidade estável independente de gravação/segundo. Recibos com IDs confiáveis ou digest/ordinal de arquivo sustentam idempotência. Entregas indistinguíveis sem identificador confiável ficam pendentes de revisão; não prometer exatamente uma vez automático. Preservar repetições legítimas.

Ambiguidade de alias e conflito de MBIDs preservam o recebimento duravelmente, sem associação silenciosa ou contagem prematura. Revisão e reprocessamento são transacionais, idempotentes, versionados e restritos ao perfil. Alterar identidade global requer ação administrativa explícita.

ListenBrainz recebe HTTP 200 no formato compatível depois da persistência durável do lote válido, inclusive pendências; falha de persistência não confirma recebimento. Playing_now permanece efêmero. Importação Last.fm continua após conflitos musicais e retoma por recebimentos estáveis, distinguindo Pending dos demais resultados. Relay depende de decisão elegível e outbox; não retransmitir indiscriminadamente um lote com pendências.

## Consequences

Implementação distribuída entre ISSUE-080/081/082/083/133/134/135, conforme doc-039. Retenção, prazo e guarda de payload bruto continuam exigindo definição na ISSUE-079; este aceite não escolhe esses valores. Semântica de médias e benchmarks permanece na ISSUE-092.

O esquema precisa admitir ocorrências sem lançamento, várias ocorrências por gravação/lançamento e vários eventos no mesmo segundo. Leitores, exportação versionada, cursores, edição, exclusão e fusão precisam acompanhar a migração. A ISSUE-133 exige ponte de compatibilidade ou corte coordenado para não habilitar escritores novos com leitores antigos incompatíveis.

Pendências não entram em estatísticas até resolução; sua existência deve ser consultável e revisável pelo perfil. Contagens por lançamento só se reconciliam com o total se incluírem “sem lançamento”. Sem evidência confiável, a revisão humana continua necessária.

Rejeitados: manter um único lançamento por gravação, relação N:N sem ocorrência na escuta, deduplicar por gravação/segundo, fundir por texto ou devolver sucesso antes da persistência. Esses atalhos perdem contexto ou eventos. A fixture valida o contrato conceitual, não a implementação de produção.
