---
id: ISSUE-066
title: 066 — Reunir arquivos web em assets/ com leitura do disco
status: Aberta
assignee: []
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T15:00:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-065
documentation:
  - doc-031
priority: high
type: enhancement
ordinal: 66000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 atualizado por decisão do usuário em 14/11/2026. Todos os arquivos web ficam em assets/ na raiz, incluindo HTML em assets/templates/{public,admin,partials}, CSS, JS, ícones, fontes, temas e arquivos públicos. Leitura do disco em produção e desenvolvimento; embed somente como opção futura.

Implementar TUNETRAIL_WEB_DIR como root assets explícito; sem configuração usar assets ao lado do executável real (symlinks resolvidos), independente do cwd. Root explícito inválido falha sem fallback. Desenvolvimento e testes passam root do checkout/fixture explicitamente.

No startup validar templates e assets obrigatórios e construir snapshot consistente em memória (templates, bytes, temas e hashes); produção serve esse snapshot. Desenvolvimento pode recarregar por snapshot completo validado, com troca atômica e erros visíveis. Não servir HTML-fonte ou todo o root como estáticos; publicar somente subpastas autorizadas, sem directory listing.

Ajustar Dockerfile na imagem final, pacote binário+assets, Makefile, scripts, configuração e documentação. Distribuir assets e binário da mesma versão em diretório imutável e reiniciar para trocar geração. Coordenar retenção de URLs com hash antigas com ISSUE-106. Preparar fronteira fs.FS para uma futura opção embed, sem implementá-la agora.

Esta mudança afeta distribuição e runtime: é funcional e publica versão própria, mesmo preservando os bytes de HTML/CSS. Depende da referência da ISSUE-065.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 HTML/CSS equivalentes à referência da 065; URLs /web-assets preservadas.
- [ ] #2 Distribuição binário+assets e imagem Docker final sobem sem depender do cwd; binário isolado falha com orientação clara se assets estiver ausente.
- [ ] #3 Root explícito inválido, template inválido e arquivo obrigatório ausente recusam partida; templates e brand não são expostos como estáticos.
- [ ] #4 Produção usa snapshot coerente de templates/assets/hashes; recarga de desenvolvimento só troca uma geração completa válida.
- [ ] #5 Execução com symlink, pacotes da mesma versão, caminhos de desenvolvimento e contrato com cache da 106 documentados e testados.
- [ ] #6 Arquivos servidos por manifesto de extensões (.css, .js, .svg, .woff2, .json, imagens), lidos só dentro do root (symlinks que escapem são recusados); a busca nas requisições usa a chave do snapshot, nunca o caminho do disco; licenças fora das rotas, mas presentes na imagem e no pacote
- [ ] #7 Root de assets só leitura para o usuário que executa o servidor (dono root na imagem, coordenado com a ISSUE-102), com checagem no build ou teste
- [ ] #8 Renderização da referência da ISSUE-065 como gate de CI (não só o parse); conjuntos de templates público e administração separados, sem colisão de nomes
- [ ] #9 Contrato de localização de langs/ igual ao de assets (executável real, sem cwd, explícito inválido falha) ou contrato próprio documentado
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-14T14:22:00Z

---

Refinamento: usuário escolheu tudo em assets/ e leitura do disco; substitui templates/ e assets/ separados e embed imediato. Tipo alterado para enhancement pois distribuição e execução mudam e exigem publicação de versão. Especificação vigente: doc-031.

---

author: claude
created: 2026-11-14T15:00:00Z

---

Requisitos do parecer do Advisor na ISSUE-061 (itens 5, 7, 8, 9 e 10) acrescentados como critérios. Pendências de verificação no código: se o Dockerfile copia assets/ só no build ou também na imagem final; tipos MIME e nosniff; comportamento atual de cache em /web-assets para não regredir antes da 106. Nota de registro (item 11): o nome do arquivo desta tarefa ainda reflete o título antigo; corrigir na próxima edição de título.

---
<!-- COMMENTS:END -->
