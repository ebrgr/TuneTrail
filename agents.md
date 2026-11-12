# TuneTrail — working notes for agents

TuneTrail registra o histórico de músicas (scrobbles), recebe eventos pela API
compatível com ListenBrainz e apresenta estatísticas em uma interface web.
O remoto `origin` aponta para o repositório TuneTrail. Não é necessário Docker nem PostgreSQL para desenvolver.

Este arquivo serve como um guia de *como trabalhar neste repositório*. Para evitar que o conteúdo fique defasado, ele propositalmente **não** repete informações documentadas em outros lugares — consulte esses documentos e, em seguida, siga as convenções abaixo.

# Subagentes Disponíveis
**`/docs-dev/agents/advisor.md`** - Quando o Codex Terra identificar problemas de lógica, concorrência ou algoritmos de alta densidade, ele deve sugerir ao usuário a delegação para o subagente @Sol com a persona do Advisor.

**`/docs-dev/agents/deslop/SKILL.md`** - Ler a Skill para gerar código melhor quando possível

**`/docs-dev/agents/deslop/ui-craft/SKILL.md`** - Orientações para formulação de UI.

**graphify** - Use para manutenção do código

## Regras de Delegação para o @Sol (Advisor)
- Para revisões de arquivos de configuração, rotas e CRUDs simples: delegar para `@sol` com effort=low`.
- Para revisões de lógica de criptografia, manipulação financeira, sistemas de mensageria ou algoritmos matemáticos: delegar obrigatoriamente para `@sol` com `effort=medium`.


## Leia isto primeiro (não duplique o conteúdo aqui)

- **`README.md`** — a apresentação: o que é o Koito, como ele se posiciona em relação ao salvamento de feeds de música. Ele utiliza links externos em vez de fornecer explicações detalhadas; mantenha esse formato.

- **Documentação do usuário** — `docs/ contém informações sobre o aplicativo e como configurar o seu uso. O histórico de versões deve constar em `CHANGELOG.md`.

- **`docs-dev/`** — Informações sobre o desenvolvimento do aplicativo e estudo de como ele funciona e das próximas alteações que vamos fazer. `docs-dev/Koito_SUM_Analise_1a_análise.md` das alterações que vamos desenolver. `docs-dev/ROADMAP.md` contém o plano geral de execução. Sempre que alterar algo por exemplo passo 1. Após as alterações e testes faça um resumo do que foi alterado.


## Convenções de trabalho

- **Nunca realize um *push* ou publicação sem autorização explícita.** Sinta-se à vontade para fazer *commits*; Qualquer ação que envolva alterações no sistema depende sempre da decisão do usuário — pergunte e aguarde. Uma *skill* (procedimento/tarefa) que mencione `git push` em suas etapas está apenas documentando o processo, não autorizando a execução.
A única exceção permanente: uma vez que um *branch* de um *Pull Request* (PR) tenha sido aprovado para envio (*push*), correções para falhas de CI nesse PR podem ser enviadas. A operação de *merge* sempre exige uma autorização explícita à parte. Consulte a *skill* **source-control** para entender os limites, o que mais não é considerado autorização, além das diretrizes sobre *staging* e reescrita de histórico.
- **Prioridade para *Issues* (*Issue-first*):** demandas substanciais ou funcionais (correções de *bugs*, novas funcionalidades, mudanças de comportamento, refatorações não triviais) devem ser registradas como uma *issue* no GitHub *antes* de escrever o código, e encerradas por meio do *commit* (usando `Fixes #N`). Apenas alterações que não alteram o comportamento em tempo de execução (*runtime*) — como formatação, *linting*, comentários ou testes — estão isentas disso; consulte a *skill* `issue-first` para entender o critério de distinção.
- **Registro de alterações (*Changelog*):** quando uma mudança for visível ao usuário, adicione uma linha ao arquivo `CHANGELOG.md`, sob a seção `[Unreleased]`, no mesmo *commit* — consulte a *skill* `changelog` para saber o que deve ser incluído e como redigir o texto.
- **Comente o código em inglês:** sempre que fazer uma mudança em alguma função deixe um comentário no código para posterior validação.

## graphify

This project has a knowledge graph at /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- When /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json exists, run `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify query "<question>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` before your first grep, search or source-file read for a codebase question, and put this rule in every subagent prompt that explores code. Read raw files only for lines the graph or an earlier read already located. Use `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify path "<A>" "<B>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` for relationships and `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify explain "<concept>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `/Users/evertonberger/.local/share/tunetrail-graphify/analisar-tunetrail.sh` to keep the graph current (AST-only, no API cost).
- A committed /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json is rewritten whole on every build, so a large diff to it in a pull request is expected. Do not flag it as churn or suggest skipping the graph rebuild or its hooks.
