---
id: doc-031
title: Estudo — reorganização do CSS e dos templates HTML
type: other
created_date: '2026-11-13T23:46:00Z'
updated_date: '2026-11-14T14:21:00Z'
tags:
  - estudo
---
**Tarefa:** ISSUE-061 · **Atualizado:** 14/11/2026 · **Plano:** doc-032, Fase 1. Sem Tailwind (ISSUE-058 / decision-006).

## Decisão do usuário

Em 14/11/2026 o usuário escolheu leitura do disco, fontes locais e incorporação no binário como opção futura. No esclarecimento sobre a localização, respondeu **“tudo em assets/”**, substituindo a proposta inicial de `templates/` na raiz.

**Contrato adotado para o planejamento:** todos os arquivos web em `assets/` na raiz do repositório; HTML em `assets/templates/`; CSS, JavaScript, ícones, temas e fontes no mesmo root. O runtime lê essa distribuição do disco em produção e desenvolvimento. A opção de embutir esses arquivos no binário será planejada como extensão futura, sem implementação antecipada.

Esta é uma decisão de arquitetura do estudo. A implementação e a mudança de distribuição pertencem às tarefas seguintes; nenhum arquivo de runtime foi movido nesta revisão. O plano anterior de `web.go` com embed imediato e root `templates/` + `assets/` foi substituído.

## Situação atual confirmada

Os templates e arquivos web hoje são embutidos: `internal/webui/preview.go:21`, `internal/adminui/admin.go:35`. O renderer público usa `ParseFS(files, "templates/*.html")` e lê SVGs do mesmo FS (`internal/webui/renderer.go`). Paletas também são carregadas do FS embutido. A imagem atual já copia `assets/`, mas os dois TTF desse diretório não são servidos como fontes web.

A medição histórica de 13/11/2026 encontrou 12 templates / 36 blocos, linhas de até 2.261 caracteres, `ui.css` com 28 KB e 1.199 linhas, e 18 media queries com 13 pontos de quebra. São a motivação para legibilidade e organização; a ISSUE-065 deve gerar medidas e referências novas antes de mover arquivos.

## Estrutura proposta

```text
assets/
  templates/
    public/       páginas públicas e prévia
    admin/        páginas administrativas
    partials/     head, navegação, cards, gráficos e diálogos
  css/
    00-layers.css
    01-reset.css
    02-tokens.css
    03-base.css
    04-layout.css
    components/
    pages/
    admin.css
    99-utilities.css
  js/
  icons/          SVGs e licença
  fonts/          fontes web e respectivas licenças
  public/         favicon e ícones de aplicativo
  themes/         palettes.json
  brand/          imagens da marca e do README
```

`assets/` é relativo à raiz do projeto, não um diretório obrigatório `/assets` na raiz do sistema operacional. Os catálogos externos de tradução continuam seguindo o contrato próprio de `langs/`.

Nomes de arquivo/bloco devem evitar colisões no `html/template`. Só os diretórios estáticos explicitamente autorizados são publicados: `templates/`, arquivos Go, licenças internas não destinadas a rota e `brand/` não ganham exposição automática. Preservar as URLs `/web-assets/...` e desabilitar listagem de diretórios. Não usar um servidor estático irrestrito sobre todo `assets/`.

## Contrato da leitura do disco — ISSUE-066

### Localização e validação

- `TUNETRAIL_WEB_DIR` aponta para o root `assets/`, não apenas para os HTML. É proposta de configuração ainda não implementada.
- Se o caminho for explícito e inválido, a partida falha com caminho/item ausente; não procurar outra cópia silenciosamente.
- Sem configuração explícita, procurar `assets/` ao lado do executável real (resolvendo symlinks). Resolver e fixar o caminho absoluto uma única vez; não depender do diretório corrente.
- `make api.dev`, `make ui.preview` e testes informam explicitamente o root do checkout/fixture, pois `go run` e Delve podem gerar executáveis temporários.
- Validar parse de todos os templates usados e referências a CSS, JS, paletas, SVGs e, após a 070, fontes. Arquivo obrigatório ausente ou inválido impede a partida; itens opcionais só são exigidos quando a funcionalidade correspondente está habilitada.

