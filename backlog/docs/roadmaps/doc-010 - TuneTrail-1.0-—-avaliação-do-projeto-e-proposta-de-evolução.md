---
id: doc-010
title: TuneTrail 1.0 — avaliação do projeto e proposta de evolução
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:54:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_2.0.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# TuneTrail 1.0 — avaliação do projeto e proposta de evolução

**Data:** 12/11/2026 · **Base avaliada:** TuneTrail 0.6, HEAD `be07697` + 113 arquivos com alterações locais não commitadas.
**Natureza:** proposta para decisão do usuário. Não autoriza implementação, não abre issues no GitHub e não altera o plano vigente em [ROADMAP.md](<doc-001 - ROADMAP.md>).
**Fontes:** `agents.md`, README, CHANGELOG, roadmaps 01–06, issues locais 001–057, [revisão de pendências](<../estudos/doc-017 - Revisão-das-issues-e-pendências-remanescentes.md>), [issue 056](<../../tasks/iss-056 - 056-—-Estudar-interações-mais-dinâmicas-na-interface.md>), grafo Graphify, CSS/templates atuais, capturas em `comparisons/` e os estudos privados em `estudos-locais/` (Maloja, maloja-rs, multi-scrobbler, Scrobblet e síntese).

> Os estudos locais estão em `.git/info/exclude`. Este documento cita apenas conclusões conceituais e nomes de projetos públicos; nenhum código externo é proposto para cópia. Se este arquivo for commitado, os links para `estudos-locais/` ficarão quebrados para outras pessoas — decisão do usuário.

---

## 1. Avaliação do estado atual

### 1.1 O que foi entregue com o Codex (resumo)

| Bloco | Issues | Resultado |
| --- | --- | --- |
| Monólito Go sem React | 001–030 | Páginas públicas, mídia, retrospectiva e administração renderizadas em Go; React removido. |
| Validação transversal | 016–021 | Permissões, CSRF/origem, fusos/DST, isolamento entre perfis, concorrência e falhas de I/O. |
| Imagens sem CGO | 031–037 | `imaging` + `webp-go-pure`; publicação atômica, limites, cancelamento; binário sem libvips. |
| Identidade e versão | 043–046 | Marca TuneTrail, versionamento 0.x, migração `koito.db` → `tunetrail.db`. |
| Configuração TOML | 038–042 | Leitura com precedência de ambiente, gravação atômica com revisão, página de configuração. 042 aguarda aceite. |
| Multilíngue | 050–054 | Catálogos externos `langs/en.json` e `pt-BR.json`; 052–054 aguardam aceite. |
| Ajustes | 047, 057 | Curva sem overshoot; controles da retrospectiva (057 aguarda avaliação). |

**Placar local:** 47 issues fechadas, 10 abertas (038, 042, 048, 049, 052–057).

### 1.2 Pontos fortes

- **Entrega simples:** um executável Go, SQLite puro (`modernc.org/sqlite`), sem CGO, sem Node em produção. É o maior diferencial em relação a Maloja (Python) e multi-scrobbler (Node/React).
- **Perfis isolados:** escuta, chaves, relay e consultas com escopo por perfil — nenhum dos quatro projetos estudados tem isso.
- **Segurança e robustez acima da média:** CSRF, origem, limites de corpo, timeouts de imagens, gravação atômica, testes de fuso/DST.
- **Funciona sem JavaScript:** navegação, filtros, busca e formulários.
- **Processo documentado:** cada decisão tem issue, evidências e capturas.

### 1.3 Pontos de atenção

