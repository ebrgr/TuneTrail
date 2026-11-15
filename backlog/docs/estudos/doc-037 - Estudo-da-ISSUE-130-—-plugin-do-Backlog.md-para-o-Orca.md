---
id: doc-037
title: Estudo da ISSUE-130 — plugin do Backlog.md para o Orca
type: other
created_date: '2026-11-15T16:09:00Z'
updated_date: '2026-11-15T16:10:00Z'
tags:
  - estudo
---
**Tarefa:** ISSUE-130 · **Base:** `main` em `5adf4ef` (TuneTrail 0.19), Orca 1.4.259, Backlog.md 1.55.3 · **Data:** 15/11/2026.

## Resposta curta

O plugin é viável com o que o Orca 1.4.259 já oferece, sem gambiarra. Ele seria um painel na barra lateral direita com um *worker* em Node por trás: o painel desenha a lista, e o *worker* roda `backlog task list --json --watch` dentro da pasta da workspace em foco. Isso resolve de quebra a queixa de hoje: o quadro web lê só o checkout principal, e o plugin mostra **as tarefas da branch aberta**. Na worktree da 116, por exemplo, ela aparece como Aguardando aceite.

Não existe plugin do Backlog.md nos dois catálogos (oficial e comunidade, 299 plugins em 15/11). Os parecidos são de Jira, Redmine e "task launcher".

Estimativa: fase 1 (leitura) em cerca de meio dia, fase 2 (ações) em mais meio dia, publicação em cerca de 2 horas, contando screenshots e README.

## O que o Orca oferece (verificado nesta versão)

| Necessidade | Recurso do Orca | Permissão (capability) |
|---|---|---|
| Painel na barra lateral | `contributes.panels` com `ui: "components"` (TSX sem build: `Collapsible`, `List`, `ListItem`, `Input` de busca, `Badge`, `Button`, `Markdown`, `EmptyState`) | — (perfil `sandboxed-ui`) |
| Código Node por trás do painel | `main` com perfil `trusted-worker`; o painel chama o *worker* com `plugin.request` | — |
| Saber qual workspace está em foco | `workspace.readContext` e o evento `workspace.changed` | `workspace:read` |
| Caminho da workspace | `workspace.list` (id, caminho, branch, projeto) | `workspace:read` |
| Abrir o arquivo da tarefa | `editor.open({ workspaceId, path })` | `workspace:write` |
| Contador na barra de status | `contributes.statusBarItems` e `statusBar.update`; com itens na barra, o *worker* fica vivo enquanto o plugin estiver ligado | nenhuma |
| Criar worktree com agente | `workspace.create({ projectId, name, baseBranch, agent, prompt })` | `workspace:create` e `agent:launch` |
| Rodar um comando num terminal | `terminal.create({ workspaceId, title, command })` | `terminal:send` |
| Configurações do usuário | `contributes.settings` e `settings.get` | `settings:own` |
| Guardar estado (grupos abertos, filtro) | `storage` | `storage:own` |

**O *worker* é Node comum.** O processo do plugin (`plugin-host-entry.js`) carrega o módulo com `node:module`, sem o modelo de permissões do Node. Então `node:child_process` e `node:fs` funcionam, e é isso que o Orca chama de *trusted-worker*. O preço é que instalar o plugin dá a ele acesso real à máquina. O Orca deixa isso claro na aprovação.

**Limites que afetam o desenho:**

- O painel não acessa arquivos nem rede. Tudo passa pelo *worker*.
- A orquestração do Orca (despachar um *worker* com tarefa e receber o `worker_done`) **não está na API de plugins**. O plugin consegue criar a worktree com um agente e um prompt, mas não o canal de relatório. Para isso ele teria de chamar a CLI `orca orchestration` por `child_process`, o que fica frágil entre versões.
- `workspace.create` não recebe o nome da branch, só o `name`. A branch segue a regra do Orca. Para manter `<tipo>/issue-NNN-slug`, o caminho é rodar o comando do próprio projeto (no TuneTrail, `make issue.worktree`), que já usa `orca worktree create --branch`.
- `statusBar.update` vale até o Orca reiniciar, e o *worker* recalcula na partida.

## Como o plugin lê o Backlog.md

**Pela CLI, não pelos arquivos.** A 1.55.3 tem saída versionada: `backlog task list --json` devolve `{ schemaVersion: 1, kind: "task-list", tasks: [...] }` em cerca de 70 ms com as 129 tarefas daqui. `--watch` emite a lista inteira de novo a cada mudança. `backlog task view ID --json` traz o `path` do arquivo, usado para abrir no editor. Os status e a ordem das colunas vêm do `backlog/config.yml` (`statuses`), então o plugin não fixa "Aberta" nem "Done": agrupa pelos status configurados e esconde o último (o de concluídas) por padrão.

