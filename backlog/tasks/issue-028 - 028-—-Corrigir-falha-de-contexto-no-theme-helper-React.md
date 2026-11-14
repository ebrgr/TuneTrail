---
id: ISSUE-028
title: 028 — Corrigir falha de contexto no theme-helper React
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-2
dependencies: []
type: bug
ordinal: 94000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/028-theme-helper-react-sem-perfil-solved.md` — numeração local 028. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada — resolvida** em 05/11/2026. Issue local, sem publicação.
Origem: [validação da issue 025](../evidencias/roadmap-06/ISSUE_025.md).

## Problema

Em `KOITO_WEB_UI=react`, acessar diretamente `/theme-helper` retorna o shell HTTP
200, mas a página hidratada falha antes de exibir o editor. Em Chrome com dados
fictícios, foi observada a mensagem `This screen requires a listening profile`.
A rota é irmã de `/:slug` e não está dentro de `ProfileProvider`; componentes de
prévia exigem esse contexto. A falha foi observada em fixture v3 e também em base
v2 migrada; não foi atribuída à migração.

## Trabalho e aceite

- Definir a semântica do helper global: preferir prévia com dados fictícios; se
  usar dados reais, exigir escolha explícita de perfil e tratar 0/1/N perfis.
- Acrescentar regressão de navegador para acesso direto, edição/aplicação do tema
  e ausência de exceções de contexto; testar 0/1/N perfis com dados descartáveis.
- Não escolher silenciosamente o primeiro perfil nem tornar `useProfile` opcional
  de forma global. Manter Go como padrão e React desativado por padrão.

## Resultado

A prévia usa dados fictícios independentes de perfil. Aplicação, validação e
restauração do tema foram corrigidas. Testes com 0/1/N perfis e base migrada
passaram; detalhes e limites em [ISSUE_028](../evidencias/roadmap-06/ISSUE_028.md).
<!-- SECTION:DESCRIPTION:END -->
