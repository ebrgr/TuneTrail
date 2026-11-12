---
id: ISS-020
title: 020 — Validar concorrência, falhas, importação/exportação e I/O
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
references: []
type: task
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/020-concorrencia-falhas-io-solved.md` — numeração local 020. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida por aceite do usuário em 05/11/2026**. Issue local, sem publicação.
Plano: [roadmap 5](<../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>). Dependências: 016–019 como base de contratos; execução integrada depende de 018/019.

## Trabalho

- Criar interleavings determinísticos com barreiras/deadlines para ingestão versus merge/delete/revisão/revogação/exclusão de perfil; evitar sleeps como prova de ordem.
- Definir a fronteira de autorização de requisição já iniciada versus revogação/exclusão; revogar não implica cancelamento retroativo automático.
- Conferir409 em plano alterado e nenhuma escrita parcial indevida; testar rollback, falha de commit, WAL e uma conexão SQLite onde pertinente.
- Importar cinco formatos, repetir arquivo, cancelar durante leitura/processamento, JSON/ZIP truncado e orçamento expandido; reinício não reimporta temporário HTTP.
- Export >1000 e timestamps empatados sob ingestão/delete: testar consistência dinâmica já aceita, writer com erro e desconexão, sem prometer snapshot ou JSON de sucesso após stream incompleto.
- Imagem/avatar: transporte lento/grande/redirect, falha de disco/commit/limpeza, referências compartilhadas e compensação; rede externa somente simulada localmente.
- Relay:429/5xx/timeout/cancelamento e término do processo; preservar best-effort, sem fila durável. Verificar interação do Sender 3min com shutdown HTTP 10s.
- Avaliar leitura integral do corpo ListenBrainz em lbz_submit_listen.go antes do teto1000; reproduzir consumo com carga autenticada limitada e definir limite de bytes compatível, se necessário.
- Observar duração, memória e recursos em cenários fixos com teto explícito; verificar goroutines/temporários/processos remanescentes. Não transformar medição local em SLA.

## Aceite

Operações atômicas preservam invariantes sob falha; import parcial/relay/export dinâmico relatam seus limites corretamente. Sem corrupção, segredo em logs ou arquivo pendente indevido. Cancelamento e encerramento têm observáveis e deadlines, não alegações genéricas. Limites novos exigem regressão e documentação de compatibilidade.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../evidencias/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Execução parcial — 05/11/2026

Correções e regressões executadas estão em [ISSUE_020.md](../evidencias/roadmap-05/ISSUE_020.md),
com os limites observados e os cenários ainda pendentes. A issue foi fechada por aceite do usuário; as limitações registradas permanecem visíveis para a integração.
<!-- SECTION:DESCRIPTION:END -->
