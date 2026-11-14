---
id: ISSUE-106
title: 106 — Cache dos arquivos estáticos e do botão Voltar
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:55:00Z'
updated_date: '2026-11-14T22:40:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-066
references:
  - backlog/evidencias/ui-responsiva/comparacao-koito-react.md
  - backlog/evidencias/issue-106/http-base-0.11.json
  - backlog/evidencias/issue-106/chrome-base-0.11.json
  - backlog/evidencias/issue-106/advisor-final.md
  - backlog/evidencias/issue-106/implementacao.md
documentation:
  - doc-033
priority: high
type: enhancement
ordinal: 106000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: comparação com o Koito React e estudo doc-033, preparado após publicar a ISSUE-066 na versão 0.11. A implementação usa assets externos e o snapshot dessa tarefa; não usa arquivos embutidos nem exige migração de banco.

Medição fictícia na 0.11: /web-assets sem Cache-Control/ETag/Last-Modified; o Chrome 157.0.8103.51 transferiu novamente seis CSS/JS/JSON locais na segunda navegação (48.737 bytes de conteúdo). Voltar nessa sessão não restaurou bfcache. No-store pode permitir bfcache em Chrome atual sob condições específicas; cabeçalhos não bastam como prova. Firefox e proxy HTTPS também foram verificados na implementação; relatórios ligados à tarefa.

Preparar catálogo imutável por bytes finais, SHA-256 por arquivo, temas pré-gerados e helper asset para público/admin/prévia. URLs de dependências CSS também recebem hash; ui.js recebe data-themes-url. URL versionada correta ganha immutable; alias sem query revalida com ETag; versão desconhecida/erro nunca recebe bytes novos imutáveis. Política adotada na execução do doc-033: manter somente a geração atual e permitir revalidação dos aliases legados /public e manifest.

HTML público 200 usa private, no-cache e Vary apropriado. Erros, administração, login, ações e formulários/CSRF continuam no-store. Corrigir o AbortController do live.js, abortado em pagehide e reutilizado ao voltar; revalidar sessão/geração e evitar controles antigos. A 068 reutiliza o catálogo depois da concatenação; a 070 adiciona as fontes locais; 075/109 reutilizam o lifecycle e a consulta de papel sem duplicá-los.

Implementação concluída e validada, com parecer final APROVADO do Reviewer. Evidências em backlog/evidencias/issue-106/implementacao.md. Aceita pelo usuário em 14/11/2026; publicação pendente de autorização.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Todos os arquivos de /web-assets com hash na URL e cache imutável; mudança de conteúdo muda a URL
- [x] #2 Segunda navegação não baixa CSS nem JS de novo (verificado nas ferramentas do navegador)
- [x] #3 Páginas públicas com private, no-cache; administração continua no-store
- [x] #4 Voltar de uma busca restaura a página do bfcache no Chrome e no Firefox
- [x] #5 Hash calculado sobre bytes finais; CSS inclui hashes das folhas e themes.css é determinístico; asset ausente/privado/não literal falha no startup.
- [x] #6 GET/HEAD/304/Range corretos com ETag; hash antigo/errado/duplicado, query extra, arquivo ausente e erros 4xx/5xx não recebem immutable nem bytes atuais sob hash antigo.
- [x] #7 URLs de fetch JS, prévia, favicon, aliases /public e site.webmanifest seguem a política documentada; templates/licenças continuam privados.
- [x] #8 Voltar recria controlador e revalida sessão, sem listeners/células duplicados ou botões travados; logout/expiração e falhas de rede testados em ciclos repetidos; administração restaurada revalida e regenera formulários.
- [x] #9 Geração anterior/reinício e seus limites documentados e exercitados; não prometer continuidade de abas ou recuperação por JS quando o script antigo não estiver disponível.
- [x] #10 Referência de renderização atualizada de forma revisável para as novas URLs/CSS/JS, preservando as bases históricas; make check sem cache e teste atrás de proxy HTTPS aprovados.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirmar as propostas do doc-033 (geração atual apenas, aliases legados revalidáveis, HEAD somente em pageshow.persisted).
2. Implementar catálogo imutável e hashes por bytes finais sobre o snapshot da 066; temas pré-gerados e CSS→folhas com URLs relativas versionadas, sem DAG/imports nesta etapa.
3. Validar chamadas asset literais no startup e integrar os três shells e data-themes-url.
4. Handler HTTP compartilhado, condicionais/HEAD/Range, política de erros e aliases. HTML público por status, Vary e geração da UI; manter proteção administrativa.
5. Corrigir lifecycle de live.js/admin.js: controladores novos, revalidação da sessão, botões sem duplicação/travamento e recarga administrativa ao restaurar.
6. Testes Go/e2e sem cache, medições Chrome e Firefox reais, proxy HTTPS e comparação de renderização com base revisada da 106.
7. Reviewer, Aguardando aceite e aviso ao usuário; publicação em versão própria somente após aceite/autorização.
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T15:00:00Z

