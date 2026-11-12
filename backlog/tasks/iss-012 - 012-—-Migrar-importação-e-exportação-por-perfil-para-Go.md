---
id: ISS-012
title: 012 — Migrar importação e exportação por perfil para Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
references:
  - docs-dev/issues/012-importacao-exportacao-go-solved.md
type: feature
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/012-importacao-exportacao-go-solved.md](../../docs-dev/issues/012-importacao-exportacao-go-solved.md) — numeração local 012.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](../../docs-dev/ROADMAP_04_ADMINISTRACAO.md). Depende de 010 e da seleção da 011.

## Trabalho

- Formulário multipart com formato e destino (nome/slug/ID) confirmados; reutilizar
  os cinco importadores e o perfil da rota, sem confiar no nome dentro do arquivo.
- Manter execução síncrona e contexto da requisição; não anunciar job persistente,
  progresso percentual, cancelamento administrativo ou rollback total inexistentes.
- Tornar esperas/loops canceláveis e definir limites do upload de 100 MiB incluindo
  multipart, JSON em memória, ZIP descompactado, membros/linhas e processamento.
- Explicar e reportar prefixo já salvo em falha/cancelamento enquanto o cliente
  estiver conectado; após desconexão não prometer relatório recuperável. Diferenciar contagens
  efetivamente importadas, duplicadas, ignoradas e erros, sem repetição automática.
- Preservar janela de importação, arquivos nomeados pelo servidor, limpeza de
  temporários e arquivamento no sucesso; evitar reimportação involuntária no boot.
- Usar export pessoal por streaming/paginação existente, com no-store, attachment,
  nome escapado e erros de escrita/cancelamento propagados; definir consistência
  durante ingestão concorrente e não anexar erro JSON a download já iniciado.
- Evitar ciclo de dependências ao extrair serviços: importer não deve passar a
  depender de `internal/admin` que o importa; tipos ListenBrainz em local apropriado.

## Aceite

Fixtures locais de Koito, Spotify, Maloja, LastFM e ListenBrainz; nomes divergentes
não mudam o destino. Arquivo inválido, JSON grande, ZIP expandido/linha longa,
limites e upload incompleto, falha intermediária, cancelamento e repetição de
arquivo. Reinício não retoma upload fracassado silenciosamente.

Round-trip em outro perfil, >1000 registros, timestamps iguais com faixas distintas,
Unicode/escaping e writer que falha. Comprovar isolamento, autorização, CSRF antes
da escrita e resultado parcial honesto em SQLite v3 descartável. Não usar dados
ou tokens pessoais nem testar formatos consultando serviços externos.

## Andamento — 05/11/2026

O handler Go preparado oferece importação e download por perfil, mas continua
isolado do dispatcher de produção até a integração da 015. A API existente
continua disponível. Os resultados distinguem inserções, duplicatas, ignorados
e erro fatal; metadados compartilhados podem ter sido criados antes de uma falha.

O orçamento é 100 MiB para o envelope HTTP, 32 MiB para JSON, 500 mil tokens
JSON no preflight e 250 mil itens/linhas, o que ocorrer primeiro. ZIP tem até
1000 membros, 256 MiB expandidos e linhas de até 1 MiB. O limite estrutural
pode rejeitar um arquivo válido bem antes de 250 mil escutas. Importação é
síncrona, sem retomada automática; o relatório parcial só existe enquanto a
resposta chegar ao cliente.

Exportação percorre páginas em leitura dinâmica, sem snapshot transacional:
escutas retroativas podem ficar fora do cursor, exclusões e alterações de
metadados podem ocorrer entre páginas, e ingestão contínua pode prolongá-la.
Consulte [relatório e validações](../../docs-dev/comparisons/roadmap-04/ISSUE_012.md).

Ainda são necessárias regressões específicas de desconexão em trabalho remoto,
falha de quarentena, reinício após encerramento abrupto e round-trip concorrente
em outro perfil; o aceite geral foi concedido pelo usuário no encerramento de 001–015.

## Integração na 015 — 05/11/2026

A administração foi ligada ao modo Go; os contratos e jornadas são verificados
pelo servidor real e pelas capturas comparativas. As pendências de integração
descritas no andamento anterior são históricas. Consulte
[entrega e limites da 015](../../docs-dev/comparisons/roadmap-04/ISSUE_015.md).

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário ao solicitar o encerramento
de todas as issues abertas até a 015. Entrega, testes e limites conhecidos
permanecem registrados; validação transversal será estudada no roadmap 5.
<!-- SECTION:DESCRIPTION:END -->
