# ISSUE-122 — verificação

Base: `backlog/evidencias/issue-067/reference` (`main` em `5451b9e`, 0.15). Nova: saída da branch da 122.

## HTML

- Comparação estrita (`backlog/evidencias/issue-067/comparar_normalizado.py --strict`, espaços preservados): as 14 páginas públicas e da prévia são idênticas.
- As 9 páginas da administração diferem só nos ícones da barra: SVG escrito no template e máscaras CSS no avatar foram trocados pelos SVGs empacotados (função `icon`, os mesmos de `assets/icons/`). Removendo os `<svg>` dos dois lados, as 9 ficam idênticas.
- `admin.css` perde as máscaras `.admin-avatar-icon`; `admin.js` passa a usar `data-avatar-image`/`data-avatar-fallback`, os mesmos atributos do avatar público.

## Capturas

`backlog/evidencias/issue-067/capturas.sh` (23 páginas × 6 larguras, Chrome 157, fontes reais):

- 111 de 138 idênticas byte a byte (todas as públicas e da prévia, e as da administração em 320, 360 e 390 px).
- 27 diferem: as 9 páginas da administração em 768, 1024 e 1440 px, sempre nos mesmos 12 pixels (retângulo 23,2293–35,2348, ícones Sobre e Configurações), com diferença máxima de 1 em 255 por canal. É suavização de borda dos mesmos ícones Lucide escritos com atributos diferentes; não há diferença perceptível.
