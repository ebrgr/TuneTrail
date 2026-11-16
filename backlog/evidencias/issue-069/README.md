# ISSUE-069 — Tokens e adaptação por espaço disponível

O usuário revogou a preservação visual e autorizou o escopo original. A referência candidata neste diretório é um contrato técnico de regressão; **não representa aceite visual**.

## Capturas

- `antes/`: 132 PNGs, gerados **antes da edição do código**, diretamente de `backlog/evidencias/ui-responsiva/base/reference/` (ISSUE-065).
- `depois/`: 132 PNGs da referência candidata `reference/` com os mesmos 11 tipos de página, temas Yuu/Pearl, larguras 320/360/390/768/1024/1440, altura 900 e DPR 1.
- Cada `manifest.json` registra versão do navegador, overflow horizontal, falhas de fontes e tipografia efetiva da retrospectiva. Capturas são da área visível, exceto retrospectiva, capturada integralmente conforme a ISSUE-065.
- `responsive.json`: auditoria adicional de todas as 23 páginas HTML exportadas × dois temas × 13 larguras (598 vistas), incluindo 639/640/641, 1023/1024/1025 e 1439/1440/1441. Mede overflow estrito e media queries efetivamente carregadas; a lista `outside` é diagnóstico e pode incluir conteúdo de contêineres com rolagem interna intencional.

As capturas “antes” usam a versão histórica 065, enquanto “depois” inclui o produto atual. Diferenças anteriores à 069 não devem ser atribuídas automaticamente a esta tarefa. Fontes Google continuam sendo usadas no produto e permitidas na captura; falhas ficam registradas. A auditoria adicional também verifica o layout com as fontes de fallback sem rede. Não se exige igualdade de pixels: a mudança visual é intencional.

## Reprodução

Instale Playwright fora do repositório e informe `PLAYWRIGHT_MODULE` com seu caminho. O navegador usado é Chrome instalado; `CHROME_PATH` pode selecionar outro binário Chromium.

```sh
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/capture_ui_base.cjs backlog/evidencias/ui-responsiva/base/reference /tmp/issue069-antes
make ui.reference OUT=/tmp/issue069-reference
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/capture_ui_base.cjs /tmp/issue069-reference /tmp/issue069-depois
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/validate_responsive.cjs /tmp/issue069-reference /tmp/issue069-responsive.json
make ui.check
make check
```

## Contrato de layout

`tokens.css` define oito espaços, sete passos fluidos (`--step--1` até `--step-5`), raios e sombras. As media queries de largura aceitam somente `min-width:40rem`, `64rem` e `90rem`. Condições de acessibilidade e capacidade do dispositivo (forced colors, redução de movimento, hover) permanecem independentes; não são breakpoints de layout.

Container queries nomeadas adaptam os destaques da Home, o heatmap e as listas. A grade usa `auto-fill` com mínimo limitado ao espaço disponível. Os blocos de layout usam limites relativos, `min()` e `clamp()`; pixels permanecem em primitivas como ícones, células do heatmap, miniaturas, bordas e alvos de toque. Heatmap e tabelas podem ter rolagem interna, sem aumentar a largura da página.

`make ui.check` já executava `check_web_style.py`; o verificador foi ampliado com a política dos três breakpoints e testes de consultas legadas, intervalos, comentários e escapes. `ui.reference.check` passa a comparar com o candidato desta issue porque a igualdade com o CSS da 067 deixou de ser o contrato autorizado. A referência original permanece intacta.

O aceite visual do AC #4 permanece exclusivamente com o usuário. Não houve push, merge, publicação ou mudança para Done.
