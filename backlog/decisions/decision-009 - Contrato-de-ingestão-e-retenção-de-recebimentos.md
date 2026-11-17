---
id: decision-009
title: Contrato de ingestão e retenção de recebimentos
date: '2026-11-17T15:28:00Z'
status: accepted
---
## Context

A ISSUE-079 formaliza o contrato de recebimentos, deduplicação e retenção, complementando a identidade aceita da ISSUE-132 (doc-039/decision-008). As cinco políticas abaixo foram decididas pelo usuário em comentário na ISSUE-079 em 17/11/2026. Esta redação permanece **proposed**, conforme pedido explícito, até revisão/aprovação do usuário; não reabre as cinco escolhas nem autoriza implementação ou publicação.

Contrato completo: doc-040, `backlog/docs/design/doc-040 - Contrato-de-ingestão-retenção-e-deduplicação-ISSUE-079.md`.

## Decision

1. Detalhes de processamento encerrados têm retenção configurável entre 7 e 3650 dias, padrão de 90 dias. Preservar escutas, pendências e metadados mínimos necessários enquanto existirem.
2. Payload bruto desativado por padrão. Diagnóstico saneado é opcional por perfil e fonte, por sete dias, quota de 100 MiB por perfil e máximo de 1 MiB por item; não substitui snapshot estruturado obrigatório.
3. Tabelas de metadados da ISSUE-055 ficam fora da v4 e serão implementadas em migração posterior. V4 contém somente infraestrutura de ingestão, coordenada com a identidade das ISSUE-133/134.
4. Não manter chaves de reimportação/tombstones após excluir uma escuta. A exclusão nunca bloqueia importação futura: o mesmo item reimportado vira evento novo após resolução/aceite. Idempotência por recibo vale enquanto recibo/evento existir. Remover bloqueios equivalentes de caches, índices, lote/arquivo e outbox vinculada; não ressuscitar evento excluído.
5. Deduplicação conservadora: Igual, Diferente ou Revisar com evidência e regra versionadas; ambiguidade exige revisão por perfil. Janelas temporais apenas selecionam candidatos, sem comprovar igualdade ou autorizar descarte.

## Consequences

O ledger mantém recebimentos e snapshots; pendências ficam fora das estatísticas. Purga de detalhes não apaga histórico ou mínimo vigente. Exclusão explícita remove recibos/chaves vinculados que impediriam recriação; reenvio automático da fonte também pode recriar a escuta. Arquivo parcialmente excluído deve reutilizar itens vivos e recriar os excluídos, sem bloqueio global por digest de lote.

Outbox é transacional e entrega pelo menos uma vez; efeito remoto já produzido não é desfeito por exclusão local. Workers atrasados não podem recriar dados locais excluídos. Não há promessa de exatamente uma vez para fonte sem ID confiável ou destino sem suporte correspondente.

ISSUE-080 precisa separar núcleo/detalhes, conflitos e revisão, sem tabelas finais da 055 nem tombstones. ISSUE-133 preserva IDs/URLs e eventos 1:1, introduz identidade estável e ponte de compatibilidade. ISSUE-134 adota envelope comum, revisão, retomada e reimportação após exclusão em todas as fontes. As edições dessas tarefas são recomendações, não foram executadas nesta spike.

Rejeitados: retenção única de todo o ledger; payload completo indiscriminado; dedup por faixa/gravação e segundo; tombstones pós-exclusão; schema de tags/datas ainda não validado dentro da v4.