| Área | Evidência | Impacto |
| --- | --- | --- |
| **Trabalho não commitado** | 113 entradas em `git status` na `main` (config TOML, i18n, retrospectiva, docs). | Alto risco de perda e de revisão difícil. Primeira ação recomendada. |
| **CSS difícil de manter** | `ui.css` com 1.199 linhas em dois estilos (expandido e linhas únicas minificadas); 40 larguras fixas em px (`350px`, `750px`, `385px`, `1000px`...); 9 breakpoints diferentes (480, 639, 640, 650, 767, 900px, 40/48/60/64/80/96rem). | Responsividade frágil: em celular, `.overview`, `.activity-card` e `.period-selector` ficam presos em 350–352px. |
| **Templates em linha única** | `page.html`, `live.html`, `media.html` concentram muitas tags por linha. | Diffs ilegíveis, revisão visual cara. |
| **Fontes externas** | `live.html` carrega Google Fonts, embora `assets/` já tenha Jost e League Spartan. | Contraria a proposta self-hosted/privacidade; falha offline. |
| **Scripts de validação por issue** | 51 scripts em `scripts/`, 37 com sufixo de issue (`validate_*_0NN.py`). | Dívida de teste: verificações pontuais que não rodam no CI como suíte. |
| **Arquivos soltos** | `agents - cópia.md`, diretório `soKz24cd/` (apenas `image_cache` e `.DS_Store`). | Ruído; limpar. |
| **Entrega do relay** | Best-effort, sem fila durável; 429 não é repetido. | Bloqueia integrações confiáveis. |
| **Container como root** | Dockerfile sem `USER`. | Risco operacional; já registrado na revisão. |
| **Semântica de estatísticas** | Média combina contagem filtrada com dias ativos históricos (018). | Números confusos para o usuário. |
| **Bug aberto de catálogo** | 049: relação artista–álbum residual após edição de faixa. | Página de artista mostra álbum indevido. |

### 1.4 UI atual vista nas capturas

- **Desktop (home):** conteúdo em coluna estreita com muito espaço vazio; estatísticas como lista de texto; os cards de topo ficam enormes quando há poucos itens; cada linha de "Last played" repete "Manage".
- **Celular:** a barra superior funciona, mas o heatmap e o resumo têm largura fixa; o cabeçalho do perfil ocupa uma faixa inteira; não há navegação inferior acessível ao polegar.
- **Administração no celular:** lista simples de links, sem hierarquia; "Sign out" com o mesmo peso visual de uma ação primária.
- **Configurações** ficam no rodapé e dependem de JavaScript.

---

## 2. Visão 1.0

> **TuneTrail 1.0: o histórico musical self-hosted mais simples de rodar e o mais confiável para entender.**
> Um binário Go, várias fontes, nenhuma escuta perdida, e cada número explicável.

Seis pilares, em ordem de dependência:

1. **Base saneada** — commits, aceite pendente e higiene.
2. **UI responsiva** — design system com tokens, mobile-first e interações progressivas.
3. **Ingestão confiável** — envelope de evento, outbox durável e central de saúde.
4. **Integrações** — fontes nativas em Go e compatibilidade com mais clientes.
5. **Insights** — comparações, evolução, redescobertas, sessões e tags.
6. **Compartilhar** — cards sociais e perfis públicos protegidos.

---

## 3. Pilar 2 — UI responsiva

### 3.1 Princípios

- **Mobile-first:** estilos base para 360px; ampliações com `min-width`.
- **Três breakpoints de layout**, mais container queries para componentes:
  - `--bp-sm: 40rem` (640px), `--bp-md: 64rem` (1024px), `--bp-lg: 90rem` (1440px).
  - Cards, heatmap e listas usam `@container` para se adaptar à largura **do espaço disponível**, não à da janela.
- **Nenhuma largura fixa em px para blocos:** `width: min(100%, 22rem)`, `grid-template-columns: repeat(auto-fill, minmax(min(100%, 14rem), 1fr))`.
- **Tipografia fluida:** `clamp()` com uma escala de tamanhos (`--step--1` a `--step-5`).
- **Tokens de espaçamento:** `--space-1` a `--space-8` substituem os valores soltos.
- **Progressive enhancement:** todo fluxo continua funcionando como documento HTML; o JS só acrescenta melhorias.

### 3.2 Reorganização do CSS

```
internal/webui/static/css/
  tokens.css      # cores (temas atuais), espaço, tipografia, raios, sombras, breakpoints
  base.css        # reset, tipografia, foco, reduced-motion, forced-colors
  layout.css      # app-shell, sidebar/bottom-nav, grades, containers
  components.css  # card, lista, tabela, botão, formulário, diálogo, chip, tabs, heatmap
  pages/*.css     # home, media, rewind, admin (somente o que for específico)
```

