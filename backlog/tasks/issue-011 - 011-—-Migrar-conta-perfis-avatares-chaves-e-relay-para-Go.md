---
id: ISSUE-011
title: 011 — Migrar conta, perfis, avatares, chaves e relay para Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
references: []
type: feature
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/011-administracao-perfis-chaves-go-solved.md` — numeração local 011. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](<../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>). Depende de 010.

## Trabalho

- Implementar login/logout/conta e página `/admin`, preservando seleção por ID em
  `?profile=`, retorno interno e acesso com zero perfis.
- Criar/editar perfis com serviço existente, chave inicial atômica, conflitos de
  nome/slug e conservação do ID/escutas/chaves após renomear.
- Migrar avatar (foto/ícone/iniciais), upload até limite vigente e limpeza de
  arquivos com referência global; informar falha de foto após perfil já criado.
- Listar/revelar/copiar/criar/renomear/revogar chaves do perfil explícito, sem
  reatribuição entre perfis; alternativa de seleção/cópia manual sem Clipboard API.
- Migrar configurar, testar e remover relay; teste não salva nem envia escuta.
  Aplicar política de rede coerente no teste/envio, incluindo timeout, redirects,
  resposta limitada e configuração self-hosted suportada; sem fila durável nova.
- Proteger HTML/JSON/segredos com sessão e no-store, e novas actions HTML com
  CSRF; JSON existente mantém SameOrigin, salvo mudança coordenada/testada. Não colocar
  token em URL, flash, log ou screenshot. Continuar tratamento de erros com contexto.
- Preparar formulários GET/POST e alternativa sem JS. Exclusão final de perfil
  e confirmação destrutiva são integradas com a 014.

## Aceite

Criar perfil/chave atomicamente; duplicidade e erro de banco; rename preserva ID,
URL antiga 404; último perfil removido não remove admin. Foto inválida/grande,
falha de commit e foto compartilhada; duas abas com perfis diferentes.

Chave de outro perfil retorna 404 ao renomear/revogar; revogação bloqueia ingestão
futura e preserva história; falha de copiar permite copiar manualmente. Relay
simulado local: token válido/inválido, erro/timeout/redirect, salvar/testar/remover
independentes. Nenhum contato com conta externa real. Conta/senha não altera perfil.
Capturas fictícias e regressões de autorização em bancos v3 descartáveis.

## Andamento — 05/11/2026

O handler HTML Go está preparado e coberto por testes funcionais, sem montagem
no dispatcher de produção: `/admin` ainda depende de importação/exportação da
issue 012. A integração completa, login com limitador compartilhado entre HTML e
JSON, layout e capturas ficam para a 015. Exclusão de perfil permanece na 014.

Os serviços de perfil, avatar, chaves e relay são compartilhados com a API JSON.
O relay usa a mesma política de rede no teste e envio. A publicação do avatar é
coordenada com a coleta de órfãs no processo; fotos anteriores aguardam a coleta.
Detalhes e testes: [relatório funcional](../evidencias/roadmap-04/ISSUE_011.md).

## Integração na 015 — 05/11/2026

A administração foi ligada ao modo Go; os contratos e jornadas são verificados
pelo servidor real e pelas capturas comparativas. As pendências de integração
descritas no andamento anterior são históricas. Consulte
[entrega e limites da 015](../evidencias/roadmap-04/ISSUE_015.md).

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário ao solicitar o encerramento
de todas as issues abertas até a 015. Entrega, testes e limites conhecidos
permanecem registrados; validação transversal será estudada no roadmap 5.
<!-- SECTION:DESCRIPTION:END -->
