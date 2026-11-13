---
id: doc-006
title: Estudo — interface multilíngue (item 3 do roadmap)
type: specification
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - roadmap
---
> Importado de `docs-dev/ROADMAP_03_UI_MULTILINGUE.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Estudo — interface multilíngue (item 3 do roadmap)

Data: 10/11/2026. Estado: **issue 050 fechada e aprovada; 051 fechada e aprovada; 052–054 implementadas e aguardando teste funcional e aceite**. Não confundir com a etapa histórica 3 de migração de mídia. [Roadmap](<doc-001 - ROADMAP.md>) · [Issues locais](<../historico/doc-025 - Índice-histórico-das-issues-locais-001–058.md>).

## Recomendação

Permitir a cada visitante escolher o idioma da interface pública e administrativa, com renderização Go/HTML, sem exigir JavaScript ou serviço externo. Propõe-se iniciar com **inglês e português brasileiro (pt-BR)** e inglês como fallback, preservando o comportamento atual. A seleção inicial de idiomas é uma proposta para validação, não uma decisão definitiva.

Recomenda-se testar **[go-i18n/v2](https://github.com/nicksnyder/go-i18n)** com catálogos de mensagens externos no diretório `langs` e [`golang.org/x/text/language`](https://pkg.go.dev/golang.org/x/text/language) para seleção de idioma. A biblioteca oferece plurais CLDR, parâmetros nomeados e arquivos JSON/TOML/YAML; possui licença MIT. Ela não traduz automaticamente: textos precisam ser escritos e revisados. A escolha depende de uma prova de conceito que avalie compatibilidade, custos, validação e fluxo de tradução; a prova de conceito adotou `go-i18n/v2` v2.5.1.

## Inventário do projeto

- A UI pública e a administração usam conjuntos distintos de `html/template`, cada um com seis arquivos HTML, incorporados com `go:embed`: `internal/webui/renderer.go`, `internal/webui/preview.go` e `internal/adminui/admin.go`.
- Há textos também nos modelos Go (`home.go`, `handlers.go`, `rewind.go`), erros e notificações em `internal/adminui` e `internal/adminweb`, e nos scripts `ui.js`, `live.js` e `admin.js`. Incluir rótulos acessíveis, mensagens de erro, títulos, `alt`, placeholders e estados vazios.
- **Risco real:** `internal/webui/templates/live.html` usa `.Title` igual a `Top tracks` e `Top albums` para controlar fluxo e CSS e usa `.Label` como chave do período. Traduzir essas strings diretamente quebra layout ou seleção. Separar identificadores estáveis de rótulos antes de traduzir.
- Datas e tempo relativo são montados em Go com formatos fixos e nomes de mês ingleses; há pluralização manual. `time.Month.String()[:3]` não é uma abreviação localizada segura. Localizar conteúdo de gráficos, datas e números preservando ISO, timestamps, coordenadas, atributos de máquina e parâmetros de formulários.
- Idioma e fuso são independentes. `live.js` já consulta o navegador para fuso e início da semana. A escolha do idioma não deve mudar cálculos, calendário, DST ou comportamento da issue 018.
- `golang.org/x/text v0.38.0` já consta como dependência indireta. O projeto declara Go 1.25 e o Docker usa Go 1.25. A versão [go-i18n v2.6.1](https://github.com/nicksnyder/go-i18n/blob/v2.6.1/go.mod) declara Go 1.24, enquanto a [branch principal observada](https://github.com/nicksnyder/go-i18n/blob/main/go.mod) declara Go 1.26. Fixar versão só após teste real de resolução e build; não instalar `@latest` automaticamente.
- Os fluxos principais de páginas públicas e admin usam `no-store`. Examinar todos os caminhos antes de planejar cache localizado. SQLite permanece no esquema consolidado v3; preferência por navegador não exige migração de banco.

## Comparação

| Opção | Benefícios | Limites a verificar |
| --- | --- | --- |
| [go-i18n/v2](https://github.com/nicksnyder/go-i18n) | Catálogos, plurais CLDR, variáveis nomeadas, fluxo de merge, MIT. | Nova dependência; a extração oficial menciona literais `i18n.Message` em Go. Não presumir extração automática de templates HTML ou JavaScript. |
| [x/text/message](https://pkg.go.dev/golang.org/x/text/message) e [language](https://pkg.go.dev/golang.org/x/text/language) | Pacote do módulo já indireto; seleção de tags e catálogos. O [comando gotext](https://pkg.go.dev/golang.org/x/text/cmd/gotext) ajuda com código Go. | Demonstrar plural, catálogo, fluxo para tradutores, cobertura dos HTML/JS e tamanho do binário. Dependência indireta não significa custo adicional zero. |
| [leonelquinteros/gotext](https://github.com/leonelquinteros/gotext) | Catálogos gettext/PO e pluralização. | Outra biblioteca e workflow PO; provar extração de templates e manutenção. Não confundir com o comando `golang.org/x/text/cmd/gotext`. |
| Catálogo próprio | Controle e poucas dependências. | O projeto passa a manter negociação, plurais, fallback, placeholders e validação; risco cresce com os idiomas. |

## Contrato proposto

1. Idioma por requisição: escolha explícita em cookie do navegador > `Accept-Language` compatível > inglês. Aceitar apenas tags suportadas; tratar limites e entradas inválidas. Decidir na prova de conceito o casamento `pt`/`pt-BR`. O cookie pertence ao navegador, não ao perfil de escuta. Seletor por formulário sem JavaScript, retorno local validado e proteção de origem/CSRF conforme desenho adotado.
2. Definir `<html lang>` para UI pública, login e admin; avaliar `Content-Language` e `Vary` sem enfraquecer `no-store`. Nomes de artistas, faixas, álbuns e aliases, URLs, parâmetros, `data-*`, JSON de API e formatos de importação são dados/protocolo e não serão traduzidos.
3. Catálogos externos carregados uma vez no início; tradutor e idioma por requisição. Não alterar `FuncMap` de um template compartilhado nem idioma global durante a requisição. Usar strings comuns e [escaping contextual de html/template](https://pkg.go.dev/html/template); jamais marcar traduções ou dados de usuário como `template.HTML`.
4. Usar chaves estáveis e mensagens completas, com parâmetros nomeados e plurais CLDR; não concatenar frases em ordem inglesa. Manter inventário para textos HTML/JS que a extração Go não identifica. Testar fallback, chaves ausentes, placeholders e catálogos inválidos; não suprimir erros de inicialização sem diagnóstico.
5. Separar locale de apresentação dos cálculos de fuso e dos formatos de máquina. Definir exemplos pt-BR/en para singular/plural, tempo relativo, datas, meses e números. Biblioteca de mensagens não resolve automaticamente todas as convenções culturais.

## Plano proposto

1. [050 — fundação e escolha de idioma](<../../tasks/issue-050 - 050-—-Fundação-e-escolha-de-idioma.md>): prova de conceito, catálogos, negociação, seletor sem JS e identificadores estáveis.
2. [051 — interface pública](<../../tasks/issue-051 - 051-—-Internacionalizar-interface-pública.md>): perfis, navegação, busca, rankings, mídia e retrospectiva.
3. [052 — administração](<../../tasks/issue-052 - 052-—-Internacionalizar-administração.md>): login, formulários, mensagens, erros e scripts essenciais.
4. [053 — datas e números](<../../tasks/issue-053 - 053-—-Localizar-datas-quantidades-e-unidades.md>): plurais, tempo relativo, unidades e acessibilidade de gráficos.
5. [054 — integração e aceite](<../../tasks/issue-054 - 054-—-Integração-e-aceite-multilíngue.md>): tradução revisada, CI, concorrência, builds e comparação visual.

As issues são locais; a 050 está fechada e aprovada, a 051 está fechada e aprovada, e 052–054 estão implementadas e aguardam teste funcional e aceite, sem publicação automática. A ordem pode ser revista após a POC; a 042 mantém seu próprio aceite de configuração. Não assumir que toda a interface foi traduzida antes do aceite.

## Validação necessária

Cobrir pt-BR/en, fallback, cookie ausente, tags inválidas, preferências e dois navegadores simultâneos; teste `-race`; tradução sem JS; compatibilidade de acessibilidade, CSRF, autenticação, links, filtros, perfis e API; placeholders e escaping; datas/DST sem mudança de timestamps; mensagens de falha e sucesso; temas claro/escuro, zoom, desktop e celular com capturas; catálogo completo, binário sem CGO, Docker/CI e documentação para adicionar idiomas. Não usar dados reais em capturas.

## Revisão

Advisor/Sol analisou o código e apontou os riscos de strings usadas como lógica, estado global por idioma, pipelines separados de administração, cobertura de JS/ARIA e diferença entre idioma e fuso. O plano acima incorpora essas ressalvas. A revisão é do estudo, não aprovação de código futuro.

Fontes primárias: [go-i18n](https://github.com/nicksnyder/go-i18n), [x/text/message](https://pkg.go.dev/golang.org/x/text/message), [x/text/language](https://pkg.go.dev/golang.org/x/text/language), [html/template](https://pkg.go.dev/html/template), [gotext](https://github.com/leonelquinteros/gotext).

## Decisão do usuário — catálogos externos

Os catálogos ficam fora do executável em `langs/`. `TUNETRAIL_LANGS_DIR` pode indicar outro diretório; o container usa `/app/langs`. Sem variável, procura primeiro `langs` ao lado do executável e depois a raiz do checkout Go. Os arquivos são carregados na inicialização; alterações exigem reinício. Inglês é fallback por mensagem. Arquivos ausentes/inválidos impedem o startup com erro. Os arquivos suportam mensagens simples e formas de plural CLDR com parâmetros nomeados; zero é apresentado no plural.