- Servir concatenado por `go:embed` com hash de conteúdo no nome (`ui.3f2a.css`) e `Cache-Control: immutable`. Sem bundler.
- Usar `@layer tokens, base, layout, components, pages;` para controlar especificidade sem `!important`.
- Formatar todo o CSS e os templates em várias linhas (uma tag/declaração por linha) — diffs legíveis.
- Fontes self-hosted (`woff2` em `static/fonts/`, `font-display: swap`); remover Google Fonts.

### 3.3 Navegação

| Largura | Padrão proposto |
| --- | --- |
| < 640px | **Barra inferior fixa** com 4–5 destinos (Início, Rankings, Histórico, Busca, Perfil/Mais) + `env(safe-area-inset-bottom)`. Cabeçalho do perfil compacto (avatar 32px + nome). |
| 640–1024px | Barra lateral só de ícones (atual), com rótulos em tooltip. |
| > 1024px | Barra lateral expansível com rótulos, perfil no topo e seletor de período persistente. |

### 3.4 Páginas

- **Home / painel:**
  - Linha de **KPIs em cards** (minutos, escutas, artistas, sequência) com variação contra o período anterior (▲ 12%).
  - **Tocando agora** no topo, quando houver.
  - Heatmap que **rola horizontalmente** em telas pequenas, ancorado na semana atual (`scroll-snap`, `overflow-x: auto`), em vez de esconder semanas.
  - Top artistas/álbuns em grade `auto-fill`: 2 colunas no celular, 4–6 no desktop, com capa quadrada menor; o primeiro colocado pode ganhar destaque.
  - Seletor de período **fixo** (sticky) abaixo do cabeçalho.
- **Listas e histórico:** no celular, linha com capa + título/artista + horário relativo; ações de administração em menu `⋯` (`<details>` ou popover), não um link "Manage" repetido.
- **Mídia (artista/álbum/faixa):** cabeçalho com capa e cor de destaque extraída da capa (calculada no servidor ao gerar o WebP — sem JS); abas *Visão geral · Faixas · Álbuns · Histórico · Gráficos*.
- **Retrospectiva:** cards em grade responsiva; botão **Compartilhar imagem** (pilar 6).
- **Administração:** layout em duas colunas no desktop (navegação de seções à esquerda, conteúdo à direita); no celular, lista de seções em cards; ações destrutivas em zona de perigo separada; badge **Admin** ao lado do nome (item 2 do roadmap atual).
- **Configurações** como página própria (`/settings`): tema, idioma, fuso, início da semana. Hoje ficam no rodapé.

### 3.5 Interações sem framework

Recomendação: **não adotar React, Vue ou HTMX como dependência de base**. Usar recursos nativos do navegador e um script próprio pequeno.

| Recurso | Técnica | JS necessário |
| --- | --- | --- |
| Transições suaves entre páginas | `@view-transition { navigation: auto; }` (View Transitions entre documentos) + `view-transition-name` na capa | Nenhum |
| Navegação instantânea | `<script type="speculationrules">` com prefetch/prerender de links do mesmo perfil | Nenhum (JSON declarativo) |
| Tocando agora ao vivo | **Server-Sent Events** (`/{slug}/events`) com `EventSource`; pausa com `visibilitychange` | ~40 linhas |
| Busca com sugestões | `<input list>` ou combobox ARIA + `fetch` de fragmento HTML `/{slug}/search?partial=1`, com `AbortController` e debounce | ~60 linhas |
| Filtros e paginação | Formulários GET normais; o script intercepta, busca o fragmento e troca `<main>`, mantendo a URL com `history.pushState` | ~50 linhas |
| Diálogos e menus | `<dialog>` e atributo `popover` nativos | Mínimo |
| Gráficos com detalhe | SVG do servidor com `<title>` por ponto + tooltip por CSS `:hover`/`:focus`; tabela acessível permanece | Nenhum |
| App instalável | `site.webmanifest` (já existe) + service worker mínimo para cache de assets e página offline | ~30 linhas |

O orçamento total de JS sobe de aproximadamente 14 KB para cerca de 20 KB, sem dependências externas. No servidor, os handlers passam a aceitar `?partial=1` (ou o cabeçalho `Sec-Fetch-Dest`) e renderizam apenas o bloco do template, reaproveitando o mesmo código.

