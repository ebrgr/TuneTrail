# ISSUE-124 — edição sem recarregar

Medido por `backlog/evidencias/issue-124/edicao_sem_recarregar.py` (Chrome, servidor local, dados fictícios).

Antes: cada ação era POST, redirecionamento 303 e GET da página inteira de gestão (27.773 bytes nesta faixa de teste), voltando ao topo e perdendo o foco.

Depois, com JavaScript, cada ação recebe só a seção alterada:

| Seção devolvida | Tamanho |
|---|---:|
| `aliases` | 2.706 bytes |
| `aliases` | 2.706 bytes |
| `listens` | 3.490 bytes |
| `artists` | 2.849 bytes |

Ou seja, de 8 a 10 vezes menos HTML por ação, sem nova avaliação de CSS e scripts, com a rolagem e o foco mantidos. Sem JavaScript a ação continua sendo POST, 303 e GET, mas volta à âncora da seção editada.

A lista de escutas só é consultada quando a seção de escutas é renderizada (página completa ou ação de escuta).
`enhance.js`: 1.759 bytes comprimidos, dentro do orçamento de 4.096 verificado por `scripts/check_web_style.py`.