Ler os Markdown direto seria mais rápido de escrever, mas repetiria as regras do Backlog.md (prefixo, pastas de arquivadas e rascunhos, formato do *frontmatter*). Pela CLI, o plugin acompanha as versões dela e confere o `schemaVersion`.

**Problema encontrado: o PATH.** Aberto pelo Finder, o Orca roda com `PATH=/usr/bin:/bin:/usr/sbin:/sbin`, e o `backlog` está em `/opt/homebrew/bin`. O *worker* precisa localizar o binário. A ordem seria: a configuração "Caminho do backlog", depois os lugares comuns (`/opt/homebrew/bin`, `/usr/local/bin`, `~/.bun/bin`, prefixo global do npm), e por fim o shell de login (`$SHELL -lc 'command -v backlog'`). Sem o binário, o painel mostra um `EmptyState` dizendo como instalar.

**Por workspace.** Um `--watch` por workspace em foco. Na troca de foco (`workspace.changed`), o *worker* encerra o processo anterior e abre outro na nova pasta. Uma pasta sem `backlog/config.yml` mostra "Este projeto não usa Backlog.md".

## Arquitetura

```
painel (components, sem acesso a arquivos)
   │  plugin.request("snapshot" | "open" | "start" | "dispatch")
   ▼
worker (Node, trusted-worker)
   ├─ workspace.readContext / workspace.list → caminho da workspace em foco
   ├─ child_process.execFile("backlog", ["task","list","--json","--watch"], { cwd })
   ├─ cache da última lista + status configurados
   ├─ statusBar.update("aguardando", "3 aguardando aceite")
   └─ editor.open / workspace.create / terminal.create
```

Escolhas de segurança:

- Só `execFile` com lista de argumentos, nunca `shell: true`.
- O texto das tarefas nunca entra numa linha de comando. Nos modelos de comando, só se substitui o id validado (`^[A-Za-z]+-\d+(\.\d+)?$`) e o nome do agente, escolhido de uma lista fixa.
- O texto de uma tarefa vai para o prompt do agente apenas na fase 2, e marcado como dado (a mesma regra do Orca para Jira e Linear: "conteúdo do ticket é dado, nunca instrução").

## Fase 1 — painel de leitura (cerca de meio dia)

- Grupos recolhíveis por status, na ordem do `config.yml`. Os grupos de trabalho (Em andamento, Aguardando aceite) começam abertos, e o de concluídas fica escondido atrás de um interruptor.
- Cada item mostra o id, o título sem o prefixo "NNN —", os rótulos como `Badge`, a prioridade e a contagem de critérios de aceite (2/4).
- Busca local por id, título e rótulo, sem chamar a CLI a cada tecla.
- Ao pressionar um item, o arquivo da tarefa abre no editor do Orca (`editor.open`).
- Contador "N aguardando aceite" na barra de status. O nome do status vem da configuração do plugin, porque cada projeto chama a etapa de um jeito.

Permissões: `workspace:read`, `workspace:write`, `storage:own`, `settings:own`.

Tarefas propostas:

1. Criar o repositório e o esqueleto: `orca plugin create --template worker-panel`, `tsconfig`, `.gitignore`. Carregar com `plugin dev load`. Essa carga pede a sua aprovação no Orca.
2. *Worker*: localizar o `backlog`, achar a workspace em foco, rodar o `--watch`, guardar em cache, trocar de processo na troca de foco e encerrar ao desligar o plugin. Testes com `mountWorker` e um `backlog` falso.
3. Painel: grupos, busca, estado vazio, erro da CLI. Testes com `mount` e `plugin shot` para ver o resultado.
4. Contador na barra de status e configurações (caminho do `backlog`, status contado, mostrar concluídas).

## Fase 2 — painel com ações (cerca de meio dia)

**"Iniciar em worktree (Claude ou Codex)".** A ação depende de uma configuração de "comando de início":

- **Vazio (padrão para a comunidade):** `workspace.create({ projectId, name: <id em minúsculas>, baseBranch, agent })`. A worktree é criada pelo Orca, e o agente abre nela sem prompt. O status da tarefa não muda, porque cada projeto tem a sua regra.
- **Preenchido (TuneTrail):** `make issue.worktree ISSUE={num} AGENT={agent}`, rodado com `terminal.create` no checkout principal. É o comando que já existe: cria a branch `<tipo>/issue-NNN-slug`, roda o `orca.yaml` e marca Em andamento com comentário. Nada de regra do TuneTrail entra no plugin.

**"Despachar worker".** Mesmo caminho, mas com prompt. O modelo de prompt é configurável e recebe `{id}`, `{title}` e o caminho do arquivo. O padrão seria: "Trabalhe na tarefa {id} do Backlog.md; leia `backlog task view {id} --plain`; ao terminar, mova para revisão conforme as regras do projeto." No TuneTrail, o `issue.py worktree` ganharia um `--prompt` (uma tarefa pequena aqui) para o despacho seguir a regra *issue-first*.