**SSE em Go (esboço):** um `broker` por perfil, com canal por assinante, que o serviço de *now playing* notifica. Inclui `http.Flusher`, heartbeat de 25s, limite de conexões por IP e encerramento pelo `context` do request. Isso cobre a recomendação de "tocando agora" da issue 056 sem polling.

### 3.6 Critérios de aceite da UI responsiva

- Sem rolagem horizontal de página em 320, 360, 390, 768, 1024 e 1440px.
- Alvos de toque de pelo menos 44×44px; foco visível; zoom de 200% sem perda de conteúdo (WCAG 2.2 AA).
- LCP < 1,5s em rede 4G simulada com 10 mil escutas; CSS < 40 KB e JS < 25 KB (comprimidos).
- Todos os fluxos principais funcionam com JS desligado.
- Capturas comparativas antes/depois para os temas Yuu (escuro) e Pearl (claro).

---

## 4. Pilar 3 — Ingestão confiável

Conclusão comum dos estudos de multi-scrobbler, Maloja e Scrobblet: **a escuta final não basta; é preciso guardar o processo.**

### 4.1 Modelo de dados (migração SQLite v4, nova e incremental)

| Tabela | Conteúdo |
| --- | --- |
| `ingest_events` | `profile_id`, `source` (lbz, import, spotify, subsonic...), `client`, `received_at`, `listened_at`, `time_quality` (exato/estimado), `payload_json` redigido, `decision` (aceita, duplicata, rejeitada), `reason`, `listen_id`. Retenção configurável (ex.: 90 dias). |
| `outbox` | `profile_id`, `destination`, `listen_id`, `idempotency_key` único, `state` (pendente, enviado, falho, descartado), `attempts`, `next_attempt_at`, `last_error` redigido. |
| `integrations` | Fonte ou destino por perfil, tipo, configuração, `token_ref` (segredo fora do JSON), `checkpoint`, `last_ok_at`, `last_error`. |

### 4.2 Componentes Go

```
internal/ingest/     # Envelope, pipeline: normalizar → deduplicar → gravar → enfileirar
internal/dedup/      # Comparador: Igual | Diferente | Revisar, com motivo e pontuação
internal/outbox/     # Worker com backoff exponencial + jitter, Retry-After e drenagem no shutdown
internal/health/     # Visão por perfil: última entrada, fontes, falhas, fila
```

- O relay atual passa a ser **um destino da outbox**; o comportamento observável é preservado.
- A deduplicação acontece **dentro do perfil**. Repetições legítimas, versões ao vivo e remasters continuam distintas. Nunca apagar automaticamente por similaridade: casos duvidosos vão para revisão.
- **Central de saúde** (`/admin/health`): por perfil, mostra último envio, cliente, fila pendente, falhas recentes e o motivo de cada rejeição — sem tokens.
- **"Por que esta escuta está aqui?"**: na página da escuta, mostra origem, horário recebido, regra aplicada, decisão de duplicata e estado de entrega.

> **Advisor:** outbox, retry e deduplicação envolvem concorrência e um sistema de mensageria. Pela regra do `agents.md`, delegar a revisão a `@sol` com `effort=medium` antes da implementação.

---

## 5. Pilar 4 — Integrações

### 5.1 Contrato de fonte

```go
// Source is a per-profile listening source with declared capabilities.
type Source interface {
    Kind() string
    Capabilities() Caps // History, NowPlaying, Webhook
    FetchHistory(ctx context.Context, since Checkpoint) ([]Envelope, Checkpoint, error)
}
```

Polling adaptativo (ativo/ocioso), relógio injetável para testes, `Retry-After` e parada por `context`. O checkpoint avança somente **depois** da gravação durável.

### 5.2 Prioridade das fontes

