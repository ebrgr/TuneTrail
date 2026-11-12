---
id: ISS-058
title: 058 — Estudar a adoção do Tailwind CSS
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - ui
milestone: m-8
dependencies: []
references:
  - docs-dev/issues/058-estudo-tailwind-css.md
type: spike
ordinal: 58000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/058-estudo-tailwind-css.md](../../docs-dev/issues/058-estudo-tailwind-css.md) — numeração local 058.

**Status:** Aberta — estudo · **Criada:** 12/11/2026 · **Roadmap:** [TuneTrail 2.0, pilar 2 — UI responsiva](../../docs-dev/ROADMAP_2.0.md#3-pilar-2--ui-20-responsiva)

## Objetivo

Avaliar se o Tailwind CSS deve ser incluído no TuneTrail para resolver os problemas de manutenção e responsividade do CSS atual. O estudo deve comparar o Tailwind com a alternativa de reorganizar o CSS próprio (tokens, `@layer` e container queries) e terminar com uma recomendação e, se for o caso, uma prova de conceito limitada. Esta issue não autoriza a migração das páginas nem publica nada no GitHub.

## Situação atual

- [ui.css](../../internal/webui/static/ui.css): 1.199 linhas e 28 KB. Mistura blocos expandidos e regras em linha única, tem 40 larguras fixas em px e 9 pontos de quebra diferentes (480, 639, 640, 650, 767 e 900px; 40, 48, 60, 64, 80 e 96rem).
- [admin.css](../../internal/webui/static/admin.css): 7 KB, quase todo em linha única.
- Os temas vêm de [palettes.json](../../internal/webui/palettes.json), com 14 cores por tema. O Go os entrega como variáveis CSS em `/themes.css`, e o navegador aceita tema personalizado em JSON (`theme.js`/`ui.js`).
- Os templates Go têm cerca de 310 atributos `class`, alguns montados dinamicamente: `class="activity-cell {{...}}"`, `class="rank {{...}}"`.
- O CSS e os templates são embutidos com `go:embed`. O Dockerfile só compila Go e não há etapa de build de front-end. O Node existe apenas no site de documentação (`docs/`, Astro).
- Princípio do projeto: um executável Go, HTML, CSS e o mínimo de JavaScript, sem dependências externas em produção.

## O que o Tailwind resolveria

| Problema atual | Com Tailwind | Sem Tailwind (CSS próprio reorganizado) |
| --- | --- | --- |
| Pontos de quebra inconsistentes | Escala única (`sm`, `md`, `lg`...) imposta pelas classes | Três variáveis de quebra + container queries, mantidas por disciplina e revisão |
| Larguras fixas em px | Utilitários `w-full`, `max-w-*`, `min-w-0` facilitam o padrão fluido | `min()`, `clamp()` e grades `auto-fill` em componentes |
| CSS que só cresce | O CSS gerado contém apenas as classes usadas | Exige remover regras mortas manualmente |
| Espaçamentos soltos | Escala de espaço consistente | Tokens `--space-*` |
| Container queries | Nativas no v4 (`@container`, `@sm:`) | Nativas no CSS atual dos navegadores |

## Custos e riscos

1. **Etapa de build nova.** É preciso gerar o CSS antes do `go build`. Dá para evitar o Node usando o **executável standalone** do Tailwind, mas ainda é um binário por plataforma (macOS/Linux, x64/arm64) para fixar a versão, verificar o checksum e manter.
2. **Classes dinâmicas.** O Tailwind só encontra classes escritas por inteiro nos arquivos. As classes montadas em templates (`activity-cell {{...}}`) precisam ser declaradas explicitamente (`@source inline(...)` no v4.1+) ou trocadas por classes completas.
3. **Legibilidade dos templates.** Os templates já estão em linha única; listas longas de utilitários (`flex items-center gap-4 md:gap-6 ...`) pioram os diffs e a revisão visual. Mitigação: quebrar os templates em várias linhas e usar `@apply` ou componentes CSS para padrões repetidos.
4. **Reset (Preflight).** O reset do Tailwind muda margens, títulos, listas e botões. Ligado de uma vez, quebra as páginas atuais e a paridade visual obtida nas issues 022–030 e 057.
5. **Navegadores.** O Tailwind v4 usa `@property`, `color-mix()` e cascade layers e exige navegadores recentes (Safari 16.4+, Chrome 111+, Firefox 128+). Definir se isso é aceitável para um app self-hosted acessado de TVs e tablets antigos.
6. **Temas.** As 14 cores podem ser mapeadas no `@theme` apontando para as variáveis já existentes (por exemplo `--color-surface: var(--color-bg)`), preservando os temas embutidos e os personalizados. Verificar colisões de nome com as variáveis `--color-*` do próprio Tailwind.
7. **Esforço de migração.** Reescrever cerca de 310 atributos de classe e 35 KB de CSS, refazer as capturas comparativas e reabrir aceites visuais. É o maior custo, e o resultado é parecido com o de reorganizar o CSS próprio.
8. **Dependência de terceiros.** Uma ferramenta a mais para atualizar, com mudanças entre versões maiores (a passagem v3 → v4 trocou a configuração em JS por configuração em CSS).

## Formas de adoção a comparar

| Opção | Como | Avaliação inicial |
| --- | --- | --- |
| A. CDN / Play | `<script>` que gera CSS no navegador | **Descartar.** Não é para produção, depende de rede e contraria o princípio de JS mínimo. |
| B. Tailwind completo via standalone | `make css` gera `static/app.css` a partir de `styles/app.css`; o CSS gerado é commitado e embutido | Viável, mas exige migrar tudo e reabrir as validações visuais. |
| C. Convivência gradual | `@import "tailwindcss" prefix(tw);` sem Preflight (importar só as camadas `theme` e `utilities`); páginas novas usam `tw:*` e as antigas continuam com o CSS atual | Menor risco; por um período convivem dois sistemas. |
| D. Sem Tailwind | CSS próprio dividido em `tokens/base/layout/components/pages` com `@layer`, três pontos de quebra, container queries e uma pequena camada de utilitários próprios (~40 classes) | Mantém zero dependências; depende de disciplina e de uma verificação no CI (lint de px fixos e breakpoints fora da escala). |

### Esboço da opção B/C sem Node (só para a prova de conceito)

```make
TAILWIND_VERSION ?= <fixar versão 4.x>
TAILWIND := bin/tailwindcss-$(TAILWIND_VERSION)

$(TAILWIND):
	# baixar o standalone da plataforma em github.com/tailwindlabs/tailwindcss/releases
	# e verificar o sha256 antes de chmod +x

css: $(TAILWIND)
	$(TAILWIND) -i internal/webui/styles/app.css -o internal/webui/static/app.css --minify

css.check: css
	git diff --exit-code internal/webui/static/app.css
```

- Commitar o CSS gerado: o Dockerfile e o `go build` continuam iguais, e o CI roda `make css.check` para garantir que o arquivo está atualizado.
- O `app.css` de entrada declara `@source "../templates"` e `@source "../../adminui/templates"`, o `@theme` ligado às variáveis atuais e `@source inline(...)` para as classes dinâmicas.

## Recomendação preliminar

**Não adotar o Tailwind como base agora.** Os problemas encontrados (pontos de quebra, px fixos, estilos misturados) vêm da falta de tokens e de organização, não da falta de um framework, e o CSS moderno resolve todos eles sem build nem dependência. A migração completa custaria tanto quanto reorganizar o CSS próprio e ainda acrescentaria uma ferramenta, uma etapa de build e templates mais difíceis de ler.

Seguir a **opção D** no pilar 2 do roadmap 2.0. Reavaliar o Tailwind (opção C) somente se a prova de conceito abaixo mostrar ganho claro.

## Prova de conceito (opcional, depois da decisão do usuário)

Escopo: **uma página**, a de configurações da administração (`settings.html`), feita duas vezes em uma branch descartável:

1. com Tailwind standalone na opção C (prefixo `tw`, sem Preflight);
2. com CSS próprio na opção D.

Medir e registrar em `comparisons/tailwind-058/`:

- Tamanho do CSS final (bruto e gzip) e do template.
- Linhas alteradas e tempo gasto.
- Legibilidade do diff do template.
- Capturas em 320, 390, 768 e 1440px, nos temas Yuu e Pearl.
- Contraste, foco e zoom de 200%.
- Efeito no CI e no build Docker (tempo e passos).

## Critérios para concluir

- [ ] Confirmar a versão atual do Tailwind, as plataformas do executável standalone, a licença (MIT) e a lista de navegadores suportados na data da decisão.
- [ ] Comparar as opções A–D com estimativas revisadas de esforço e risco.
- [ ] Verificar o mapeamento dos 14 tokens de tema e dos temas personalizados, e o tratamento das classes dinâmicas dos templates.
- [ ] Definir o impacto no Makefile, no CI, no Dockerfile e no fluxo de desenvolvimento (VS Code, `make ui.preview`).
- [ ] Executar a prova de conceito, se autorizada, e registrar as evidências.
- [ ] Apresentar a recomendação final ao usuário antes de encerrar esta issue. Se a decisão for adotar, abrir issues de implementação separadas (e a issue no GitHub, conforme `issue-first`).
<!-- SECTION:DESCRIPTION:END -->
