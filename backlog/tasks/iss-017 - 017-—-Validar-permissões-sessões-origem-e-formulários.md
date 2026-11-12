---
id: ISS-017
title: 017 — Validar permissões, sessões, origem e formulários
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - validacao
milestone: m-1
dependencies: []
references: []
type: task
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/017-permissoes-sessoes-origem-solved.md` — numeração local 017. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 5](<../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>). Dependências: 016.

## Trabalho

- Exercitar servidor real em Go e React/API com visitante, administrador, não-admin sintético, sessão expirada/revogada, cookie inválido/duplicado e chaves de ingestão A/B/revogadas.
- Cruzar cada família administrativa, download e método com os atores; chave nunca autoriza gestão, payload/query não altera identidade.
- Validar login/logout/senha, prazo fixo24 h/30 d, revogação, limitador HTML/JSON compartilhado, no-store e ausência de credenciais em URL/flash/log.
- HTML: CSRF ausente/inválido/expirado/de outra sessão, nonce de login, multipart antes/depois de token, ticket ausente/repetido/duas abas e reinício.
- Origin/Fetch Metadata: ausente, same-origin, cross-site e host enganoso; preservar contrato JSON sem exigir token novo.
- GET/HEAD e métodos inesperados não executam ações; validar SafeReturn contra redirecionamento externo e perfil diferente.
- Verificar cookies HTTP/TLS/proxy controlado segundo implantação documentada; não redesenhar confiança de proxy nesta issue.

## Aceite

Negativas comprovam ausência de mutação/arquivo novo, além do status esperado. Erro de infraestrutura é distinguido de autenticação inválida onde o contrato exigir. Todos os fluxos válidos continuam funcionando sem JS quando aplicável. Alteração necessária recebe regressão prévia e changelog se visível.

## Evidências

Registrar procedimentos e resultados em [roadmap 5](../evidencias/roadmap-05/README.md).
Usar apenas bases SQLite v3 descartáveis e credenciais fictícias. O estudo e o
fechamento de 001–015 não significam que esta issue já foi executada.

## Entrega — 05/11/2026

Corrigidos logs de parâmetros e senha inicial, logout com cookies duplicados e
cache de rejeição por origem. [Relatório de testes, evidências e limites](../evidencias/roadmap-05/ISSUE_017.md).
Matriz com servidores reais, testes Go/race/vet e jornadas administrativas
passaram; Advisor aprovou. Aceite explícito do usuário recebido em 05/11/2026.

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário.
<!-- SECTION:DESCRIPTION:END -->