---

Requisito do parecer do Advisor na ISSUE-061 (item 4): o hash na URL vale também para arquivos referenciados por url() no CSS (fontes woff2, ícones), não só para os <link> e <script> dos templates. Com a decisão da 061 (assets servidos do disco), implementar sobre a estrutura da ISSUE-066.

---

author: claude
created: 2026-11-14T15:10:00Z

---

Nova ordem da Fase 1 aprovada pelo usuário em 14/11/2026: os assets passam a ser servidos do disco pela ISSUE-066 e os templates são reformatados pela ISSUE-067, então as tarefas de fluidez vêm depois delas para evitar retrabalho e conflitos. Esta tarefa calcula hashes sobre a estrutura nova da 066.

---

author: Codex
created: 2026-11-14T20:25:00Z

---

Estudo solicitado pelo usuário concluído em doc-033, com leitura da versão 0.11 e medição em servidor/banco fictícios. Status Planejada porque existe plano de resolução; nenhum critério de runtime foi marcado. Ordem vigente: 106 sobre 066 publicada, antes de completar a 067. Parecer inicial do Advisor incorporado; reavaliação documental final em andamento. Propostas ainda não são decisões aceitas. Sem mudança de produto, publicação ou push.

---

author: Codex
created: 2026-11-14T20:28:00Z

---

Reavaliação do Advisor Claude Opus: APROVADO, sem lacunas bloqueantes; parecer em backlog/evidencias/issue-106/advisor-final.md. Recomendações documentais incorporadas ao doc-033: testar também histórico sem bfcache, não presumir Vary como garantia de histórico, tratar falha de paletas e identificar a versão nas evidências reproduzíveis. Estudo pronto; propostas de retenção atual/aliases/HEAD aguardam confirmação antes da implementação. Nenhuma alteração de runtime ou AC marcado.

---

author: ebrgr
created: 2026-11-14T20:33:00Z

---

Início: branch `feat/issue-106-cache-dos-arquivos-estaticos-e` criada a partir de `main`.

---

author: Codex
created: 2026-11-14T20:33:00Z

---

Execução autorizada pelo usuário após o doc-033: adotar as propostas recomendadas (somente geração atual com limite de abas antigas documentado; aliases/manifest legados revalidáveis; HEAD de contexto apenas em pageshow.persisted). Parecer técnico do estudo aprovado. Implementação funcional na branch da tarefa; sem autorização de publicação ou push nesta etapa.

---

author: Codex
created: 2026-11-14T21:51:00Z

---

Implementação concluída e Reviewer Claude Opus APROVADO. Catálogo por bytes finais, URLs versionadas, condicionais/Range, políticas públicas/admin e restauração segura do histórico. make check 12/12 jornadas, make api.race, Chrome/Firefox/HTTPS aprovados. Matriz visual: 276 textos/geometrias iguais; 249 capturas idênticas, 27 com diferenças de rasterização documentadas. Referência textual atualizada passa. Evidências e limites em implementacao.md. Sem aceite, publicação, merge ou push nesta etapa.

---

author: ebrgr
created: 2026-11-14T21:51:00Z

---

Verificação: Reviewer Claude Opus APROVADO; make check, api.race, Chrome/Firefox e proxy HTTPS aprovados; cache sem novas transferências CSS/JS/JSON e bfcache real; 276 comparações com texto/geometria preservados. Evidências em backlog/evidencias/issue-106/implementacao.md. Falta aceite explícito do usuário para publicar.

---

author: claude
created: 2026-11-15T01:30:00Z

---

Sugestões finais do Reviewer aplicadas e referência regenerada para os novos hashes; make check (12/12) e make api.race aprovados.

Aceite: aprovada pelo usuário em 14/11/2026 ("aceito as mudanças"). Publicação local autorizada na mensagem: "commite e publique". Push não solicitado.

---

author: ebrgr
created: 2026-11-14T22:40:00Z

---

Publicação: versão 0.12 (tag `v0.12`), branch `feat/issue-106-cache-dos-arquivos-estaticos-e` integrada ao `main` por fast-forward.

---
<!-- COMMENTS:END -->
