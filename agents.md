# TuneTrail — working notes for agents

TuneTrail registra o histórico de músicas (scrobbles), recebe eventos pela API
compatível com ListenBrainz e apresenta estatísticas em uma interface web.
O remoto `origin` aponta para o repositório TuneTrail. Não é necessário Docker nem PostgreSQL para desenvolver.

Este arquivo serve como um guia de *como trabalhar neste repositório*. Para evitar que o conteúdo fique defasado, ele propositalmente **não** repete informações documentadas em outros lugares — consulte esses documentos e, em seguida, siga as convenções abaixo.

# Subagentes Disponíveis
Os agentes e skills ficam em **`/.agents/`**, diretório local mantido fora do Git (`.git/info/exclude`). Não adicione esses arquivos a commits.

**`/.agents/advisor.md`** - Persona do Advisor. Quando o Codex Terra ou o Claude Sonnet identificar problemas de lógica, concorrência ou algoritmos de alta densidade, ele deve sugerir ao usuário a delegação para o subagente, @Sol no Codex ou @Opus no Claude, essa persona.

**`/.agents/golang-master/SKILL.md`** - Escrever ou revisar Go de produção: erros, concorrência, `context`, tipos, genéricos, testes, benchmarks e estrutura de módulos.

**`/.agents/deslop/SKILL.md`** - Revisar um diff em busca de complexidade acidental ou ruído de código gerado; ler a skill para gerar código melhor quando possível.

**`/.agents/issue-first/SKILL.md`** - Ciclo de uma tarefa do Backlog.md: registrar, iniciar a branch, enviar para aceite e publicar com `make issue.*`.

**`/.agents/changelog/SKILL.md`** - Quando e como escrever entradas no `CHANGELOG.md`.

**`/.agents/ui-craft/SKILL.md`** - Orientações para projetar, construir e revisar UI: páginas, componentes, formulários e estados de carregamento e erro.

**graphify** - Use para manutenção do código


## Regras de Delegação para o @Sol (Advisor)
- Para revisões de arquivos de configuração, rotas e CRUDs simples: delegar para `@sol` com effort=low`.
- Para revisões de lógica de criptografia, manipulação financeira, sistemas de mensageria ou algoritmos matemáticos: delegar obrigatoriamente para `@sol` com `effort=medium`.


## Regras de Delegação para o @Opus (Advisor)
- Para revisões de arquivos de configuração, rotas e CRUDs simples: delegar para `@Opus` com effort=medium`.
- Para revisões de lógica de criptografia, manipulação financeira, sistemas de mensageria ou algoritmos matemáticos: delegar obrigatoriamente para `@Opus` com `effort=high`.

## Leia isto primeiro (não duplique o conteúdo aqui)

- **`README.md`** — a apresentação: o que é o TuneTrail, como ele se posiciona em relação ao salvamento de feeds de música. Ele utiliza links externos em vez de fornecer explicações detalhadas; mantenha esse formato.

