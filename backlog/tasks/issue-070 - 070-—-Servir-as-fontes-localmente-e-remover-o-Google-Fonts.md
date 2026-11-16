---
id: ISSUE-070
title: 070 — Servir as fontes localmente e remover o Google Fonts
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-16T19:20:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-068
references:
  - backlog/evidencias/issue-061/fontes-retrospectiva.md
  - backlog/evidencias/issue-070/README.md
priority: medium
type: enhancement
ordinal: 70000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-031 e investigação backlog/evidencias/issue-061/fontes-retrospectiva.md. Usuário aprovou fontes locais, todas em assets/fonts/, lidas do disco como parte da distribuição externa; sem embed nesta implementação.

Já existem Jost Regular 400 (61.524 bytes) e League Spartan Medium 500 (57.540 bytes), ambos estáticos com OFL1.1 declarada em metadados. Inter, WOFF2 e textos OFL separados não foram encontrados. Inventariar os pesos/itálicos usados, obter arquivos/licenças necessários, registrar origem e hashes, servir via @font-face com font-display:swap e pesos corretos. Não anunciar estáticos como fontes variáveis nem remover TTF sem conferir consumidores.

Remover links/preconnects/imports de Google Fonts de todos os heads/CSS. Assegurar fontes locais em público, prévia e administração. Medir bytes totais e subconjuntos com acentos portugueses.

Corrigir também a divergência tipográfica confirmada da retrospectiva. **Decisão do usuário (14/11/2026): igualar o Koito nos pesos e estilos das fontes** — família, peso, caixa (uppercase ou não) e itálico seguem a referência Koito: vencedor 600 sem uppercase (hoje 400 uppercase por herança de h3), rótulos 400 (hoje 600), ano Jost 20/500 (hoje League Spartan 20/600). **Tamanhos não fazem parte da decisão**: título 36 versus 42 px (provável clamp), rótulos 16 versus 18 px e demais medidas só mudam após a medição da ISSUE-065 em três larguras e confirmação do usuário. Conferir regra do álbum com título longo e limitar ajustes às classes da retrospectiva. Registrar mudança visual separadamente da simples mudança de origem das fontes.

Requisitos do parecer do Advisor na ISSUE-061: (a) preferir WOFF2 publicado pelo upstream; se converter TTF para WOFF2 ou gerar subconjunto, verificar Reserved Font Name na licença OFL de cada família (Jost, League Spartan, Inter) e registrar no inventário; (b) os url() do @font-face recebem hash na URL, coordenado com a ISSUE-068 e a ISSUE-106; (c) textos de licença acompanham a imagem e o pacote, fora das rotas servidas; (d) endurecer CSP para font-src 'self' se houver CSP, e servir woff2 com o tipo MIME correto e nosniff.

Usar comparação controlada da 065 no mesmo tema, viewport e dados fictícios. Validar a fonte efetivamente renderizada, pesos e caracteres sem síntese indevida, e funcionamento com rede externa de fontes bloqueada. O aplicativo ainda pode requisitar capas e integrações: o critério é eliminar requisições externas de fontes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Nenhuma requisição externa de fontes (CSS, preconnects, imports ou arquivos) nas páginas públicas, prévia e administração.
- [x] #2 Fontes locais carregam com rede externa bloqueada; pesos/estilos, acentos e licenças documentados e verificados.
- [x] #3 Inventário de arquivos, origem, hash e tamanho total registrado; TTF antigos só removidos após conferir consumidores.
- [x] #4 Retrospectiva com pesos e estilos (família, peso, caixa, itálico) iguais à referência Koito, sem regressão global dos headings; tamanhos alterados só após a medição em três larguras e confirmação do usuário
- [x] #5 Licença OFL verificada por família (Reserved Font Name), url() das fontes com hash e licenças fora das rotas servidas
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: Codex
created: 2026-11-14T14:22:00Z

---

Refinamento autorizado: fontes locais em assets/fonts/, sem embed. Investigação confirmou diferenças de peso/uppercase/tamanho nas retrospectivas, além da dependência de Google Fonts; duas fontes TTF atuais não cobrem todos os estilos. Detalhes no doc-031 e evidência da ISSUE-061.

---

author: claude
created: 2026-11-14T15:00:00Z

---

Decisão do usuário em 14/11/2026 (via ISSUE-061, item 2 do parecer do Advisor): igualar o Koito nos pesos e estilos das fontes; tamanhos dependem da medição e de confirmação. Requisitos do Advisor (itens 4, 6, 11) registrados na descrição.

---

author: ebrgr
created: 2026-11-16T19:00:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-070` na branch `feat/issue-070-servir-as-fontes-localmente-e` criada a partir de `main`.

---

author: advisor
created: 2026-11-16T19:02:00Z

---

Parecer do Advisor (Modo B): APROVADO para execução com validação de licenças/RFN e referência Koito antes de decisões dependentes. Plano ai-memory 01a1e5bc-e22f-7cb0-8fae-4b1d44e0b6b4 (plans/issue-070-fontes-locais.md). Head compartilhado concentra Google Fonts; catálogo já oferece hash/MIME/nosniff. Criar faces e catálogo de fontes, preservar TTF, corrigir só tipografia da retrospectiva sem tamanhos, medir 390/768/1440 e verificar fonte real/rede bloqueada; evidências e checks antes do commit. AC4 aguarda usuário; sem publicação.

---

author: codex
created: 2026-11-16T19:07:00Z

---

Implementação: fontes WOFF2 upstream locais (951.648 bytes), OFL/RFN e cobertura documentados; TTF preservados, URLs com hash e head sem Google em público/prévia/admin. Tipografia rewind limitada a família/peso/caixa, tamanhos preservados e medidos em 390/768/1440. Três capturas Yuu; CDP confirma fontes customizadas/acentos com rede externa bloqueada. AC4 segue desmarcado até confirmação do usuário. make check (12/12 e2e) e ui.check verdes.

---

author: codex
created: 2026-11-16T19:08:00Z

---

Inventário final computado nas onze páginas identificou League Spartan 500 em mídia/admin; incluído WOFF2 Medium upstream, sem síntese. Total final WOFF2: 983.452 bytes (corrige o total preliminar do comentário anterior). make check e ui.check repetidos e verdes, 12/12 e2e. Revisão própria/deslop APROVADO em backlog/evidencias/issue-070/revisao.md.

---

author: codex
created: 2026-11-16T19:09:00Z

---

Verificação: fontes locais e acentos confirmados por CDP com rede externa bloqueada em público/prévia/admin; MIME/nosniff e OFL privados verificados; inventário final 983.452 bytes WOFF2 e três capturas Yuu 1440px registrados. make check e ui.check verdes, e2e 12/12, revisão própria APROVADO. Medidas em 390/768/1440 preservam tamanhos. AC4 permanece desmarcado até confirmação do usuário. Sem dúvidas pendentes, push, merge, publish ou Done.

---

author: ebrgr
created: 2026-11-16T19:20:00Z

---

Aceite do usuário (2026-11-16) após ver as capturas antes/depois da retrospectiva e a Home. Decisão: manter os tamanhos atuais (título 42/36, rótulo 18 em 1440px); aceita apenas pesos, caixa e itálico alinhados ao Koito. Qualquer igualar de tamanho (título 36, rótulo 16, regra de título longo >28 caracteres) fica fora desta issue. Publicação e push dependem de autorização à parte.

---
<!-- COMMENTS:END -->
