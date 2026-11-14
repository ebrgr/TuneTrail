---
id: ISSUE-013
title: 013 — Migrar edição de mídia, imagens e ações de escutas para Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
type: feature
ordinal: 79000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/013-edicao-midia-escutas-go-solved.md` — numeração local 013. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](<../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>). Depende de 010.

## Trabalho

- Preservar `/{slug}/manage/{artist|album|track}/{id}` com forms Go de aliases,
  principal, MBID, compilação e relações realmente oferecidas pelo cliente atual.
  Não apresentar edição de duração como paridade existente.
- Reutilizar serviços com identidade explícita e aviso de catálogo compartilhado;
  validar IDs/MBIDs/aliases e tratar 404/409/erro real sem sucesso falso.
- Migrar capas por upload/URL e sugestão existente, com alternativa ao arraste;
  em faixa usar o álbum real, inclusive se ID da faixa difere do ID do álbum.
- Limitar arquivo/rede, contexto/timeout/redirect e conteúdo; compensar arquivo
  novo se banco falha e apagar anterior somente após commit e referência global.
- Implementar adição e exclusão de escuta pessoal, confirmação, fuso/DST e retorno
  com filtros/paginação válidos; compartilhar cálculo/serviço com API e live.js.
- Preservar entrada digitada em erros e evitar cliques duplos; alternativa sem JS,
  sessão/CSRF, 303 após sucesso e formulários sem credenciais em query.

## Aceite

Alias principal/secundário, vazio/Unicode, MBID inválido, compilação, múltiplos
artistas e relações ausentes. Caso slug `manage`, mudança de perfil e duas abas.

Capa com IDs distintos, imagens compartilhadas, falha de rede/processamento/commit/
limpeza, arquivo inválido/grande, upload e arraste reais no navegador. Escuta manual
no fuso efetivo e ambiguidades de DST; apagar um registro não altera pessoa com
mesma faixa/timestamp. Testar sessão expirada e falha de escrita sem inventar sucesso.
Capturar desktop/celular/teclado; completar casos não executados na 009 sem reabri-la.

## Andamento — 05/11/2026

A primeira parte prepara o editor Go isolado, sem ativá-lo no dispatcher. Há
formulários para aliases, MBID/compilação, artistas da faixa, artista principal,
capa e adição manual de escuta. Os serviços verificam sessão, escopo de perfil
e identidade. A API JSON de aliases, imagens e escutas reutiliza serviços ou
invariantes compartilhadas. As mudanças de catálogo são globais.

Corrigidas a transação de adição de artistas à faixa, a seleção de relações
inexistentes e a remoção que acionava uma limpeza global de faixas sem escutas.
A troca de capa limita upload/rede/imagem, permite até três redirects validados
e preserva imagem anterior para a coleta global; capa de faixa revalida a relação
com o álbum no próprio UPDATE. Horário local trata DST inexistente/ambíguo com
offset explícito; offsets históricos com segundos não são suportados.

**Pendente para concluir a issue:** concluir a prevenção de cliques duplicados,
integração dos caminhos de histórico com o editor sem JavaScript, testes de
paginação/retorno e demais jornadas adversariais. O editor mantém campos em erros,
inclui busca autenticada de artistas, sugestão CAA e confirmação de exclusão das
25 escutas recentes, com seleção de fuso e retorno à consulta da gestão.
A integração, arraste em navegador e comparação visual ficam para a 015.
Detalhes e evidências: [relatório](../evidencias/roadmap-04/ISSUE_013.md).

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
