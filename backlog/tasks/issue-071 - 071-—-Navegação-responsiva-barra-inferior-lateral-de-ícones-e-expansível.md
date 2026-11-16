---
id: ISSUE-071
title: '071 — Navegação responsiva: barra inferior, lateral de ícones e expansível'
status: Aguardando aceite
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-16T18:18:00Z'
labels:
  - ui
milestone: m-10
dependencies:
  - ISSUE-069
references:
  - backlog/evidencias/issue-071/README.md
  - backlog/evidencias/issue-071/revisao.md
priority: high
type: feature
ordinal: 71000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §3.3. Requer decisão do usuário sobre a barra inferior no celular (doc-010 §10, item 2).

Passos:
1. < 40rem: barra inferior fixa com 4–5 destinos (Início, Rankings, Histórico, Busca, Mais) e `env(safe-area-inset-bottom)`; cabeçalho de perfil compacto.
2. 40–64rem: barra lateral de ícones com rótulo acessível.
3. > 64rem: barra lateral expansível com rótulos, perfil no topo e seletor de período persistente.
4. Funciona sem JavaScript; foco e leitura por leitor de tela verificados.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Decisão do usuário sobre a barra inferior registrada
- [x] #2 Três padrões de navegação conforme a largura, sem JavaScript
- [x] #3 Alvos de toque de pelo menos 44×44px
- [ ] #4 Capturas e aceite visual
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T01:12:00Z

---

Coordenar com a ISSUE-110: no celular, o modal de tema e idioma (Palette) e o submenu de administração (Settings2) ficam no destino 'Mais' da barra inferior.

---

author: ebrgr
created: 2026-11-16T17:56:00Z

---

Decisão do usuário (2026-11-16): ACEITA a barra de navegação inferior fixa no celular (<40rem), mudando a identidade herdada do Koito (doc-010 §10, item 2). Destino 'Mais' reúne tema/idioma e configurações/admin (coordenar com ISSUE-110).

---

author: ebrgr
created: 2026-11-16T17:56:00Z

---

Início: worktree Orca `/Users/evertonberger/orca/workspaces/TuneTrail/issue-071` na branch `feat/issue-071-navegacao-responsiva-barra-inferior-lateral` criada a partir de `main`.

---

author: advisor
created: 2026-11-16T17:57:00Z

---

Advisor Modo B: APROVADO. Plano em ai-memory plans/issue-071-navegacao.md. Rankings leva ao ranking existente de artistas (/perfil/chart/top/artists); Histórico a /perfil/listens. Mais usa details nativo e reúne Palette/Settings2; segunda barra preserva empurrar/sobrepor no desktop e abre acima da barra no celular. Período/fuso seguem em URLs, sem cookies; seletor disponível na lateral expandida. Rótulos visíveis substituem tooltips em mobile/desktop; aria-label na faixa de ícones. Capturas antes/depois, reserva inferior incluindo footer/safe-area, verificação sem JS/Tab/foco/semântica/44px/overflow, changelog e checks; AC4 permanece pendente.

---

author: codex
created: 2026-11-16T18:16:00Z

---

Implementação: navegação inferior nativa em <40rem; ícones em 40–64rem; rótulos recolhíveis, perfil e períodos na lateral >=64rem. Rankings abre /perfil/chart/top/artists; Histórico abre /perfil/listens; Mais reúne perfil, Palette, Settings2 e Retrospectiva. Período/fuso preservados por URL inclusive na Busca, sem cookie. Segunda barra mantém empurrar/sobrepor e se posiciona acima da barra móvel. Revisão própria sem delegação; AC4 permanece desmarcado. Acessibilidade validada por teclado e árvore real do Chromium; VoiceOver manual pendente.

---

author: codex
created: 2026-11-16T18:18:00Z

---

Verificação: make check e make ui.check verdes; 12/12 jornadas end-to-end; check_web_style.py e diff sem erros; 72 vistas de navegação com/sem JavaScript verificam foco, Tab, nomes e estado expandido na árvore de acessibilidade, alvos >=44px, footer e segunda barra (empurrar/sobrepor e altura desktop). 598 vistas responsivas sem rolagem horizontal. Capturas Yuu/Pearl em 320/360/390/768/1024/1440: 132 antes e 132 depois em backlog/evidencias/issue-071/. Grafo atualizado e revisão própria aprovada. AC2/AC3 verificados; AC4 não marcado, aguardando aceite visual. VoiceOver manual não executado. Sem push, merge, publish ou Done.

---

author: codex
created: 2026-11-16T18:18:00Z

---

Precisão da evidência: a checagem de whitespace passou no diff de fontes; reference/ é um espelho de bytes exportados e preserva espaços dos templates e CRLF dos SVGs existentes. Essa referência é comparada sem normalização por ui.reference.check, que passou. Registro detalhado em backlog/evidencias/issue-071/README.md.

---
<!-- COMMENTS:END -->
