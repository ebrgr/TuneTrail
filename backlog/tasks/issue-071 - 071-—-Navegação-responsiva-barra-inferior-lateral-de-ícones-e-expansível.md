---
id: ISSUE-071
title: '071 — Navegação responsiva: barra inferior, lateral de ícones e expansível'
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-16T18:57:00Z'
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
- [x] #4 Capturas e aceite visual
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

author: codex
created: 2026-11-16T18:43:00Z

---

Aceite com ressalva informado pelo usuário: barra inferior aprovada e preservada. Ajustar escolha de rótulos para o painel de tema com preferência local aplicada antes do paint; padrão com rótulos/sem JS. Faixa de ícones de 58px também no desktop, com recuos coerentes. Corrigir duplicação do nome e heatmap desktop se simples; manter AC4 desmarcado e retornar a Aguardando aceite após verificação.

---

author: codex
created: 2026-11-16T18:46:00Z

---

Pendência de persistência solicitada pelo usuário: existe conf.toml com SettingsStore (internal/cfg/settings_store.go), mas tomlDocument/managedDefaults (internal/cfg/toml.go) não possuem campos de preferências de interface; Save rejeita campos não suportados. DEFAULT_THEME é configuração de ambiente, e nav-layout atual usa localStorage. Não foi inventado novo formato/arquivo nem persistência global sem contrato. Opções: (1) arquivo novo exclusivo de UI; (2) estender o conf.toml existente com campos de navegação, reutilizando validação, revisões e gravação atômica; (3) preferências por perfil no banco ou variáveis de ambiente. Recomendação: opção 2 para padrão global do servidor, com gravação administrativa autenticada e valores renderizados no html; banco caso a intenção seja preferência por perfil. Decidir escopo global x perfil e autorizar extensão do esquema antes da persistência. Restante do ajuste continua; rótulos no painel mudam só o documento atual, sem nova gravação local; nav-layout mantém por enquanto seu mecanismo anterior. Capturas limitadas a duas em 1440px/Yuu, antes e demais imagens preservados.

---

author: codex
created: 2026-11-16T18:51:00Z

---

Verificação da ressalva: controle Rótulos da navegação no painel de tema; checkbox removido da lateral; modos de ícones de 58px no tablet/desktop, tooltips e aria-label preservados; período nativo acessível em Mais. Nome do perfil sem duplicação e heatmap inteiro em 1440px. make check e ui.check verdes, 12/12 jornadas; 108 vistas de navegação com/sem JS e com/sem rótulos; 598 vistas sem overflow. Só duas imagens atualizadas em estados/desktop-yuu-{labels,icons}.png (1440px/Yuu); antes, 132 depois e demais imagens preservados. Revisão incremental própria registrada. PENDENTE: persistência global/perfil e renderização das preferências salvas pelo servidor, pois conf.toml não tem contrato de UI; opções e recomendação de extensão do conf.toml registradas no comentário anterior e README. Nova escolha de rótulos é temporária no documento; nav-layout conserva mecanismo anterior. AC4 não marcado. Sem push, merge, publish ou Done.

---

author: ebrgr
created: 2026-11-16T18:57:00Z

---

Aceite visual do usuário (2026-11-16) da barra inferior (celular) e das laterais, com a ressalva atendida: opção 'Rótulos da navegação' no painel de tema (>64rem, só ícones igual ao intermediário). Persistência das opções de tema no conf.toml NÃO incluída: promovida a nova issue por decisão do usuário. Rótulos valem só no documento; empurrar/sobrepor segue no navegador. Publicação e push dependem de autorização à parte.

---
<!-- COMMENTS:END -->
