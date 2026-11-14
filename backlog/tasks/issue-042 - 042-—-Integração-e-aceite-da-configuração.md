---
id: ISSUE-042
title: 042 — Integração e aceite da configuração
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-13T23:39:00Z'
labels:
  - config
milestone: m-4
dependencies:
  - ISSUE-038
  - ISSUE-039
  - ISSUE-040
  - ISSUE-041
priority: high
type: feature
ordinal: 108000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/042-aceite-migracao-configuracao.md` — numeração local 042. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Implementada e validada — aguardando aceite · **Depende de:** 038–041

## Objetivo

Validar os dez campos e a experiência administrativa definidos no
[contrato da 038](<../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>), incluindo documentação de
implantação, atualização e recuperação offline.

## Aceite

- Executar a matriz do contrato: arquivo ausente/presente/inválido, defaults,
  ambiente e `_FILE` (inclusive vazio/falha), duas composições e overrides.
- Salvar, reiniciar e conferir valores ativos e persistidos dos dez campos,
  campos visíveis no admin e ausência de nova configuração nas respostas públicas.
- Cobrir sessão admin, negativas para visitante/chave, CSRF/origem, conflitos,
  readonly e falhas de escrita, sem confirmação adicional de senha por save.
- Validar fuso forçado/automático versus TZ, Subsonic em servidor simulado,
  teste de rascunho sem salvar e recuperação offline após configuração inválida.
- Conferir relay por perfil já existente, reinício sem reassociação do legado,
  importação e integridade; preservar SQLite v3, dados e arquivos de migração.
- Verificar execução local, Docker/Compose, VS Code e reset-password; registrar
  limites, captura desktop/celular e testes pertinentes ao código final.
- Atualizar docs de uso, exemplo TOML fictício, changelog e roadmap; não apresentar
  planejamento como funcionalidade disponível nem publicar automaticamente.

## Resultado — 09/11/2026

Matriz de configuração, reinícios, recuperação offline, permissões, migração/importação e execução real via Compose foram validadas. Exemplo TOML e documentação de implantação/recuperação adicionados. Revisão do Advisor aprovada. O lançamento equivalente pelo Delve passou; o F5 da interface gráfica do VS Code não foi exercitado. [Evidências e limites](../evidencias/configuracao/ISSUE_042.md).

## Ajuste solicitado — bloqueio por ambiente

Os campos controlados por ambiente ou `_FILE` ficam desabilitados e informam o nome da variável. A gravação não altera o valor existente no TOML e rejeita tentativas forjadas; o teste Subsonic usa o valor efetivo dos campos bloqueados. A precedência permanece inalterada.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-13T23:39:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026, com instrução explícita para fechar a ISSUE-042. Encerramento baseado nas verificações e limitações já registradas; sem nova execução de testes nesta etapa.

---
<!-- COMMENTS:END -->
