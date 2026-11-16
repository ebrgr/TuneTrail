# TuneTrail — working notes for agents

TuneTrail registra o histórico de músicas (scrobbles), recebe eventos pela API
compatível com ListenBrainz e apresenta estatísticas em uma interface web.
O remoto `origin` aponta para o repositório TuneTrail. Não é necessário Docker nem PostgreSQL para desenvolver.

Este arquivo serve como um guia de *como trabalhar neste repositório*. Para evitar que o conteúdo fique defasado, ele propositalmente **não** repete informações documentadas em outros lugares — consulte esses documentos e, em seguida, siga as convenções abaixo.

# Subagentes Disponíveis
Os agentes e skills ficam em **`/.agents/`**, diretório local mantido fora do Git (`.git/info/exclude`). Não adicione esses arquivos a commits. Leia uma skill somente quando a tarefa a exigir.

- **`/.agents/reviewer.md`** — Reviewer. Antes do commit, audita o diff local (bugs, segurança, desempenho, qualidade) e emite `APROVADO` ou itens obrigatórios; o parecer fica em `backlog/evidencias/<issue>/`.
- **`/.agents/advisor.md`** — Advisor. Revisão crítica de lógica, concorrência ou algoritmos densos, e plano de implementação. É acionado pelo usuário (@Sol no Codex, @Opus no Claude).
- **`/.agents/golang-master/SKILL.md`** — Go de produção: erros, concorrência, `context`, tipos, testes, estrutura de módulos.
- **`/.agents/deslop/SKILL.md`** — Revisar diff em busca de complexidade acidental ou ruído de código gerado.
- **`/.agents/issue-first/SKILL.md`** — Ciclo de uma tarefa do Backlog.md com `make issue.*`.
- **`/.agents/changelog/SKILL.md`** — Quando e como escrever entradas no `CHANGELOG.md`.
- **`/.agents/ui-craft/SKILL.md`** — Projetar, construir e revisar UI.
- **graphify** — Navegação e manutenção do código (seção `graphify` abaixo).

## Regra anti-recursão (economia de tokens)
- **Quem recebeu a delegação executa; não re-delega.** Se você é o Reviewer, o Advisor, @Sol ou @Opus, ou foi chamado por outro agente, faça o trabalho você mesmo. É proibido invocar `claude -p`, `codex exec`, outro subagente ou outra persona para repetir a mesma análise.
- Só o agente de **nível superior** (o que conversa com o usuário) decide delegar, e delega **uma vez** por tarefa. Uma delegação já feita não é refeita para "confirmar".
- Se o parecer já existe em `backlog/evidencias/<issue>/` ou como `[HANDOFF]` no `ai-memory`, leia-o em vez de refazer a auditoria. Reaudite só o que mudou desde o parecer.
- Delegue só quando valer o custo: configuração, rotas e CRUDs simples não precisam de Advisor; o próprio agente revisa.

## Níveis de delegação (Advisor)
| Tipo de revisão | `@Sol` (Codex) | `@Opus` (Claude) |
| --- | --- | --- |
| Configuração, rotas, CRUDs simples (opcional) | `effort=low` | `effort=medium` |
| Criptografia, finanças, mensageria, algoritmos matemáticos (obrigatório) | `effort=medium` | `effort=high` |

## Protocolo Multi-Agente (Claude ↔ Codex) — Memória e economia
Os dois CLIs trabalham no mesmo repositório. Use `ai-memory` e `graphify` antes de ler arquivos inteiros. O Backlog.md continua sendo o registro oficial; o `ai-memory` guarda apenas o resumo curto para a troca entre agentes, sem duplicar o texto da issue.

