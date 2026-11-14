# ISSUE-065 — Referência da interface

Data: 14/11/2026. Referência do TuneTrail 0.10, sem mudança de runtime. Somente dados fictícios, SQLite em memória e configuração independente do ambiente operacional.

## Reproduzir

```sh
make ui.reference OUT=/tmp/tunetrail-reference-new
make ui.compare BASE=backlog/evidencias/ui-responsiva/base/reference NEW=/tmp/tunetrail-reference-new
python3 -m unittest discover -s scripts -p test_compare_ui_reference.py
PLAYWRIGHT_MODULE=/caminho/node_modules/playwright node scripts/capture_ui_base.cjs /tmp/tunetrail-reference-new /tmp/tunetrail-capturas
```

O diretório de saída da exportação deve ser novo. Chrome instalado é usado em modo headless; `CHROME_PATH` permite outro Chromium. Playwright é dependência apenas da ferramenta de captura, instalada separadamente, não da aplicação. Nesta execução foi instalado em `/tmp`, sem alterar dependências do projeto.

## Conteúdo e critérios

`reference/` contém 23 páginas HTML, CSS/JS, paletas, arquivos públicos, ícones, avatar e cinco capas SVG da prévia. `routes.json` lista rotas e status. Os 113 arquivos foram comparados em duas gerações independentes por `make`, iniciadas com TZ externos Honolulu e Tóquio; a receita fixa UTC. Resultado: **igualdade byte a byte, código 0**.

O comparador verifica todos os arquivos, inclusive fontes e outros binários que venham a ser exportados. Código 1 identifica somente indentação entre tags e não autoriza aceite visual; código 2 indica mudança de conteúdo/conjunto de arquivos. Não remove comentários, ordena regras CSS nem elimina espaços inline. Versão do aplicativo continua sendo conteúdo: uma publicação pode produzir diferença legítima a analisar.

As capturas cobrem 11 páginas representativas (prévia, seleção, home, artista, retrospectiva, login, perfis, conta, configuração, editor e confirmação de merge) × seis larguras (320,360,390,768,1024,1440) × dois temas (Yuu/Pearl). Altura 900, escala 1, locale en-US, fuso UTC e relógio JS fixo. A retrospectiva usa full-page para incluir o conteúdo abaixo da dobra no celular. Os outros HTMLs cobrem rankings, histórico, álbum, faixa e confirmação de exclusão, sem uma captura redundante de cada combinação.

`capturas/manifest.json` registra arquivos, viewport, tema, navegador, overflow e falhas de fontes. A tipografia da retrospectiva é registrada em 390/768/1440, nos dois temas, para seis seletores: título, ano, rótulo, vencedor, números e controles. Cada entrada contém estilos computados e as fontes efetivamente usadas, obtidas via `CSS.getPlatformFontsForNode` do Chrome. Isto documenta o estado atual; não aprova um novo alvo tipográfico da ISSUE-070.

## Determinismo e limites

- Páginas públicas usam relógio de 01/06/2024 e escutas de maio de 2024. Todos os registros foram criados em SQLite em memória.
- A fixture do gráfico de interesse consulta a contagem real do store e a apresenta em um intervalo agregado fixo. Testa renderização; não representa a forma completa da distribuição temporal. Os testes de agregação existentes continuam responsáveis por essa lógica.
- A prévia usa seus templates reais, com células de calendário fixadas na fixture de teste. Não foi adicionado relógio nem modificado handler de produção para isso.
- Os campos ocultos `csrf_token`, `submission_token` e `plan` são substituídos por `REFERENCE_TOKEN` após renderizar. Formulários exportados são evidência visual, não formulários utilizáveis. Não são guardados cookies, credenciais reais ou banco.
- O teste limpa variáveis operacionais `TUNETRAIL_*`/`KOITO_*`; o Makefile exporta o `.env`, e isso não deve interferir na referência.
- Fontes Google continuam permitidas na captura porque o runtime atual as usa. As falhas e a fonte real ficam registradas; a rede externa não é garantia de determinismo de pixels. Não houve substituição por fontes locais nem alteração do CSS. Imagens sem capa usam o fallback existente.
- O servidor de captura lê somente a exportação em loopback efêmero. Falta de CSS/JS/imagem local gera erro, salvo a tentativa de `/image/*` cuja ausência aciona o fallback esperado. Capas e avatar da prévia estão incluídos.
- Capturas não são teste automático de igualdade de pixels entre versões de Chrome/OS. A comparação visual das próximas tarefas deve usar condições equivalentes.

## Revisão e verificações

O planejamento e a revisão foram feitos pelo Claude Code CLI / Opus, effort medium. A revisão pediu calendário fixo da prévia, exportação de assets dinâmicos/públicos, comparação de todos os arquivos, testes no `ui.check` e UTC na geração. Correções realizadas. O parecer final aprova condicionado a UTC; essa condição foi aplicada e verificada com dois fusos externos distintos.

`make check` aprovado (Go/vet/JS/build e 11/11 e2e), testes do comparador incluídos em `ui.check`, comandos públicos de geração/comparação exercitados, e `git diff --check` sem erros. Alterações são ferramentas/testes/evidências: sem migração, publicação, push ou mudança na aplicação. A ISSUE-061 continua aguardando seu próprio aceite; gerar esta base atual não decide o alvo visual futuro.
