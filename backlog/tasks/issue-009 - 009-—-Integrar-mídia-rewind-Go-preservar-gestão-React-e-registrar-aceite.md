---
id: ISSUE-009
title: 009 — Integrar mídia/rewind Go, preservar gestão React e registrar aceite
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-0
dependencies: []
type: feature
ordinal: 75000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/009-integracao-midia-retrospectiva-solved.md` — numeração local 009. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida — aceite confirmado pelo usuário em 05/11/2026**. Issue local, sem publicação.
Plano: [roadmap 3](<../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>). Depende de 006, 007 e 008.

## Trabalho

- Estender em conjunto dispatcher Go, ponte React e manifesto de casos de rotas
  para artista/álbum/faixa/rewind. Formatos conhecidos com ID inválido devem
  alcançar a validação HTML, sem cair na SPA por engano.
- Manter `KOITO_WEB_UI=react` padrão, opção `go`, API, imagens, assets e admin.
  Preservar URLs legadas, seleção com `next` interno e troca de perfil.
- Criar aliases transitórios React propostos `/{slug}/manage/{artist|album|track}/{id}`,
  sob ProfileLayout, reutilizando loaders/componentes atuais. São rotas novas,
  não caminhos já disponíveis. Go oferece acesso à gestão e retorno público.
- Preservar login e retorno interno, edição/aliases, duração/MBID/artistas,
  substituição de imagem, mesclagem nos dois sentidos, exclusão de catálogo e
  adição de escuta da faixa no perfil selecionado. Reutilizar sessão, confirmação
  e proteção de origem existentes; chave de player continua sem poder de admin.
- Corrigir extração frágil de identidade no upload arrastado: passar tipo/ID
  explicitamente, incluindo álbum real da faixa. Não usar último segmento da URL
  como identidade do objeto a alterar. Conferir callbacks após trocar de perfil.
- Manter exclusão/adição de escuta por perfil e catálogo compartilhado. Após ação,
  revalidar dados e preservar destino correto, inclusive merge reverso/exclusão.
- Ampliar validação descartável de `scripts/validate_webui.py` e produzir manifesto,
  pares React/Go e relatório em `backlog/evidencias/roadmap-03/`.
- Atualizar documentação de configuração/edição e resumo do roadmap após testes.
  Não remover a SPA nem publicar nesta issue.

## Aceite

Go → artista → álbum → faixa → histórico/ranking filtrado → rewind; Go → gestão
React → Go; acesso direto, login, recarga, voltar/avançar e duas abas/perfis.
Negativas de sessão/chave/origem e testes de escrita somente em bancos descartáveis.
Zero perda de acesso às ações existentes durante a transição para a etapa 4.

Pares desktop/celular, escuro/claro, mesmo SQLite v3/fuso/viewport; cabeçalhos,
gráficos e regiões abaixo da dobra, rewind mensal/anual, tempos, vazios/erros e
retorno da gestão. Testes Go, cliente, builds e regressão multiusuário adequados
passam antes de habilitar as novas rotas no modo Go. Nenhuma alteração no banco
em uso ou publicação faz parte desta proposta.

## Revisão visual consolidada

Por orientação mais recente do usuário, conferir layout e screenshots do conjunto
**006 + 007 + 008 + 009 ao final desta issue**. Esta decisão substitui o agendamento
anterior ao final da 008. As issues anteriores mantêm seus marcos de implementação
funcional/testes, sem antecipar o aceite visual.

## Implementação e evidências

A integração e os aliases foram implementados. Testes, comparações, revisão do
Advisor e diferenças observadas estão em [ISSUE_009.md](../evidencias/roadmap-03/ISSUE_009.md).
A rota Go está habilitada no código para `KOITO_WEB_UI=go`, sem alterar ou reiniciar
a instância em uso. Permanecem pendentes a reprodução manual de arrastar imagem
e a validação completa de navegação por teclado e dos fluxos de edição de todos os
tipos de mídia; a issue fica aberta até completar esses itens.

## Encerramento

O usuário confirmou em 05/11/2026 que todas as issues desta etapa estão completas.
Este aceite encerra a issue. Os registros anteriores permanecem como histórico
das verificações executadas e das verificações manuais que não foram realizadas;
o encerramento não as apresenta como testes executados.
<!-- SECTION:DESCRIPTION:END -->
