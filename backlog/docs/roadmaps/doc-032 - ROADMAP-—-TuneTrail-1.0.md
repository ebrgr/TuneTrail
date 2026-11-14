---
id: doc-032
title: ROADMAP — TuneTrail 1.0
type: specification
created_date: '2026-11-14T00:39:00Z'
updated_date: '2026-11-14T03:30:00Z'
tags:
  - roadmap
---

# ROADMAP — TuneTrail 1.0

**Criado em:** 13/11/2026, na versão **0.8** · **Substitui:** a lista de tarefas do [ROADMAP antigo](<doc-001 - ROADMAP.md>) para os itens 2, 4, 5, 7 e 8 e o plano de execução da [avaliação TuneTrail 1.0](<doc-010 - TuneTrail-1.0-—-avaliação-do-projeto-e-proposta-de-evolução.md>) §9.
**Fontes:** ROADMAP (doc-001) itens 2, 4, 5, 7 e 8 · avaliação TuneTrail 1.0 (doc-010) · [estudo do CSS e templates](<../estudos/doc-031 - Estudo-—-reorganização-do-CSS-e-dos-templates-HTML.md>) (doc-031) · [revisão de pendências](<../estudos/doc-017 - Revisão-das-issues-e-pendências-remanescentes.md>) (doc-017).

Cada linha abaixo é uma tarefa do Backlog.md com descrição, passos, critérios de aceite e dependências. Este documento é o índice; o detalhe vive na tarefa. Consultas: `backlog task list -m m-10 --plain` (por fase), `backlog task list --ready --plain` (sem dependências pendentes), `backlog board`.

## Como trabalhar este roadmap