### Consistência e atualização

Produção lê o disco no startup e constrói **um snapshot em memória** contendo templates parseados, bytes dos assets servidos, paletas e hashes da mesma geração. O atendimento das requisições usa esse snapshot; não mantém CSS/JS/fontes sendo relidos de um diretório mutável enquanto templates e hashes ficam antigos. Isso continua sendo distribuição e leitura de arquivos externos, sem incorporação no executável.

Atualizações distribuem binário + assets da mesma versão e reiniciam o processo. Usar diretórios de release imutáveis e troca controlada, sem editar arquivos parcialmente durante a partida. A implementação deve medir o volume carregado e limitar a carga ao manifesto de runtime conhecido.

Em desenvolvimento, recarga é uma opção explícita: montar e validar um snapshot completo antes de trocá-lo atomicamente. Erros devem ficar visíveis sem servir uma mistura de gerações. Não adotar hot reload de produção implicitamente.

A ISSUE-106 precisa definir retenção/revalidação das URLs com hash de gerações anteriores. O snapshot por processo não promete que um novo processo ainda servirá assets de um release antigo. Coordenar esse contrato com o cache e com as próximas alterações da ISSUE-068.

### Distribuição

O binário isolado deixa de ser uma distribuição completa da interface: o pacote deve incluir `tunetrail`, `assets/` e os catálogos exigidos pelo contrato de idiomas. O Dockerfile deve copiar os assets para a **imagem final** e indicar o root, além de ajustar a etapa de build se necessário. Manter esses arquivos separados do banco e de diretórios graváveis.

A ISSUE-066 muda instalação/execução e exige o ciclo funcional e publicação de versão, mesmo que HTML e CSS permaneçam iguais. A equivalência visual não torna essa alteração neutra em runtime.

## Organização de CSS e templates

Na 067, separar páginas e parciais e compartilhar head/navegação quando a equivalência estiver comprovada. Usar indentação consistente e limitar linhas a 160 caracteres, tratando espaços entre elementos inline. Não misturar mudança tipográfica com reformatação.

Na 068, dividir CSS por função em camadas `reset, tokens, base, layout, components, pages, utilities`; concatenação em ordem determinística pelo Go, sem ferramenta de build adicional. Preservar o comportamento da cascata e coordenar hashes com a 106.

Na 069, alinhar pontos de quebra em 40, 64 e 90rem e usar container queries onde adequadas. Essa fase altera layout e exige comparação visual e aceite.

## Fontes locais e retrospectiva — ISSUE-070

Investigação reproduzível e fontes consultadas: `backlog/evidencias/issue-061/fontes-retrospectiva.md`.

Já existem dois arquivos estáticos: Jost Regular 400 (61.524 bytes) e League Spartan Medium 500 (57.540 bytes), total 119.064 bytes. Não foram encontrados Inter, WOFF/WOFF2 ou textos OFL separados no projeto. Ambos os TTF declaram OFL 1.1 em metadados; incluir os documentos de licença na distribuição antes de reutilizá-los. Os arquivos atuais não cobrem sozinhos os pesos e estilos necessários.

As páginas de referência e TuneTrail declaram Jost/League Spartan e obtêm fontes pelo Google Fonts. A diferença confirmada não depende de supor uma falha de carregamento: no TuneTrail, o vencedor herda de h3 peso 400 e uppercase; na referência usa peso 600 sem uppercase. O título tem 42 px contra 36 px e os rótulos 18 px / 600 contra 16 px / 400. O tamanho do álbum da referência também pode variar com a extensão do título.

A implementação deve:

1. Inventariar pesos/itálicos realmente utilizados nas páginas públicas, prévia e administração; obter arquivos locais com origem/hash/licença e cobertura de acentos portugueses. Preservar as famílias existentes (Jost, League Spartan e Inter como fallback usado) salvo decisão posterior.
2. Preferir WOFF2, estático ou variável conforme os pesos utilizados, sem declarar o TTF Regular/Medium atual como variável. Definir `@font-face` com peso/estilo corretos e `font-display: swap` em `assets/css/fonts.css`; fontes e licenças em `assets/fonts/`.
3. Remover links, preconnects e imports de Google Fonts de todos os heads/CSS, mantendo rotas locais coerentes com a 066/106.
4. Corrigir especificamente os estilos da retrospectiva para o alvo observado, sem transformar todos os h3 do aplicativo. Confirmar a regra original para título de álbum longo, hierarquia, nomes, números, labels e controles. Só servir fontes locais não corrige peso, caixa e tamanho.
5. Verificar fonte efetivamente renderizada, acentos e caracteres faltantes, ausência de negrito artificial e ausência de dependência de rede externa para fontes. Não anunciar que capas e integrações do aplicativo inteiro funcionam offline.
6. Conferir consumidores dos TTF antes de convertê-los/movê-los/removê-los. O protótipo de geração de imagem está desativado e recebe bytes do chamador; qualquer reativação futura precisa considerar o novo caminho.

## Sequência de execução e validação

| Issue | Escopo atualizado | Verificação principal |
| --- | --- | --- |
| 065 | Referência do Go atual e alvo de tipografia da retrospectiva | HTML/CSS reproduzíveis, capturas em seis larguras e dois temas; referência e destino no mesmo tema/viewport/dados fictícios |
| 066 | Root único `assets/`, leitura externa, snapshot e distribuição | Equivalência HTML/CSS; execução fora do cwd; root explícito inválido; arquivos ausentes; templates inválidos; symlinks; binário+assets e imagem final; gerações consistentes |
| 067 | Templates/parciais legíveis | HTML equivalente após normalizar espaços e capturas |
| 068 | CSS por função/camadas e concatenação | Ordem/cascata equivalentes e integração com cache |
| 069 | Tokens, pontos de quebra e container queries | Capturas antes/depois e aceite visual |
| 070 | Fontes locais e correção tipográfica da retrospectiva | Fontes/estilos/pesos/licenças; rede externa bloqueada para fontes; capturas antes/depois e teste de não regressão |

Não aplicar a exigência de CSS byte a byte idêntico da mudança de lugar à correção visual deliberada da 070. A fonte local deve ser comparada após carregar, evitando medir um fallback transitório.

## Opção futura: arquivos embutidos

Preparar a fronteira de leitura com `fs.FS` e injeção da fonte de arquivos, usando o disco na implementação atual. Não adicionar provider embed, build tag, cópia duplicada de templates ou fallback silencioso agora.

Quando essa opção for escolhida, criar uma tarefa específica que compare build selecionável (disco ou embed) e modo selecionável de distribuição, defina precedência e erros, incorpore somente arquivos de runtime e licenças e use a mesma suíte de renderização nos dois modos. Os bytes e os hashes servidos devem ter o mesmo contrato. O modo disco permanece disponível conforme decisão do usuário. Planejar também pacote standalone e Docker para cada modo, sem prometer hoje um executável autossuficiente.

## Pareceres e limites

A primeira chamada ao Claude Code CLI, dentro da sandbox, retornou `Not logged in`. Em 14/11/2026, a investigação da invocação confirmou que a mesma sessão é reconhecida fora da sandbox; o teste real ao Opus respondeu `ADVISOR_OK`. A revisão documental pelo Claude Code CLI / Opus (effort medium) foi então concluída: **REPROVADO COM RESSALVAS**, registrada em `backlog/evidencias/issue-061/parecer-claude-opus.md`. Os ajustes propostos nesse parecer ainda não foram aplicados ao desenho. A revisão anterior do Advisor @Sol aprovou o contrato de snapshot/root/deploy; o novo parecer acrescenta ressalvas de documentação, critérios de aceite e validação. O aceite continua reservado ao usuário.

As duas páginas foram inspecionadas em modo somente leitura, com estilos computados e CSS obtidos por HTTP. Não se alterou o tema da instalação, não foram medidos glifos efetivamente renderizados e não foi executada uma nova matriz visual completa. Este estudo não altera runtime nem requer testes de aplicação; os testes e capturas estão discriminados nas tarefas de implementação.