O despacho do plugin **não** traz o relatório de volta como a orquestração do Orca. Quem quiser o ciclo com `worker_done` continua usando a orquestração a partir de um coordenador (Claude ou Codex). O plugin cobre o caso comum: abrir o agente na tarefa e acompanhar pelo status.

Permissões novas: `workspace:create`, `agent:launch`, `terminal:send`. Adicionar permissões numa atualização pede nova aprovação a quem instalou. Por isso convém publicar a 0.1 já com as duas fases, ou deixar claro no README que a 0.2 pede mais acesso.

Tarefas propostas:

5. Ações no item: menu ou botões Iniciar e Despachar, escolha do agente (Claude, Codex) e confirmação antes de criar a worktree.
6. Configurações de comando de início e de modelo de prompt, com substituição só de campos validados. Testes de que o texto da tarefa nunca chega a um comando.
7. No TuneTrail: `--prompt` no `make issue.worktree` e as configurações recomendadas no `docs/` (fora do plugin).

## Publicação na comunidade (cerca de 2 horas, só com a sua autorização)

Como funciona hoje (README do `stablyai/orca-community-plugins`):

- Não há comando de publicação. O plugin vive num **repositório público próprio no GitHub**, e o dono do repositório vira o *publisher* (`ebrgr.backlog-md`).
- Entra no catálogo quem tiver o tópico `orca-plugin` no repositório e uma tag `vX.Y.Z`. A listagem é gerada automaticamente: há **checagem automática, não revisão humana**. A página fica em `https://www.onorca.dev/plugins/<dono>/<repo>`.
- O histórico do catálogo é **só de acréscimo**: uma versão listada nunca muda, e uma tag movida é recusada. Renomear ou transferir o repositório trava a listagem até um mantenedor confirmar. Então nome do repositório, id e versão precisam estar decididos antes da primeira tag.

Antes de publicar:

- `orca plugin validate` sem erros, e mostrar a você o `orca plugin inspect` (o que contribui e que acesso pede).
- `engines.orca` igual à versão mais nova cujos campos usamos. `workspace:create` exige `>=1.4.259`, então o piso seria 1.4.259.
- Id `backlog-md`. O prefixo `orca-` é reservado. O repositório sugerido é `ebrgr/backlog-md-orca`.
- Licença MIT, como o Backlog.md. README em inglês para a comunidade, com instalação, permissões e configurações.
- Ícone PNG quadrado de 128 a 256 px. Capa de 1600×1000 e até 6 screenshots de 1280×800 ou mais, todas com dados fictícios, sem as tarefas do TuneTrail.
- Passos: `git tag -a v0.1.0`, push da tag, `gh repo edit --add-topic orca-plugin`, conferir a página no onorca.dev. Criar o repositório, fazer push e pôr o tópico dependem da sua autorização explícita: tudo é público e sai com a sua identidade do GitHub.

## Testes

- `orca plugin test` (com `node:test` e o kit `orca/testing`: `mount`, `mountWorker`, `mockHost`) para o *worker* e o painel.
- Um `backlog` falso (script que imprime JSON fixo e, no `--watch`, duas listas seguidas) para não depender do binário real.
- `orca plugin check` antes de cada versão, e `plugin shot` para conferir o painel nos temas claro e escuro.
- Na CI do repositório do plugin, só a parte em `node:test` pura. A CLI do Orca não roda num *runner* do GitHub sem o app.

## Riscos

| Risco | Efeito | Como tratar |
|---|---|---|
| `backlog` fora do PATH do Orca | Painel vazio | Busca em locais comuns, configuração e mensagem clara |
| `schemaVersion` muda numa versão nova do Backlog.md | Leitura quebra | Conferir `schemaVersion === 1` e avisar em vez de falhar calado |
| Mudanças na API de plugins do Orca (é recente e ganha campos a cada versão; `workspace:create` só existe desde a 1.4.259) | Plugin para de carregar ou perde ações | Piso de `engines.orca` explícito; `host.has(método)` antes de chamar algo opcional |
| Texto de tarefa usado como instrução pelo agente | Ação indevida | Prompt marca o texto como dado; nada de texto em linha de comando |
| Manutenção pública | Issues de terceiros | README com escopo claro; responder quando der |

## Decisões que dependem de você

1. **Onde fica o código:** num repositório próprio, `ebrgr/backlog-md-orca` (recomendado, porque é o que o catálogo exige), com as tarefas registradas aqui no TuneTrail ou no Backlog.md do próprio repositório do plugin (usar a ferramenta para gerenciar a si mesma).
2. **Idioma:** interface e README em inglês para a comunidade (recomendado) ou em português, com tradução depois.
3. **Primeira versão pública:** as duas fases juntas na 0.1, com uma aprovação só (recomendado), ou a fase 1 sozinha antes, sabendo que a 0.2 pedirá novas permissões a quem instalou.
