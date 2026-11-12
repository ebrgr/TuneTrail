---
id: ISS-041
title: 041 — Edição de configuração pelo master
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - config
milestone: m-4
dependencies:
  - ISS-038
  - ISS-039
  - ISS-040
references: []
type: feature
ordinal: 41000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/041-administracao-configuracao-solved.md` — numeração local 041. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Fechada e aprovada pelo usuário em 09/11/2026 · **Depende de:** 038–040

## Objetivo

Criar administração Go para os dez campos globais do
[contrato da 038](<../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>), com login/senha e sessão
administrativa existentes. Valores são visíveis/editáveis, sem mascaramento.

## Aceite

- Implementar navegação e rotas próprias propostas em `/admin/settings`, save e
  teste Subsonic; autorizar leitura/escrita no servidor somente para admin.
  Reaproveitar sessão, origem, CSRF, no-store e limites; negar visitante e chave
  de ingestão. Não exigir nova senha a cada salvamento.
- Mostrar todos os valores solicitados, inclusive Last.fm e Subsonic params,
  sem tratamento especial de segredo. Não ampliar o endpoint público de configuração
  nem exibir a senha de login administrativa como opção de configuração.
- Mostrar valor ativo, salvo, origem/override e reinício pendente. Não prometer
  aplicar TOML enquanto existir override. Preservar entrada em erros e indicar conflitos.
- Fuso: oferecer IANA forçado ou automático; informar `TZ` observado somente como
  dado do processo. Não editar Compose nem alterar `time.Local` em execução.
- Subsonic: URL/params juntos, fetch de dados/imagens, teste explícito do rascunho
  com helper da 039; sem persistir, alterar cliente ativo ou testar ao salvar.
  Informar diferença entre rascunho testado e valor efetivo por override.
- Linkar a gestão existente de relay por perfil; não duplicar URL/token no TOML
  global, não criar donos de perfil e não mudar bootstrap/realocação de dados.
- Usar templates Go e JS essencial; cobrir teclado, erro, no-JS e permissões.
  Registrar capturas desktop/celular para aceite na 042.

## Implementação e validação — 09/11/2026

Tela `/admin/settings` integrada à administração, com os dez campos visíveis,
estados salvo/efetivo/ativo, origem, override, reinício, preservação de rascunho
e teste Subsonic sem salvar. O relay permanece por perfil, com token visível.
Login, sessão, CSRF, origem, limites e tickets existentes protegem as rotas.

[Comparações desktop/celular, testes e limites](../evidencias/configuracao/ISSUE_041.md).
A suíte Go, race dirigido e jornadas administrativas passaram; Advisor aprovou
após correções. A 042 continua responsável pelo aceite integrado ampliado.

## Encerramento

Issue fechada e aprovada por solicitação do usuário em 09/11/2026. A validação integrada continua na issue 042.
<!-- SECTION:DESCRIPTION:END -->
