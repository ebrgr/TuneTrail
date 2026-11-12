---
id: ISS-010
title: 010 — Estabelecer contratos administrativos e corrigir bloqueadores de escrita
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
references: []
type: feature
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/010-contratos-escrita-admin-solved.md` — numeração local 010. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](<../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>). Precede 011–015.

## Trabalho

- Extrair identidade/sessão e regras de operações para serviços reutilizáveis por
  JSON e HTML; sem loopback HTTP nem uso de handlers como serviços.
- Definir prazo de sessão coerente entre banco e cookie, remember-me e revogação
  após alteração de senha; preservar bcrypt, HttpOnly, SameSite e configuração TLS.
- Especificar/implementar CSRF das novas forms: vinculado à sessão, comparação
  constante e rejeição antes de mutar; proteção pré-login distinta e logout;
  manter Origin/Fetch Metadata como defesa adicional. Em multipart, validar
  antes de spoolar arquivo, usando leitura sequencial limitada ou equivalente;
  não extrair token com ParseMultipartForm após já gravar temporários.
- Definir actions POST e validação de retorno interno; autenticação, autorização,
  IDs positivos/existência, limites de corpo, no-store e erros sem SQL/segredos.
- **Correções funcionais explícitas:** rejeitar self merge de artista e corrigir
  leitura de artistas de álbum dentro da transação; verificar/propagar erros de
  origem/destino no merge de faixa. Usar o mesmo contrato nas APIs existentes.
- Mapear 400/422, 401, 403, 404, 409 e 500; resultados e confirmação de escrita
  não podem depender apenas de botões desabilitados no cliente.
- Registrar mudanças visíveis/contratuais no changelog.

## Aceite

Reproduções atuais em [analysis-checks.json](../evidencias/roadmap-04/analysis-checks.json).
Testes devem provar que self merge falha sem alterar nenhuma tabela/escuta,
MergeAlbums funciona com uma conexão e leituras no tx, e IDs inexistentes/falha
injetada produzem rollback. Cobrir dois perfis compartilhando catálogo.

Visitante/chave/cookie expirado/não-admin; token ausente/incorreto e sessão
renovada/trocada; login sem sessão prévia; origem inválida; senha alterada;
redirecionamentos externos e POST repetido. Nenhuma nova tela de escrita é
ativada antes desses contratos. Usar somente bases descartáveis.

## Implementação e sequência

Os serviços, contratos HTTP/CSRF, sessão e correções de persistência estão
implementados. APIs JSON usam os serviços e mantêm seus payloads. Nenhuma nova
rota/tela administrativa foi ativada. Evidências em
[ISSUE_010.md](../evidencias/roadmap-04/ISSUE_010.md).

Por orientação do usuário, **layout, screenshots e comparações visuais de toda a
etapa 4 serão conferidos somente ao final da issue 015**. Até lá, executar apenas
as validações funcionais necessárias às entregas. Não houve revisão visual na 010.

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