1. **Início de sessão ou issue:** antes de planejar ou editar, consulte o `ai-memory` (`memory_query`) pelo `ISSUE-NNN` ou módulo afetado. Depois use `graphify query` para achar dependências e só então abra os arquivos.
2. **Plano para outro agente executar:** grave no `ai-memory` com tag `[PLANO ISSUE-NNN]`: **Objetivo** (1 linha); **Arquivos-alvo** (caminhos e funções exatos); **Restrições** (o que NÃO alterar e edge cases). Não despeje planos no chat; confirme só o ID salvo. O plano completo, se necessário, vai na tarefa do Backlog.
3. **Handoff ao trocar de agente ou CLI:** registre `[HANDOFF ISSUE-NNN]`: **Feito** (arquivos e motivo técnico); **Pendente** (próximo passo imediato); **Alertas** (riscos, débitos, testes quebrados).
4. **Auditoria cruzada:** (a) leia `[PLANO]` e `[HANDOFF]` da issue; (b) rode `git diff` só nos arquivos listados; (c) valide o impacto estrutural no `graphify`. Saída: apenas `[BUG]`, `[RISCO]` ou `[OK]`, com arquivo e linha. Sem elogios e sem reescrever código que já funciona.

## Leia isto primeiro (não duplique o conteúdo aqui)

- **`README.md`** — a apresentação: o que é o TuneTrail, como ele se posiciona em relação ao salvamento de feeds de música. Ele utiliza links externos em vez de fornecer explicações detalhadas; mantenha esse formato.

- **Documentação do usuário** — `docs/ contém informações sobre o aplicativo e como configurar o seu uso. O histórico de versões deve constar em `CHANGELOG.md`.

- **`backlog/`** — Registro oficial de issues, roadmaps, estudos e decisões, gerenciado pelo [Backlog.md](https://github.com/MrLesk/Backlog.md). Veja a seção **Registro do projeto — Backlog.md** abaixo. Após as alterações e testes, faça um resumo do que foi alterado e registre-o na tarefa correspondente.

- **`docs-dev/`** — Removido em 12/11/2026. Issues, roadmaps, estudos e PRs estão em `backlog/`; as evidências (capturas e relatórios) em `backlog/evidencias/`; os agentes em `/.agents/`. O conteúdo original permanece no histórico do Git.


## Convenções de trabalho

- **Nunca realize um *push* ou publicação sem autorização explícita.** Sinta-se à vontade para fazer *commits*; Qualquer ação que envolva alterações no sistema depende sempre da decisão do usuário — pergunte e aguarde. Uma *skill* (procedimento/tarefa) que mencione `git push` em suas etapas está apenas documentando o processo, não autorizando a execução.
A única exceção permanente: uma vez que um *branch* de um *Pull Request* (PR) tenha sido aprovado para envio (*push*), correções para falhas de CI nesse PR podem ser enviadas. A operação de *merge* sempre exige uma autorização explícita à parte. Para publicar uma tarefa aceita (`make issue.publish`, inclusive o que `PUSH=1` envia e dispara), consulte a *skill* `issue-first`.
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

Grafo do projeto fora do repositório. Abaixo, `G` = `/Users/evertonberger/.local/share/tunetrail-graphify`; o binário é `G/bin/graphify` (também no PATH) e o grafo fica em `G/analysis/graphify-out/`.

- Antes do primeiro grep, busca ou leitura de fonte numa pergunta sobre o código, rode `graphify query "<pergunta>" --graph G/analysis/graphify-out/graph.json`. Use `path "<A>" "<B>"` para relações e `explain "<conceito>"` para um conceito (sempre com `--graph`). Leia arquivos brutos apenas nas linhas que o grafo já localizou. Repasse esta regra a todo subagente que explore código.
- Para navegação ampla use `G/analysis/graphify-out/wiki/index.md`; `GRAPH_REPORT.md` só para revisão ampla de arquitetura.
- Depois de alterar código, rode `/Users/evertonberger/.local/share/tunetrail-graphify/analisar-tunetrail.sh` (AST, sem custo de API). Nunca rode `/graphify .`, `graphify extract .` ou extração semântica (LLM) dentro do repositório; `.graphifyignore` mantém `backlog/docs/estudos-locais/` e `.agents/` fora de qualquer varredura.
- Arquivos sujos em `graphify-out/` e um diff grande no `graph.json` (reescrito inteiro a cada build) são esperados; não os sinalize como ruído. Só ignore o grafo se a tarefa for sobre saída incorreta ou se o usuário pedir.
- `/graphify` digitado pelo usuário: use a skill instalada antes de qualquer outra coisa.
