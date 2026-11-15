# ISSUE-107 — tempos da busca

Medidos por `backlog/evidencias/issue-107/busca_instantanea.py`: servidor local, dados fictícios, 30 requisições, mediana, sem navegador no caminho.

| Requisição | Mediana | Tamanho |
|---|---:|---:|
| Página completa de busca (`/{slug}/search?q=Cache`) | 1,81 ms | 25.210 bytes |
| Fragmento da busca instantânea (`&partial=1`) | 0,85 ms | 1.772 bytes |

Antes: cada busca exigia Enter e uma navegação para a página completa, com HTML, CSS e scripts avaliados de novo.

Depois: enquanto se digita, só o fragmento é pedido, 120 ms depois da última tecla, cancelando a requisição anterior. O fragmento é cerca de 14 vezes menor que a página e leva metade do tempo no servidor.

`search.js`: 1.564 bytes comprimidos, dentro do orçamento de 4.096 verificado por `scripts/check_web_style.py`.