- **Documentação do usuário** — `docs/ contém informações sobre o aplicativo e como configurar o seu uso. O histórico de versões deve constar em `CHANGELOG.md`.

- **`backlog/`** — Registro oficial de issues, roadmaps, estudos e decisões, gerenciado pelo [Backlog.md](https://github.com/MrLesk/Backlog.md). Veja a seção **Registro do projeto — Backlog.md** abaixo. Após as alterações e testes, faça um resumo do que foi alterado e registre-o na tarefa correspondente.

- **`docs-dev/`** — Removido em 12/11/2026. Issues, roadmaps, estudos e PRs estão em `backlog/`; as evidências (capturas e relatórios) em `backlog/evidencias/`; os agentes em `/.agents/`. O conteúdo original permanece no histórico do Git.


## Convenções de trabalho

- **Nunca realize um *push* ou publicação sem autorização explícita.** Sinta-se à vontade para fazer *commits*; Qualquer ação que envolva alterações no sistema depende sempre da decisão do usuário — pergunte e aguarde. Uma *skill* (procedimento/tarefa) que mencione `git push` em suas etapas está apenas documentando o processo, não autorizando a execução.
A única exceção permanente: uma vez que um *branch* de um *Pull Request* (PR) tenha sido aprovado para envio (*push*), correções para falhas de CI nesse PR podem ser enviadas. A operação de *merge* sempre exige uma autorização explícita à parte. Consulte a *skill* **source-control** para entender os limites, o que mais não é considerado autorização, além das diretrizes sobre *staging* e reescrita de histórico.
- **Prioridade para *Issues* (*Issue-first*):** demandas substanciais ou funcionais (correções de *bugs*, novas funcionalidades, mudanças de comportamento, refatorações não triviais) devem ser registradas como tarefa `ISSUE-NNN` em `backlog/` *antes* de escrever o código, feitas na branch `<tipo>/issue-NNN-slug` com commits prefixados por `ISSUE-NNN: ` e, após o aceite do usuário (Done), publicadas com `make issue.publish ISSUE=N`, que incrementa a versão, fecha a seção do `CHANGELOG.md` e cria a tag local (push só com `PUSH=1`, autorizado pelo usuário). Apenas alterações que não alteram o comportamento em tempo de execução (*runtime*) — como formatação, *linting*, comentários ou testes — estão isentas disso; consulte a *skill* `issue-first` para entender o critério de distinção.
- **Registro de alterações (*Changelog*):** quando uma mudança for visível ao usuário, adicione uma linha ao arquivo `CHANGELOG.md`, sob a seção `[Unreleased]`, no mesmo *commit* — consulte a *skill* `changelog` para saber o que deve ser incluído e como redigir o texto.
- **Comente o código em inglês:** sempre que fazer uma mudança em alguma função deixe um comentário no código para posterior validação.

## Registro do projeto — Backlog.md

Ferramenta instalada no sistema (`brew install backlog-md`); a configuração fica em `backlog/config.yml`. Em agentes, use sempre `--plain` (ou `--json`) para obter saída não interativa. Recomenda-se `git config core.quotepath false`, pois os nomes dos arquivos contêm acentos.

| Conteúdo | Onde | Comando |
| --- | --- | --- |
| Issues | `backlog/tasks/` — IDs `ISSUE-NNN` continuam a numeração local (001–058 importadas) | `backlog task create "NNN — Título" -d "..." -s Aberta -l <área> --type <tipo> -m "<milestone>" --plain` |
| Roadmaps | `backlog/docs/roadmaps/` e milestones | `backlog doc create "Título" -p roadmaps -t specification` · `backlog milestone add "Nome"` |
| Estudos | `backlog/docs/estudos/` | `backlog doc create "Título" -p estudos -t other`, depois `backlog doc update doc-NNN --content "..." --tags estudo` |
| Design e contratos | `backlog/docs/design/` | idem, com `-p design` |
| Descrições de PR | `backlog/docs/prs/` | idem, com `-p prs` |
| Evidências (capturas, relatórios, fixtures de comparação) | `backlog/evidencias/<issue ou roadmap>/` | arquivos comuns; não são lidos pelo Backlog.md — ligue-os à tarefa com `--add-ref backlog/evidencias/...` |
| Decisões de arquitetura | `backlog/decisions/` | `backlog decision create "Título" -s proposed` (`accepted` somente com aprovação do usuário) |
| **Estudos privados** | `backlog/docs/estudos-locais/` | idem, com `-p estudos-locais` |

**Ciclo de uma issue:** `Aberta → Planejada → Em andamento → Aguardando aceite → Done`.

- Toda mudança de status leva um comentário que explique o motivo: `backlog task edit ISSUE-NNN -s "Aguardando aceite" --comment "Verificação: ..." --comment-author <agente>`. Use comentários para verificação, refinamento, parecer do Advisor e aceite; não reescreva o histórico anterior.
- `Done` somente após aceite explícito do usuário, registrado em comentário. O nome `Done` é necessário para que milestones e estatísticas do Backlog.md contem a tarefa como concluída.
- Critérios de aceite com `--ac "..."`; marque com `--check-ac N` somente o que foi de fato verificado.
- Ligue a tarefa às evidências com `--add-ref <caminho ou URL>` e ao estudo com `--add-doc doc-NNN`; dependências com `--depends-on ISSUE-NNN`.
- Labels permitidas estão em `backlog/config.yml`; o Backlog.md não as valida, então não invente labels novas sem atualizar a configuração.

**Consultas úteis:** `backlog task list --exclude-status Done --sort priority --plain` · `backlog task list --ready --plain` · `backlog search <termo> --plain` · `backlog task view ISSUE-NNN --plain` · `backlog milestone list --plain` · quadro: `backlog board` (terminal) ou `backlog browser` (web em 127.0.0.1).

**Estudos privados:** `backlog/docs/estudos-locais/` está em `.git/info/exclude`. Nunca adicione esses arquivos ao Git (nem com `git add -f`), e nunca copie seu conteúdo para issues, PRs, commits ou qualquer serviço externo. Eles não existem no histórico do Git: faça backup local deles.

**GitHub:** a regra *issue-first* continua valendo para issues publicadas. Ao publicar uma tarefa, registre o link com `--add-ref https://github.com/.../issues/N`. Criar issues no GitHub, fazer *push* e abrir PRs dependem de autorização explícita do usuário.

## graphify

This project has a knowledge graph at /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- When /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json exists, run `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify query "<question>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` before your first grep, search or source-file read for a codebase question, and put this rule in every subagent prompt that explores code. Read raw files only for lines the graph or an earlier read already located. Use `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify path "<A>" "<B>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` for relationships and `/Users/evertonberger/.local/share/tunetrail-graphify/bin/graphify explain "<concept>" --graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `/Users/evertonberger/.local/share/tunetrail-graphify/analisar-tunetrail.sh` to keep the graph current (AST-only, no API cost).
- The `/graphify` skill is installed for Claude Code and Codex, and `graphify` is on the PATH. In this repository the graph lives outside the project: always pass `--graph /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json` to `query`, `path` and `explain`, and rebuild only with the script above. Never run `/graphify .`, `graphify extract .` or any semantic (LLM) extraction inside the repository; `.graphifyignore` keeps `backlog/docs/estudos-locais/` and `.agents/` out of any scan as a second safeguard.
- A committed /Users/evertonberger/.local/share/tunetrail-graphify/analysis/graphify-out/graph.json is rewritten whole on every build, so a large diff to it in a pull request is expected. Do not flag it as churn or suggest skipping the graph rebuild or its hooks.
