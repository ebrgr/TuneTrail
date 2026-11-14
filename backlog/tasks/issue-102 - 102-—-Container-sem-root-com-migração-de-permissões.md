---
id: ISSUE-102
title: 102 — Container sem root com migração de permissões
status: Aberta
assignee: []
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-14T15:00:00Z'
labels:
  - operacao
milestone: m-15
dependencies:
  - ISSUE-063
references:
  - backlog/evidencias/container/imagem-base.md
priority: medium
type: chore
ordinal: 102000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-017 (ISSUE-034) e doc-010 §8. `USER 65532`, nota de migração de ownership dos volumes existentes (inclusive Unraid) e teste com volume antigo.

## Complemento (14/11/2026): imagem base Alpine

Decisão do usuário: trocar a base `debian:bookworm-slim` pela Alpine junto com a execução sem root, porque as duas mudanças mexem no Dockerfile, no healthcheck e no smoke de contêiner do CI. Medição e comparação em `backlog/evidencias/container/imagem-base.md`: imagem atual 188 MB, dos quais 17 MB são o binário; Alpine estimada em ~30 MB. O binário é Go puro (CGO desligado), então musl e o resolvedor do Alpine não alteram o comportamento.

Passos:
1. Estágio final `alpine` com versão fixada, só `ca-certificates`; criar usuário e grupo sem privilégios (UID/GID 65532) e `USER` correspondente.
2. Fusos embutidos no binário (`time/tzdata` ou `-tags timetzdata`, ~450 KB), sem depender de `tzdata` do sistema; manter o teste de fuso (America/Sao_Paulo) por dentro do binário.
3. Healthcheck sem `curl`: subcomando `tunetrail healthcheck` que consulta `/apis/web/v1/health` em 127.0.0.1 com timeout curto e devolve o código de saída; usado no `HEALTHCHECK` do Dockerfile.
4. Permissões de volumes existentes criados por root (Docker e Unraid): detectar diretório sem escrita na partida e falhar com mensagem que indica o comando de correção (`chown -R 65532:65532 <dir>`) ou a opção `user:` do compose; documentar no DEPLOYMENT.md, incluindo PUID/PGID do Unraid.
5. Ajustar o smoke de contêiner do CI (hoje usa `dpkg-query` e confere `/usr/share/zoneinfo`) e o `scripts/validate_container.py`.
6. Validar como na 0.10: validate_container, simulação de atualização a partir da 0.10 com volume existente (`backlog/evidencias/identidade/simulacao-atualizacao.md` como modelo) e as três variantes do smoke do CI.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Imagem roda sem root
- [ ] #2 Volumes existentes migrados com instrução testada
- [ ] #3 Documentação de atualização
- [ ] #4 Imagem final em Alpine, abaixo de 40 MB, com certificados e fusos funcionando (teste de HTTPS e de fuso)
- [ ] #5 Healthcheck por subcomando do binário, sem curl
- [ ] #6 Smoke de CI e validate_container adaptados à nova base, com as três variantes de configuração
- [ ] #7 Atualização a partir da 0.10 com volume existente simulada e documentada
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T05:02:00Z

---

Complemento registrado a pedido do usuário: troca da base Debian slim (188 MB) pela Alpine (~30 MB) junto com a execução sem root; healthcheck nativo, fusos embutidos no binário e smoke de CI adaptado. Medição em backlog/evidencias/container/imagem-base.md.

---

author: claude
created: 2026-11-14T15:00:00Z

---

Coordenação com a ISSUE-066 (parecer do Advisor na ISSUE-061, item 7): na imagem sem root, o diretório assets/ deve pertencer ao root e ser só leitura para o usuário 65532, porque templates lidos do disco executam sobre os dados.

---
<!-- COMMENTS:END -->