- Ciclo de cada tarefa conforme a skill `issue-first`: `make issue.start` → commits `ISSUE-NNN: ` → (opcional) `make issue.review` → aceite do usuário → `make issue.publish`. O agente sempre avisa o usuário quando coloca uma tarefa em Aguardando aceite.
- Versão conforme a [regra de versionamento](#versionamento): mudanças no produto publicam uma versão cada; estudos, documentação, testes e ferramentas entram no `main` sem versão (coluna **Versão**).
- Tarefas com concorrência, mensageria ou criptografia (079, 081, 082, 084, 086, 090, 100) exigem parecer do Advisor antes da implementação (`AGENTS.md`).
- Tarefas da UI usam a referência da ISSUE-065: as que não mudam o visual provam equivalência por comparação; as que mudam pedem capturas antes/depois e aceite visual.

## Visão geral

| Fase | Marco | Objetivo | Tarefas | Versões | Versão ao final* |
| --- | --- | --- | ---: | ---: | --- |
| 0 · Base | `m-9` | Registros reconciliados, suíte e2e no CI, recuperação testada, 049 corrigida | 4 | 1 | **concluída** (0.9) |
| 1 · UI responsiva | `m-10` | Interações rápidas, ações de mídia como no Koito, sessão de admin visível, barra lateral reorganizada, CSS e templates legíveis, design mobile-first e acessibilidade | 21 | 14 | 0.24 |
| 2 · Ingestão confiável | `m-11` | Nenhuma escuta perdida: eventos, deduplicação, outbox, saúde, SSE | 6 | 5 | 0.29 |
| 3 · Integrações | `m-12` | ROADMAP item 4: fontes nativas e compatibilidade | 7 | 6 | 0.35 |
| 4 · Insights | `m-13` | ROADMAP item 7 e novas análises | 7 | 6 | 0.41 |
| 5 · Compartilhar | `m-14` | ROADMAP itens 2 e 5: cards sociais e perfis públicos/privados | 4 | 3 | 0.44 |
| 6 · Operação | `m-15` | Container sem root, backup, observabilidade, logs, site | 5 | 4 | 0.48 |
| Marco 1.0 | `m-8` | Candidatura ao TuneTrail 1.0 | 1 | — | 1.0 (decisão do usuário) |

\* Projeção pela [regra de versionamento](#versionamento), a partir da 0.9, na ordem das fases e contando a ISSUE-111 (marco Identidade, 1 versão) antes da Fase 1.

**Paralelismo:** a Fase 0 vem primeiro. Depois, as fases 1 e 2 podem correr juntas (camadas diferentes). A 3 depende da 2. A 4 começa pela ISSUE-092 assim que a fase 0 terminar; a 097 depende da 2. A 5 depende de partes da 1 (070, 074). A 6 pode começar a qualquer momento após 063/064.

## Plano detalhado — Fase 0 · Base (concluída em 13/11/2026)

Ordem sugerida, uma tarefa por vez:

1. **ISSUE-062** — reconciliar registros e higiene. Desbloqueia a 064 e deixa o histórico confiável antes do trabalho grande.
2. **ISSUE-049** — corrigir a associação residual de artista em álbum. Bug de catálogo de prioridade alta; a 096 depende dela. Exige estudo de procedência das relações antes de mudar o esquema.
3. **ISSUE-063** — suíte e2e única no `make check` e no CI no lugar dos 37 scripts por issue. Todas as fases seguintes passam a ter rede de segurança.
4. **ISSUE-064** — testes de recuperação de migração, importação e restauração. Pré-requisito da migração v4 (079/080) e do backup pela UI (103).

**Saída da fase:** `make check` cobre as jornadas principais; nenhuma falha conhecida de catálogo; recuperação de banco testada.

## Plano detalhado — Fase 1 · UI responsiva

Quatro blocos, nesta ordem:

**Bloco 0 — responsividade imediata (antecipado após a [comparação com o Koito React](<../../evidencias/ui-responsiva/comparacao-koito-react.md>))**

O servidor Go responde em 1–4 ms, como o do Koito; a lentidão percebida vem de busca e edição refazerem a página inteira e de arquivos estáticos sem cache. A página de mídia, por sua vez, não tem os ícones de ação do original e leva a uma página de gestão que mistura todas as tarefas. Estas cinco tarefas não dependem do redesign. Ordem: **106 → 109 → 107 → 108**, e a **110** logo depois da 109:

- **ISSUE-106** — cache imutável dos arquivos estáticos com hash na URL e `private, no-cache` nas páginas públicas, liberando o botão Voltar (bfcache).
- **ISSUE-109** — selo ShieldCheck no avatar do cabeçalho quando a sessão de administrador está ativa, com a sessão detectada no servidor (base para a 108 e a 110).
- **ISSUE-107** — busca enquanto digita no diálogo e edição de faixa que atualiza só a seção alterada, com fragmentos HTML do servidor e um script pequeno; sem JavaScript, tudo continua funcionando e as ações voltam à âncora da seção.
- **ISSUE-108** — barra de ícones no topo das páginas de faixa, álbum e artista, como no Koito (MusicBrainz, adicionar escuta, editar, trocar imagem, mesclar, excluir), só para administradores; cada ícone abre um diálogo com uma única tarefa, e a página de gestão misturada é dividida por tarefa.
- **ISSUE-110** — barra lateral: o penúltimo ícone vira Palette e abre um modal só com tema e idioma; o último (engrenagem) abre um submenu com Administração, Configuração do servidor, Conta e Sobre.


**Bloco A — estrutura sem mudança visual (prova por comparação)**
1. **ISSUE-061** — aceite do estudo e três decisões: embutido + leitura do disco em desenvolvimento; nomes `templates/` e `assets/`; fontes locais nesta fase.
2. **ISSUE-065** — referência de HTML/CSS renderizados e capturas de base (320–1440px, Yuu e Pearl).
3. **ISSUE-066** — mover para `templates/` e `assets/` na raiz com `go:embed` e `TUNETRAIL_WEB_DIR`. HTML idêntico byte a byte.
4. **ISSUE-067** — reformatar templates, unificar head e barra lateral, limite de 160 caracteres por linha no `ui.check`.
5. **ISSUE-068** — CSS em camadas (`@layer`), concatenado pelo Go com hash e cache imutável.

**Bloco B — design responsivo (aceite visual)**
6. **ISSUE-069** — tokens, pontos de quebra 40/64/90rem, container queries, fim das larguras fixas.
7. **ISSUE-070** — fontes locais, sem Google Fonts.
8. **ISSUE-071** — navegação: barra inferior no celular (decisão do usuário), lateral de ícones e expansível.
9. **ISSUE-072**, **ISSUE-073** e **ISSUE-074** — home com KPIs e heatmap rolável; listas e mídia compactas com menu `⋯`; administração em duas colunas, badge Admin e `/settings` (ROADMAP item 2, parte de layout). Podem andar em paralelo depois da 071.

**Bloco C — interações e fechamento**
10. **ISSUE-075** — estende os fragmentos da 107 a rankings, histórico e mídia; View Transitions e Speculation Rules; encerra o estudo **ISSUE-056** (ROADMAP item 8).
11. **ISSUE-076** (PWA) e **ISSUE-077** (curvas do gráfico) — baixa prioridade, quando houver espaço.
12. **ISSUE-078** — auditoria WCAG 2.2 AA, metas de desempenho (CSS < 40 KB, JS < 25 KB, LCP < 1,5s) e aceite da fase.

**Saída da fase:** critérios do doc-010 §3.6 cumpridos e aceitos pelo usuário.

## Tarefas por fase

### Fase 0 · Base (`m-9`) — concluída

ISSUE-049 publicada na 0.9; ISSUE-062, ISSUE-063 e ISSUE-064 integradas sem versão (13/11/2026).

### Fase 1 · UI responsiva (`m-10`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-056](<../../tasks/issue-056 - 056-—-Estudar-interações-mais-dinâmicas-na-interface.md>) | Estudar interações mais dinâmicas na interface | spike | — | — | — | Aberta |
| [ISSUE-061](<../../tasks/issue-061 - 061-—-Estudar-a-reorganização-do-CSS-e-dos-templates-HTML.md>) | Estudar a reorganização do CSS e dos templates HTML | spike | — | — | — | Aguardando aceite |
| [ISSUE-065](<../../tasks/issue-065 - 065-—-Referência-de-renderização-e-capturas-de-base-da-interface.md>) | Referência de renderização e capturas de base da interface | chore | high | 061 | — | Aberta |
| [ISSUE-066](<../../tasks/issue-066 - 066-—-Mover-templates-e-assets-para-templates-e-assets-na-raiz.md>) | Mover templates e assets para templates/ e assets/ na raiz | chore | high | 065 | — | Aberta |
| [ISSUE-067](<../../tasks/issue-067 - 067-—-Reformatar-templates-e-unificar-head-barra-lateral-e-nomes-de-blocos.md>) | Reformatar templates e unificar head, barra lateral e nomes de blocos | chore | high | 066 | — | Aberta |
| [ISSUE-068](<../../tasks/issue-068 - 068-—-Dividir-o-CSS-em-camadas-com-concatenação-e-cache-no-Go.md>) | Dividir o CSS em camadas com concatenação e cache no Go | chore | high | 067 | — | Aberta |
| [ISSUE-069](<../../tasks/issue-069 - 069-—-Tokens-de-design-três-pontos-de-quebra-e-container-queries.md>) | Tokens de design, três pontos de quebra e container queries | enhancement | high | 068 | +0.1 | Aberta |
| [ISSUE-070](<../../tasks/issue-070 - 070-—-Servir-as-fontes-localmente-e-remover-o-Google-Fonts.md>) | Servir as fontes localmente e remover o Google Fonts | enhancement | medium | 068 | +0.1 | Aberta |
| [ISSUE-071](<../../tasks/issue-071 - 071-—-Navegação-responsiva-barra-inferior-lateral-de-ícones-e-expansível.md>) | Navegação responsiva: barra inferior, lateral de ícones e expansível | feature | high | 069 | +0.1 | Aberta |
| [ISSUE-072](<../../tasks/issue-072 - 072-—-Home-e-painel-KPIs-tocando-agora-heatmap-rolável-e-grade-de-destaques.md>) | Home e painel: KPIs, tocando agora, heatmap rolável e grade de destaques | feature | medium | 071 | +0.1 | Aberta |
| [ISSUE-073](<../../tasks/issue-073 - 073-—-Listas-histórico-e-páginas-de-mídia-compactas-com-menu-de-ações.md>) | Listas, histórico e páginas de mídia compactas, com menu de ações | feature | medium | 071 | +0.1 | Aberta |
| [ISSUE-074](<../../tasks/issue-074 - 074-—-Administração-em-duas-colunas-badge-de-admin-e-página-de-configurações.md>) | Administração em duas colunas, badge de admin e página de configurações | feature | medium | 071 | +0.1 | Aberta |
| [ISSUE-075](<../../tasks/issue-075 - 075-—-Fragmentos-parciais-View-Transitions-Speculation-Rules-e-busca-com-sugestões.md>) | Fragmentos parciais, View Transitions e Speculation Rules | feature | medium | 073, 056, 107 | +0.1 | Aberta |
| [ISSUE-076](<../../tasks/issue-076 - 076-—-App-instalável-com-service-worker-mínimo-e-página-offline.md>) | App instalável com service worker mínimo e página offline | feature | low | 068, 070 | +0.1 | Aberta |
| [ISSUE-077](<../../tasks/issue-077 - 077-—-Curvas-mais-suaves-no-gráfico-de-escutas-sem-ultrapassar-os-pontos.md>) | Curvas mais suaves no gráfico de escutas sem ultrapassar os pontos | enhancement | low | — | +0.1 | Aberta |
| [ISSUE-078](<../../tasks/issue-078 - 078-—-Auditoria-de-acessibilidade-desempenho-e-aceite-da-UI-responsiva.md>) | Auditoria de acessibilidade, desempenho e aceite da UI responsiva | chore | high | 072, 073, 074, 075, 070 | — | Aberta |
| [ISSUE-106](<../../tasks/issue-106 - 106-—-Cache-dos-arquivos-estáticos-e-do-botão-Voltar.md>) | Cache dos arquivos estáticos e do botão Voltar | enhancement | high | — | +0.1 | Aberta |
| [ISSUE-107](<../../tasks/issue-107 - 107-—-Busca-instantânea-e-edição-de-faixas-sem-recarregar-a-página.md>) | Busca instantânea e edição de faixas sem recarregar a página | feature | high | 106 | +0.1 | Aberta |
| [ISSUE-108](<../../tasks/issue-108 - 108-—-Barra-de-ações-nas-páginas-de-mídia-e-edição-separada-por-tarefa.md>) | Barra de ações nas páginas de mídia e edição separada por tarefa | feature | high | 107, 109 | +0.1 | Aberta |
| [ISSUE-109](<../../tasks/issue-109 - 109-—-Indicador-de-sessão-de-administrador-no-cabeçalho-do-perfil.md>) | Indicador de sessão de administrador no cabeçalho do perfil | feature | high | 106 | +0.1 | Aberta |
| [ISSUE-110](<../../tasks/issue-110 - 110-—-Barra-lateral-ícone-de-tema-com-modal-simples-e-menu-de-administração.md>) | Barra lateral: ícone de tema com modal simples e menu de administração | feature | high | 109 | +0.1 | Aberta |

### Fase 2 · Ingestão confiável (`m-11`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-079](<../../tasks/issue-079 - 079-—-Contrato-de-ingestão-retenção-e-revisão-do-Advisor.md>) | Contrato de ingestão, retenção e revisão do Advisor | spike | high | — | — | Aberta |
| [ISSUE-080](<../../tasks/issue-080 - 080-—-Migração-SQLite-v4-ingest_events-outbox-e-integrations.md>) | Migração SQLite v4: ingest_events, outbox e integrations | feature | high | 079 | +0.1 | Aberta |
| [ISSUE-081](<../../tasks/issue-081 - 081-—-Pipeline-de-ingestão-com-deduplicação-explicável.md>) | Pipeline de ingestão com deduplicação explicável | feature | high | 080 | +0.1 | Aberta |
| [ISSUE-082](<../../tasks/issue-082 - 082-—-Outbox-durável-no-lugar-do-relay-best-effort.md>) | Outbox durável no lugar do relay best-effort | feature | high | 081 | +0.1 | Aberta |
| [ISSUE-083](<../../tasks/issue-083 - 083-—-Central-de-saúde-da-ingestão-e-por-que-esta-escuta-está-aqui.md>) | Central de saúde da ingestão e "por que esta escuta está aqui | feature | medium | 082 | +0.1 | Aberta |
| [ISSUE-084](<../../tasks/issue-084 - 084-—-Tocando-agora-em-tempo-real-com-Server-Sent-Events.md>) | Tocando agora em tempo real com Server-Sent Events | feature | medium | 081, 072 | +0.1 | Aberta |

### Fase 3 · Integrações (`m-12`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-085](<../../tasks/issue-085 - 085-—-Prova-de-conceito-com-coletores-externos-e-rota-de-compatibilidade.md>) | Prova de conceito com coletores externos e rota de compatibilidade | spike | high | 081 | — | Aberta |
| [ISSUE-086](<../../tasks/issue-086 - 086-—-Contrato-de-fontes-e-agendador-de-sincronização.md>) | Contrato de fontes e agendador de sincronização | feature | high | 082 | +0.1 | Aberta |
| [ISSUE-087](<../../tasks/issue-087 - 087-—-Conciliação-do-histórico-com-o-ListenBrainz-com-prévia.md>) | Conciliação do histórico com o ListenBrainz, com prévia | feature | medium | 086 | +0.1 | Aberta |
| [ISSUE-088](<../../tasks/issue-088 - 088-—-Fonte-Subsonic-Navidrome-tocando-agora-e-histórico.md>) | Fonte Subsonic/Navidrome: tocando agora e histórico | feature | medium | 086 | +0.1 | Aberta |
| [ISSUE-089](<../../tasks/issue-089 - 089-—-Webhooks-do-Jellyfin-e-do-Plex.md>) | Webhooks do Jellyfin e do Plex | feature | medium | 081 | +0.1 | Aberta |
| [ISSUE-090](<../../tasks/issue-090 - 090-—-Spotify-por-perfil-com-OAuth.md>) | Spotify por perfil com OAuth | feature | low | 086, 085 | +0.1 | Aberta |
| [ISSUE-091](<../../tasks/issue-091 - 091-—-Endpoint-compatível-com-Audioscrobbler-Last.fm.md>) | Endpoint compatível com Audioscrobbler/Last.fm | feature | low | 081 | +0.1 | Aberta |

### Fase 4 · Insights (`m-13`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-055](<../../tasks/issue-055 - 055-—-Aproveitar-tags-e-datas-de-lançamento-das-músicas.md>) | Aproveitar tags e datas de lançamento das músicas | feature | — | 079 | +0.1 | Aberta |
| [ISSUE-092](<../../tasks/issue-092 - 092-—-Semântica-das-estatísticas-médias-e-desempenho-em-históricos-grandes.md>) | Semântica das estatísticas, médias e desempenho em históricos grandes | spike | high | — | — | Aberta |
| [ISSUE-093](<../../tasks/issue-093 - 093-—-Comparação-de-períodos-e-setas-de-posição-nos-rankings.md>) | Comparação de períodos e setas de posição nos rankings | feature | medium | 092, 072 | +0.1 | Aberta |
| [ISSUE-094](<../../tasks/issue-094 - 094-—-Redescobertas-e-relógio-de-escuta.md>) | Redescobertas e relógio de escuta | feature | low | 092 | +0.1 | Aberta |
| [ISSUE-095](<../../tasks/issue-095 - 095-—-Sessões-de-escuta.md>) | Sessões de escuta | feature | low | 092 | +0.1 | Aberta |
| [ISSUE-096](<../../tasks/issue-096 - 096-—-Grupos-de-artistas-sem-fundir-entidades.md>) | Grupos de artistas sem fundir entidades | feature | low | 092 | +0.1 | Aberta |
| [ISSUE-097](<../../tasks/issue-097 - 097-—-Qualidade-do-histórico-e-tempo-realmente-ouvido.md>) | Qualidade do histórico e tempo realmente ouvido | feature | low | 083 | +0.1 | Aberta |

### Fase 5 · Compartilhar (`m-14`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-098](<../../tasks/issue-098 - 098-—-Especificação-dos-cards-sociais.md>) | Especificação dos cards sociais | spike | medium | 070 | — | Aberta |
| [ISSUE-099](<../../tasks/issue-099 - 099-—-Gerar-e-baixar-cards-sociais.md>) | Gerar e baixar cards sociais | feature | medium | 098 | +0.1 | Aberta |
| [ISSUE-100](<../../tasks/issue-100 - 100-—-Perfis-públicos-ou-privados-com-senha-de-visualização.md>) | Perfis públicos ou privados com senha de visualização | feature | high | 074 | +0.1 | Aberta |
| [ISSUE-101](<../../tasks/issue-101 - 101-—-Widget-SVG-de-tocando-agora.md>) | Widget SVG de tocando agora | feature | low | 100 | +0.1 | Aberta |

### Fase 6 · Operação (`m-15`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-048](<../../tasks/issue-048 - 048-—-Documentação-e-site-oficial-do-TuneTrail.md>) | Documentação e site oficial do TuneTrail | docs | — | — | — | Aberta |
| [ISSUE-102](<../../tasks/issue-102 - 102-—-Container-sem-root-com-migração-de-permissões.md>) | Container sem root com migração de permissões | chore | medium | — | +0.1 | Aberta |
| [ISSUE-103](<../../tasks/issue-103 - 103-—-Backup-e-restauração-pela-interface.md>) | Backup e restauração pela interface | feature | medium | — | +0.1 | Aberta |
| [ISSUE-104](<../../tasks/issue-104 - 104-—-OpenAPI-e-métricas-Prometheus.md>) | OpenAPI e métricas Prometheus | feature | low | 082 | +0.1 | Aberta |
| [ISSUE-112](<../../tasks/issue-112 - 112-—-Logs-diagnóstico-correções-e-visualização-na-administração.md>) | Logs: diagnóstico, correções e visualização na administração | feature | medium | — | +0.1 | Aberta |

### Identidade TuneTrail (`m-5`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-111](<../../tasks/issue-111 - 111-—-Trocar-Koito-por-TuneTrail-na-identidade-configuração-e-navegador.md>) | Trocar Koito por TuneTrail na identidade, configuração e navegador | enhancement | high | — | +0.1 | Aberta |

### Marco TuneTrail 1.0 (`m-8`)

| Tarefa | Título | Tipo | Prioridade | Depende de | Versão | Status |
| --- | --- | --- | --- | --- | --- | --- |
| [ISSUE-105](<../../tasks/issue-105 - 105-—-Candidatura-ao-TuneTrail-1.0.md>) | Candidatura ao TuneTrail 1.0 | chore | medium | 078, 083, 099, 100, 102, 103, 048 | — | Aberta |

## Rastreabilidade

### Itens do ROADMAP antigo (doc-001)

| Item | Pedido | Tarefas |
| --- | --- | --- |
| 2 | Layout da administração, badge de admin, perfis públicos/privados com senha simples | 109 (selo de sessão de admin) · 110 (menu de administração na barra lateral) · 074 (layout, `/settings`) · 100 (senha de visualização, só admin apaga e cria perfis) · 101 (widget respeita privacidade) |
| 4 | Escutas do Spotify, Deezer, Tidal e outros serviços | 085 (PoC externa; orientação Deezer/Tidal) · 086 (contrato de fontes) · 087 (ListenBrainz) · 088 (Subsonic/Navidrome) · 089 (Jellyfin/Plex) · 090 (Spotify) · 091 (Last.fm) — sobre a base da Fase 2 |
| 5 | Imagens para redes sociais | 098 (especificação) · 099 (geração e download) |
| 7 | Tags, gêneros e datas de lançamento | 055 (existente), coordenada com a migração v4 (079/080) |
| 8 | Interações e funções dinâmicas | 056 (estudo existente) → 106 (cache e bfcache), 107 (busca instantânea e edição sem recarregar), 108 (barra de ações e edição por tarefa nas páginas de mídia), 075 (fragmentos e transições) e 084 (tocando agora por SSE) |

### Pendências da revisão (doc-017) e da avaliação (doc-010 §1.3)

| Pendência | Tarefa |
| --- | --- |
| Registros: roadmap com React, 046 sem critérios marcados, estudo antigo | 062 |
| Scripts de validação por issue fora do CI | 063 |
| Recuperação de migração, importação e restauração (046, 020/021) | 064 |
| Bug de catálogo 049 | 049 |
| CSS difícil de manter, templates em linha única, 13 pontos de quebra | 061, 065–069 |
| Fontes externas (Google Fonts) | 070 |
| Busca e edição lentas em relação ao Koito React; estáticos sem cache; `no-store` nas páginas públicas | 106, 107 |
| Logs só na saída do contêiner, senha do reset-password no log, sem visualização na interface | 112 |
| Página de mídia sem ícones de ação; link de gestão visível a visitantes; página de gestão misturando todas as tarefas | 108, 073 |
| Nenhum sinal de sessão de administrador; ícone Info só com link; configurações em modal com abas | 109, 110 |
| Acessibilidade sem validação abrangente (024/026) | 078 |
| Curvas do gráfico (047) | 077 |
| Relay best-effort, 429 sem repetição (020/034) | 082 |
| Semântica das médias e desempenho com histórico grande (018) | 092 |
| Container como root (034) | 102 |
| Gerador social desativado (035) | 098, 099 |
| Documentação e site (048) | 048 |
| Calendários excepcionais (018), coordenação entre instâncias (037), teto global de memória (032/036) | Sem tarefa: condicionais, conforme o doc-017; abrir somente com demanda real |

## Decisões pendentes do usuário

| Decisão | Bloqueia | Recomendação |
| --- | --- | --- |
| Arquivos web embutidos ou externos; nomes `templates/` e `assets/`; fontes locais na fase 1 | 061 → 065–070 | Embutidos com `TUNETRAIL_WEB_DIR` em desenvolvimento; sim; sim |
| Barra de navegação inferior no celular | 071 | Aceitar (doc-010 §3.3) |
| Retenção de `ingest_events` e guarda do payload bruto | 079 | 90 dias, payload redigido |
| Spotify nativo ou somente coletor externo | 090 | Decidir depois da PoC (085) |
| Semântica das médias | 092 | Documentar antes de mudar cálculos |
| Candidatura ao 1.0 | 105 | Reservada ao usuário |

## Versionamento

**Regra decidida pelo usuário em 14/11/2026**, a partir da prática das versões 0.8 e 0.9:

| A tarefa… | Como entra no `main` | Exemplos |
| --- | --- | --- |
| muda o produto: comportamento, interface, dados, API ou forma de instalar e executar (`bug`, `feature`, `enhancement` e `chore` que altere imagem ou implantação) | **publica uma versão**: `make issue.publish ISSUE=N`, que sobe 0.x → 0.x+1, fecha `[Unreleased]` no CHANGELOG e cria a tag `v0.x` | 049 → 0.9; 106; 111; 102 |
| não muda o produto: estudo (`spike`), documentação (`docs`), testes, ferramentas, CI e registros do backlog | **sem versão**: merge fast-forward no `main`, sem tag; uma entrada no CHANGELOG, se houver, espera a próxima versão publicada | 062, 063, 064; 061, 065–068, 078 |

- Uma tarefa que publica versão gera exatamente uma versão; tarefas não são agrupadas numa publicação.
- O push e a tag dependem de autorização do usuário em cada publicação (`PUSH=1`); o push da tag publica a imagem Docker da versão e `latest`.
- O CI exige versão não decrescente (`version.py --check-not-lower`); a publicação exige aumento estrito (`--check-parent`).
- Projeção: das 52 tarefas abertas, **39 publicam versão**, levando de 0.9 a cerca de **0.48** antes do 1.0. A decisão do 1.0 continua reservada ao usuário (ISSUE-105).
- Descartadas: uma versão por fase e versões de correção 0.x.y.