| Ordem | Fonte | Viabilidade | Observação |
| --- | --- | --- | --- |
| 1 | **Conciliação ListenBrainz** (ler histórico do usuário no LB) | Alta: API pública e estável | Recupera escutas não enviadas pelo TuneTrail (roadmap item 4). Mostrar prévia antes de gravar. |
| 2 | **Subsonic/Navidrome** | Alta | A configuração Subsonic já existe; usar `getNowPlaying` e scrobble. |
| 3 | **Jellyfin / Plex (webhook)** | Alta | O servidor de mídia empurra o evento; sem polling. |
| 4 | **Spotify** | Média | OAuth por perfil e `recently-played` (últimas 50) + `currently-playing`. Cada usuário cria seu próprio app de desenvolvedor; o polling precisa ser frequente para não perder faixas. |
| 5 | **Endpoint Audioscrobbler/Last.fm-compatível** | Média | Atende players que só falam Last.fm (`/apis/audioscrobbler/2.0`), mapeado ao mesmo `ingest`. |
| 6 | **WiiM / dispositivos domésticos** | Média | Fonte por dispositivo atribuída a um perfil; ignorar metadados "Unknown". |
| — | Deezer | Baixa | API oficial de histórico depreciada; alternativas dependem de sessão interna (ARL). Não recomendado nativamente. |
| — | Tidal | Baixa | Sem API pública de histórico; orientar o uso de clientes que enviam via ListenBrainz. |

Antes de qualquer código nativo, fazer uma **prova de conceito externa** com Scrobblet (Spotify/WiiM) apontando para `/apis/listenbrainz` com uma chave de perfil. Isso valida o contrato sem custo de implementação. O adaptador Koito do multi-scrobbler consulta rotas de leitura sem slug e falha com mais de um perfil; vale oferecer uma rota de compatibilidade ou um cliente TuneTrail.

**Licenças:** TuneTrail é MIT; Maloja e Scrobblet são GPL-3.0. Implementar com código próprio a partir dos conceitos, sem tradução literal.

---

## 6. Pilar 5 — Insights

| Recurso | Descrição | Dependência |
| --- | --- | --- |
| **Comparação de períodos** | Este mês × anterior, com setas de posição (▲3, ▼1, NOVO) nos rankings. | Tipo `ValidRange` compartilhado entre consulta, tabela e gráfico. |
| **Evolução de ranking** | Gráfico de posição de um artista ou faixa ao longo dos meses (SVG do servidor). | Agregação SQL por intervalo; medir em base grande. |
| **Redescobertas** | "Você ouvia muito e parou": faixas com alta contagem e sem escuta há N meses. | Apenas o histórico local. |
| **Sessões** | Agrupa escutas próximas (intervalo configurável), mostra duração e transições. | Somente visualização; não altera eventos. |
| **Tags e gêneros** | Issue 055: tags da escuta com prioridade, complementadas pelo MusicBrainz; ranking por gênero. | Migração própria. |
| **Grupos de artistas** | Banda + projetos solo contados juntos sem fundir entidades, por perfil. | Nova tabela; reversível. |
| **Qualidade do histórico** | Durações ausentes, horários suspeitos, importações repetidas, lacunas de ingestão. | Pilar 3. |
| **Relógio de escuta** | Distribuição por hora do dia e dia da semana (radial ou barras). | Consulta simples. |
| **Tempo realmente ouvido** | Separar duração do catálogo e duração observada. | `ingest_events`. |

Antes disso, decidir e documentar a semântica das médias (pendência da 018).

---

## 7. Pilar 6 — Compartilhar

- **Cards sociais** (roadmap item 5): PNG 1080×1350 (feed) e 1080×1920 (stories) gerados em Go com `imaging` + `x/image/font` e as fontes já em `assets/`. Modelos: Top 5 do mês, Retrospectiva anual, Artista do mês. Prévia exata antes de baixar; opções para ocultar o nome do perfil e escolher o período.
- **Perfis públicos/privados** (roadmap item 2): senha simples por perfil só para visualização; admin com badge; somente o admin apaga dados e cria ou remove perfis.
- **Embed / widget:** `/{slug}/widget/now-playing.svg` para README do GitHub ou blog (opcional por perfil).

---

## 8. Pilar 1 e operação

