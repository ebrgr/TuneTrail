# ISSUE-067 — capturas

Roteiro: `capturas.sh BASE NOVA` (Chrome headless 157, escala 1, janela de 2.400 px de altura, 3 s de tempo virtual, fontes do Google carregadas da rede nos dois lados).

- Base: `backlog/evidencias/issue-106/reference` (saída do `main` em `8700fbb`).
- Nova: saída da branch da 067, igual a `backlog/evidencias/issue-067/reference`.
- 23 páginas (12 públicas, 9 de administração, 2 da prévia) × 6 larguras (320, 360, 390, 768, 1024, 1440).

Resultado: **138 de 138 capturas idênticas byte a byte**, incluindo as 9 páginas da administração, que agora carregam as fontes como as páginas públicas.

Limite: o Chrome headless não reduz a janela abaixo de cerca de 500 px de largura, então as capturas de 320, 360 e 390 px mostram o layout dessa largura mínima, igual nos dois lados. A equivalência abaixo disso vem das comparações de HTML e CSS (`comparar_normalizado.py`), não das capturas.
