# ISSUE-073 — implementação e verificação

Verificação do executor Codex em 18/11/2026. Status permanece **Em andamento**; parecer do Reviewer e aceite visual do usuário pendentes.

## Implementação

- Histórico compartilhado pela página inicial, listagem e páginas de mídia: capa, título em até duas linhas com nome completo no link, artistas e horário relativo (datas antigas mantêm a apresentação absoluta existente, com instante completo no `time`).
- Reticências com `details`/`summary` nativos, link normal sem scripts e ação de confirmação existente com scripts. A melhoria progressiva permanece dentro da mesma célula, sem criar uma quarta coluna; após excluir, o foco retorna ao resumo da próxima linha.
- Cabeçalho de artista/álbum/faixa mantém capa grande, tipo, título, ranking, contagens e primeira escuta. A média RGB é extraída da imagem orientada/recortada no processamento do WebP e salva em `average-color` junto à imagem. Caches antigos recebem metadados na primeira leitura, com o processamento limitado existente; ausência/erro usa cor neutra.
- A cor é escurecida até luminância ≤0,1: texto branco tem contraste ≥7:1; o degradê só acrescenta preto. A fixture produziu `#4e5950`, contraste de 7,32:1 antes do degradê. A barra de ações conserva URLs, ícones, comportamento e superfície do tema; nenhuma página de gestão foi alterada.
- Cinco links de seção, sem funções de abas em JavaScript. Artista usa âncoras locais; faixa liga Faixas ao álbum e Álbuns ao álbum correspondente; em álbum, Álbuns retorna ao próprio cabeçalho.
- Referência determinística em `reference/`, usada por `make ui.reference.check`. A referência histórica da ISSUE-072 foi preservada. Os arquivos de gestão dessa referência são exportações da fixture, não alterações nas páginas de gestão.

## Evidências

Navegador embutido do Orca, dados fictícios de `scripts/serve_roadmap06.py`, arquivos e banco descartáveis em `/tmp`. A fixture recebeu uma escuta de dois minutos atrás para verificar o horário relativo. Nenhum dado real foi usado.

- `history-320-keyboard.png`: menu aberto com Enter, ação alcançada por Tab; título longo e caracteres japoneses, largura de documento e viewport de 320px.
- `history-320-light-keyboard.png`: mesma checagem em Pearl. Texto e contorno de foco medidos em 17,17:1; Yuu em 14,49:1 (`browser-checks.json`).
- `track-desktop-dark.png`, `track-desktop-light.png`: 1440×1000, capa de 384px, ranking, metadados e degradê.
- `artist-320-no-script.png`: cabeçalho e cinco links em 320px, scripts da página bloqueados.
- `history-320-no-script.png`: menu e link normal operáveis por teclado sem scripts.
- `no-script-tabs.json`: os cinco links ativados com Enter, hash correto e ausência de overflow em 320px. Para bloquear scripts, um proxy HTTP local retransmitiu os GETs da fixture acrescentando `Content-Security-Policy: script-src 'none'`; `window.TuneTrailEnhance` permaneceu `undefined`. A instrumentação do navegador apenas leu estado/estilos e enviou teclas.
- Exclusão real na fixture: sete para seis linhas, diálogo fechado, foco nas reticências seguintes, documento com 320px. Registrado em `browser-checks.json`.

## Testes

- `make check`: aprovado; Go vet e testes, verificações de UI/JavaScript/CSS, referência determinística e **13/13 jornadas E2E** (`make-check.log`).
- Novos testes: média RGB/alpha/bounds, processamento do WebP com cor, gravação e recuperação de metadados legados, fallback/corrupção/remoção, contraste em 4.096 cores, cinco destinos em artista/álbum/faixa e histórico com/sem administrador/escaping/URL de retorno.
- `scripts/validate_listen_dialogs.cjs` adaptado à abertura das reticências e foco no resumo, incluindo 320px; sintaxe verificada. A matriz Playwright completa desse script não foi executada; as provas de navegador acima foram feitas pelo Orca.
- `git diff --check`: aprovado. Revisão própria do diff com deslop; sem delegação e sem parecer de Reviewer emitido.
- Grafo AST atualizado com `analisar-tunetrail.sh "$PWD"`; 4.121 nós e 17.982 arestas. Saídas externas ao Git. Avisos esperados: `tokens.css` excluído pelo detector de nomes sensíveis e cópias de scripts sem símbolos nos exports da referência.

Comparação visual baseada na descrição do Koito em `backlog/evidencias/ui-responsiva/comparacao-koito-react.md`: capa grande, hierarquia tipográfica, posição no ranking, metadados e degradê. Não foi feito novo acesso à instância do Koito. O aceite visual permanece com o usuário.