- **Commitar o trabalho atual** em blocos lógicos (config TOML, i18n, retrospectiva, docs) e concluir os aceites pendentes de 042, 052–054 e 057.
- Corrigir a **049** antes de novos recursos de catálogo.
- **Consolidar os scripts `validate_*_0NN.py`** em uma suíte e2e única (Go `httptest` + Playwright opcional), rodando no CI; arquivar os scripts de issue.
- **Container sem root** (`USER 65532`) com nota de migração de permissões (Unraid).
- **Backup e restauração pela UI**: snapshot consistente (o mecanismo da migração 046 já existe), manifesto, verificação e ensaio de restauração.
- **OpenAPI** para `/apis/web/v1` e `/apis/listenbrainz`; **`/metrics`** Prometheus com labels de baixa cardinalidade (fonte, destino, estado), sem títulos nem perfis por padrão.
- Limpar `agents - cópia.md` e `soKz24cd/`.

---

## 9. Plano de execução proposto

Numeração indicativa: a 059 foi usada pela adoção do Backlog.md e a 058 pelo [estudo do Tailwind CSS](<../../tasks/iss-058 - 058-—-Estudar-a-adoção-do-Tailwind-CSS.md>), que antecede a 061. Os números abaixo serão renumerados quando as issues forem criadas. Esforço: **P** ≤ 2 dias, **M** 3–7 dias, **G** > 1 semana (estimativas iniciais).

| Fase | Issues | Entrega | Esforço |
| --- | --- | --- | --- |
| **0 · Base** | 058 commits e aceites pendentes · 059 correção da 049 · 060 limpeza e suíte e2e consolidada | `main` limpa, CI cobrindo jornadas | M |
| **1 · UI responsiva** | 061 tokens + reorganização do CSS + fontes locais · 062 navegação responsiva (barra inferior/lateral) · 063 home/painel com KPIs e heatmap rolável · 064 listas, mídia e menus `⋯` · 065 admin em duas colunas + `/settings` · 066 View Transitions + Speculation Rules + fragmentos parciais · 067 PWA | Interface mobile-first | G |
| **2 · Ingestão** | 068 migração v4 (`ingest_events`, `outbox`, `integrations`) · 069 pipeline `ingest` + dedup explicável · 070 outbox substitui relay best-effort · 071 central de saúde + "por que esta escuta" · 072 SSE de tocando agora | Nenhuma escuta perdida | G |
| **3 · Integrações** | 073 PoC externa Scrobblet → TuneTrail · 074 conciliação ListenBrainz com prévia · 075 Subsonic/Navidrome · 076 webhooks Jellyfin/Plex · 077 Spotify OAuth por perfil · 078 endpoint Audioscrobbler | Roadmap item 4 | G |
| **4 · Insights** | 079 comparação de períodos + setas de ranking · 080 redescobertas e relógio de escuta · 081 sessões · 082 tags/gêneros (055) · 083 grupos de artistas | Novas páginas de análise | G |
| **5 · Compartilhar** | 084 cards sociais · 085 perfis públicos/privados + badge admin · 086 widget SVG | Roadmap itens 2 e 5 | M |
| **6 · Operação** | 087 container sem root · 088 backup/restauração na UI · 089 OpenAPI + `/metrics` · 048 site de documentação | Pronto para o 1.0 | M |

As fases 1 e 2 podem correr em paralelo, pois tocam camadas diferentes. A 3 depende da 2. As partes da 4 que não dependem de `ingest_events` podem começar após a fase 1.

**Marco de versão sugerido:** o projeto está na **0.7**. Encerrar as fases 0–2 ao longo das versões **0.8–0.9**; com as fases 3–5 e a estabilização, candidatar a **TuneTrail 1.0** (a decisão sobre o 1.0 continua reservada ao usuário, conforme o README). Renomeado de "TuneTrail 2.0" para "TuneTrail 1.0" em 12/11/2026, pois esta visão corresponde à primeira versão estável.

---

## 10. Decisões pendentes do usuário

1. Ordem entre **UI responsiva** e **ingestão confiável** (recomendação: começar pela fase 0 e depois rodar 1 e 2 em paralelo).
2. Aceitar uma **barra de navegação inferior** no celular (muda a identidade herdada do Koito).
3. **Retenção** de `ingest_events` e se o payload bruto deve ser guardado.
4. **Spotify nativo × somente coletor externo**.
5. Commitar este documento ou mantê-lo como estudo local, por causa dos links para `estudos-locais/`.
