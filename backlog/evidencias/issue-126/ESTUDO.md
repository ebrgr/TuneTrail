# ISSUE-126 — Estudo de desenho: conta e perfil num só lugar

Fase 1 (desenho). Nada foi implementado. Leitura estática da worktree em `5e642c7` (`internal/adminui/{routing,authorization,admin,account,actions}.go`, `internal/webassets/access_nav.go`, `assets/templates/admin/{account,admin}.html`), do estudo da 163 (§3–§5) e do doc-041 (§2, §5, §7).

## 1. O problema, em fatos do código

Hoje existem duas casas para o mesmo ser humano (conta e perfil são 1:1 desde o schema v5):

| | «Minha conta» `/account` | «Meu perfil» `/admin?profile=N` |
| --- | --- | --- |
| Conteúdo | Identidade (username), Senha, Sessão, Zona de perigo vazia | Nome, slug, proteção, avatar, chaves, importar/exportar, relay, link para Saúde; para root ainda Conta (username, token, papel, revisão de exclusão) |
| Alvo | derivado da sessão | **ID na URL**, validado no servidor |
| Forma | 4 cartões (125), atualização por fragmento (124) | uma página longa com âncoras `#details #keys #import #relay` |
| Quem vê | qualquer sessão | dono do perfil ou root |
| Menu (+) | «Minha conta» | «Meu perfil», «Saúde» |

Três fontes de confusão:

1. **Dois nomes para uma pessoa.** Username (login), nome de exibição e slug moram em páginas diferentes; o usuário comum não sabe onde está «o meu cadastro».
2. **Dois caminhos.** A seção «Conta» do root dentro do perfil repete o conceito da página Conta (username aparece nas duas).
3. **Uma URL por ID** para o próprio perfil, embora o servidor saiba quem é o dono (a 126 já pedia «alvo derivado da sessão»).

O que **precisa** de duas visões é só uma situação: root operando a conta/perfil de **outra** pessoa. Todo o resto é o mesmo usuário. O desenho deve separar *quem eu sou* de *quem eu estou operando*, e não *conta* de *perfil*.

Fatos que limitam o desenho (verificados no código):

- `/account` rejeita qualquer query string (`accountPage`: `len(r.URL.Query()) != 0` → 400). Novas abas por query exigem relaxar isso; abas por caminho não.
- `IsPage`/`IsAction` (`routing.go`) são listas fechadas: toda rota nova entra ali e em `authorizePage`/`authorizeAction`, que falham fechadas.
- `page()` já aceita `section=profiles|keys|relay|import|export` com `profile=N`; só `section=account` redireciona sempre para `/account` (`dispatch`, linha ~314).
- Todas as ações do perfil são `POST /admin/profile/<id>/…` com `profile_slug` no formulário e `AuthorizeProfile` (dono ou root) no servidor. Esse contrato é o que mais vale preservar.
- Os redirecionamentos pós-ação usam `profilePath(id)` = `/admin?profile=N` (`actions.go`).
- O usuário comum recebe 403 em `/admin`, `/admin/settings`, `/admin?profile=<outro>` e `/admin?section=account&profile=<outro>` (`engine/navigation_issue163_test.go`). Isso não muda.
- `AccessNavigation` já usa a mesma política do destino: o menu nunca oferece o que o servidor negaria.

## 2. Avaliação da proposta original da 126

Proposta: grade de cartões de perfil, página do perfil em cartões, zona de perigo, «Meu perfil» para o comum, lista root de contas e perfis.

| Peça | Veredito |
| --- | --- |
| Página do perfil em cartões + zona de perigo + funciona sem JS + fragmentos (124) | **Manter.** É o ganho visual da 074 e reaproveita o padrão da 125. |
| Grade de cartões de perfil + cartão «Criar» | **Manter, mas só para o root** (Administração). Para o comum, a grade teria um cartão: ruído. |
| «Meu perfil» como entrada separada de «Minha conta» | **Descartar.** É exatamente o que o usuário relatou como confuso. |
| Alvo derivado da sessão | **Manter** e estender a todas as abas do próprio usuário. |
| Aviso do relay junto à proteção (AC6) | **Manter** (já existe `protected-relay-help`). |

Conclusão: a proposta original organiza *dentro* de cada página, mas deixa a **arquitetura de duas entradas** intacta. Sozinha não resolve o problema relatado.

## 3. Opções

Legenda dos wireframes: `[ ]` botão/campo, `▸` item de menu, `●` aba atual.

### Opção 0 — Proposta original da 126 (referência)

Desktop `/admin?profile=N`: título, cartões Perfil e avatar | Chaves | Importar/exportar | Relay (grade 2 colunas), zona de perigo no fim. `/account` continua separada. Menu (+): «Minha conta», «Meu perfil».

Prós: pouca mudança estrutural; sem novas rotas.
Contras: mantém dois caminhos; root continua com a seção Conta dentro do perfil e duplicada com `/account`; não responde ao problema. **Não recomendada isolada.**

### Opção A — Um local único «Minha conta» com abas + Administração root separada

Pedida pelo usuário como exemplo. Detalho abaixo a variante refinada (A′, §4), porque A «pura» deixa uma lacuna: onde o root edita a pessoa Maria? Resposta: dentro da Administração, reutilizando o mesmo conjunto de abas em modo «contexto».

Wireframe desktop (≥64rem), usuário comum, `/account/profile`:

```text
┌ Tema/idioma ─────────────────────────────── (avatar) ▾ ┐   ← cabeçalho comum
│ Minha conta                                             │
│ Você · everton (@everton)                               │   ← identidade da sessão
│ [Conta] [●Perfil] [Chaves de API] [Histórico] [Relay] [Saúde]
│ ┌ Perfil e avatar ───────────┐ ┌ Visibilidade ─────────┐│
│ │ Nome  [__________]         │ │ [ ] Exigir login para ││
│ │ Identificador [____]       │ │     ver as escutas    ││
│ │ Avatar ▾ ícone  [Enviar]   │ │ Só você e administra- ││
│ │ [Salvar]                   │ │ dores veem. O relay   ││
│ └────────────────────────────┘ │ continua enviando.    ││
│                                │ [Salvar]              ││
│                                └───────────────────────┘│
│ ┌ Zona de perigo ─ (vazia: «sem ações ainda») ─────────┐│
└─────────────────────────────────────────────────────────┘
```

Wireframe celular (390 px): título e «Você · everton» empilhados; faixa de abas rolável na horizontal (cada aba 44 px, `aria-current`, sem JS); cartões em uma coluna; barra inferior fixa do app reservada.

Root («Minha conta») é idêntico ao do comum; a área root é outra:

```text
Administração  (menu (+) > Administração)
 [Contas e perfis] [Saúde] [Servidor]
 ┌ Contas e perfis ───────────────────── [+ Criar conta e perfil]
 │ (avatar) Maria  /maria  admin · ativa · Protegido   [Gerenciar ›]
 │ (avatar) João   /joao   usuário · pendente          [Gerenciar ›]
```

Mapa de rotas (A′, §5). Prós: conta e perfil de uma pessoa ficam juntos; nome «Minha conta» é o mais familiar; o comum nunca cruza uma fronteira root. Contras: «Conta» e «Perfil» continuam rótulos separados dentro do mesmo lugar (mitigado pelo cabeçalho único); seis abas no celular exigem faixa rolável.

### Opção B — Duas áreas, rótulos e shell distintos, abas compartilhadas

«Conta» = acesso (username, senha, sessão); «Perfil» = escutas (nome, slug, proteção, avatar, chaves, histórico, relay). Mesma faixa de abas nas duas, com um seletor de duas áreas no topo.

```text
[Conta ▪ Perfil]   ← alternador de área
Conta:  [●Identidade] [Senha] [Sessão]
Perfil: [●Perfil] [Chaves] [Histórico] [Relay] [Saúde]
```

Prós: a separação conceitual acesso x escutas é nítida; mudança de código menor que a A (não move `/account`).
Contras: continua havendo **duas entradas e dois destinos** no menu; o usuário comum ainda precisa decidir onde está o nome e onde está o username; duas faixas de abas aumentam a navegação (dois níveis) no celular. Resolve metade da queixa.

### Opção C — Centrar tudo no perfil (`/admin?profile=N`) com aba Conta

Prós: um só shell para dono e root (o root já usa o perfil como alvo).
Contras: troca senha, sessão e «Sair…» ficam sob «Meu perfil» (semântica estranha); o alvo próprio precisaria do ID na URL (contra o AC5 «derivado da sessão»); quebra a decisão da 163 de que `/account` é o destino de **qualquer** sessão e da 125 (cartões de conta em `/account`); `/account` teria de virar redirecionamento e todos os testes da 125/163 mudam. **Descartada.**

### Opção D — Um só local, mas **sem abas** (página única com âncoras fixas)

Uma página `/account` rolável com todos os cartões. Prós: zero rotas novas. Contras: é a página longa que hoje incomoda, agora para todos; mistura login com chaves de API e relay; cartões de importação/relay carregam e validam mesmo quando não usados. **Descartada.**

## 4. Recomendação: **A′** — «Minha conta» único com abas + Administração root, com modo «contexto»

Decisão em uma linha: *o usuário nunca tem dois caminhos para si mesmo; só o root tem um segundo caminho, e ele fica na Administração, com rótulo e cabeçalho que dizem «você está operando outra pessoa».*

Regras do desenho:

1. **Abas** (seis, mapeadas às seções já existentes): **Conta** (identidade, senha, sessão) | **Perfil** (nome, identificador, proteção com aviso do relay, avatar com dropdown de ícones) | **Chaves de API** | **Histórico** (importar/exportar) | **Relay** | **Saúde** (aba-link para `/admin/health?profile=N`, que passa a mostrar a mesma faixa de abas).
2. **Um único shell** (parcial de cabeçalho + faixa de abas) renderiza dois modos, decididos **no servidor**:
   - **Próprio**: alvo = perfil do usuário da sessão. URL sem ID: `/account`, `/account/profile`, `/account/keys`, `/account/history`, `/account/relay`.
   - **Contexto** (somente root, alvo de outra conta): `/admin?profile=N` (e `section=…`), cabeçalho «Maria (@maria) · você está operando como administrador», botão «Voltar para Contas e perfis». Na aba Conta mostra as ferramentas root (username somente leitura, estado, papel, token de ativação/recuperação, revisão de exclusão) **no lugar** de senha/sessão do ator.
3. **Cartões, não páginas longas**: cada aba é um conjunto de cartões no padrão 074/125 (uma coluna até 40rem, duas a partir de 64rem, alvos de 44 px, mensagens junto do cartão, fragmentos pelo mecanismo da 124, sem JS funcional).
4. **Administração** (root): «Contas e perfis» (grade de cartões e cartão «Criar conta e perfil»), «Saúde» e «Servidor». É o lugar da lista de contas; exclusão de conta continua pela revisão root.
5. **Menu (+)**: comum = Diretório · Tema e idioma · **Minha conta** · Sair…; root = igual + grupo **Administração** (Contas e perfis · Saúde · Servidor). «Meu perfil» deixa de existir como entrada.

### Wireframes do modo contexto

Desktop `/admin?profile=7` (root editando Maria):

```text
Administração › Contas e perfis › Maria
┌ ⚑ Você está operando a conta de outra pessoa ─ Maria (@maria) · usuário ─┐
│ Você: everton (root)                                  [‹ Contas e perfis] │
└────────────────────────────────────────────────────────────────────────────┘
[●Conta] [Perfil] [Chaves de API] [Histórico] [Relay] [Saúde]
┌ Acesso (somente leitura/root) ─┐ ┌ Papel ───────────────┐
│ Usuário maria · ativa          │ │ usuário  [Tornar adm]│
│ [Emitir token de recuperação]  │ └──────────────────────┘
└────────────────────────────────┘ ┌ Zona de perigo ──────┐
                                   │ [Revisar exclusão ›] │
```

Celular: faixa de contexto no topo (texto + ícone, sem depender de cor), faixa de abas rolável, cartões em coluna única; «Voltar» é o primeiro foco após o cabeçalho.

O aviso de contexto usa texto + ícone + borda (não só cor), em Yuu e Pearl. O ator («Você») e o alvo («Maria») são sempre ambos nomeados.

## 5. Mapa de rotas e redirecionamentos (A′)

Nenhuma URL atual deixa de responder. Todos os `POST` permanecem **exatamente** como estão (rota, campos, `profile_slug`, CSRF, ticket de uso único, SameOrigin).

| URL atual | Comportamento novo |
| --- | --- |
| `GET /account` | Igual: aba **Conta**, qualquer sessão. Continua rejeitando query. |
| `GET /admin?profile=N`, N = perfil **próprio** (comum ou root) | **303 → `/account/profile`**. Único caminho do próprio. |
| `GET /admin?profile=N`, N de **outra** conta, root | Igual URL; renderiza o shell em modo contexto, aba Perfil. Comum: 403 como hoje. |
| `GET /admin?section=keys\|relay\|import\|export&profile=N` | Próprio: 303 → `/account/keys`, `/account/relay`, `/account/history` (`import` e `export` → history). Outro/root: modo contexto na aba correspondente. |
| `GET /admin?section=account` | Igual: 303 → `/account`. |
| `GET /admin?section=account&profile=N` | Próprio: 303 → `/account`. Outro/root: aba **Conta** em contexto (novo significado; hoje sempre redirecionava). Comum: 403 (coberto por `navigation_issue163_test`). |
| `GET /admin?section=profiles` (sem perfil) | Lista root «Contas e perfis». Comum: 403 como hoje. |
| `GET /admin` (root, sem query) | Igual: lista root. |
| `GET /admin/profile/<id>/export`, `…/delete-review` | Inalterados (root/dono como hoje). |
| `POST /admin/profile/<id>/{save,avatar,avatar-clear,import,keys/*,relay/*,account-token,account-role,delete}`, `POST /admin/profile/create`, `POST /admin/account/{username,password}`, `/admin/login`, `/admin/logout`, `/admin/redeem`, `/admin/settings/*` | **Inalterados.** Só muda o destino do 303 de sucesso: `profileURL(ator, perfil, aba)`; se o ator é o dono → `/account/<aba>`, senão `/admin?profile=N&section=<aba>`. Decisão calculada no servidor, sem campo novo do cliente. |
| `GET /admin/health?profile=N` | Inalterada; ganha a faixa de abas (Saúde atual). |
| `GET /admin/settings[?profile=N]` | Inalterada (root). |

Rotas **novas** (todas GET, em `IsPage`): `/account/profile`, `/account/keys`, `/account/history`, `/account/relay`. Autorização: sessão válida + `ManageAccount`; o alvo é `OwnProfile(sessão)`; nenhum parâmetro (`profile`, `id`) é lido. Há um caminho por aba para o próprio e nenhum para o de outro, o que elimina uma classe de erro de alvo.

Âncoras antigas (`#keys`, `#import`, `#relay`, `#details`) viram abas; o fragmento da URL é preservado pelo navegador no 303 mas ignorado, sem erro.

## 6. Autorização e fronteira dono/root

- O **modo** é derivado de `alvo.UserID == ator.ID` (próprio) ou `ManageAccounts` (contexto). Nunca de query, cookie ou campo de formulário.
- Controles root (token, papel, revisão de exclusão, criar conta, lista de contas, Servidor) são renderizados por `AuthorizeAction(...)` no servidor **e** continuam validados em `authorizeAction` no POST. Esconder não é o controle; é só coerência.
- No modo próprio **nenhum** controle root aparece, mesmo para o root; o root usa a Administração para operar outros. Isso impede o root de confundir «meu» com «dele».
- Root visitando o próprio perfil em `/admin?profile=<próprio>` cai em `/account/profile`: o contexto «outra pessoa» só existe quando o alvo é outra pessoa.
- Fragmentos (124) herdam a regra: o cartão root de contexto só é emitido se `ManageAccounts` e alvo ≠ ator; testar o fragmento diretamente.
- Mantidos: `SafeReturn`/`login_return.go` (inclui `/account`; adicionar as quatro rotas novas à lista de destinos de leitura válidos), cache privado e `Vary: Cookie`, exceção de importação do dono (doc-041 §2).

Riscos principais:

| Risco | Mitigação |
| --- | --- |
| Rota nova esquecida em `IsPage`/`authorizePage` (falha fechada = 404/403 para o dono) | Testes de tabela por papel: visitante, comum, dono, root, terceiro, em cada rota nova. |
| Redirecionamento do próprio pode vazar a existência de IDs | O 303 só ocorre depois de `AuthorizeProfile` ter autorizado o alvo; negado continua 403/404 como hoje. |
| Root vê o shell de «próprio» com ferramentas root por engano | Modo calculado pelo servidor; teste que afirma ausência de `account-token`, `account-role`, `delete-review` em `/account/*` mesmo para root. |
| Perda de âncoras/links externos | Mapa §5 com 303; testes de redirecionamento. |

## 7. Coerência com decisões anteriores

- **163**: `/account` segue destino de qualquer sessão; o 403 do comum fora do próprio escopo não muda; o modelo de navegação continua o único ponto que compõe os links (apenas troca «Meu perfil» pelas abas e agrupa Administração).
- **125**: cartões, fragmentos `data-section`, e «Zona de perigo» vazia na Conta própria são mantidos; a exclusão de conta continua exclusivamente root (revisão).
- **128 (suspensa)**: login, resgate, revisões e mensagens ficam intocados; o shell em contexto prepara o terreno (revisões continuam em `/admin/profile/<id>/delete-review`).
- **Não tocar**: ingestão, schema, lógica de relay, resolvedor, arquivos da 172.

## 8. Impacto no modelo de navegação (`internal/webassets/access_nav.go`)

- `NavAccess.AccountLinks`: remover o item «Meu perfil» e o «Saúde» do dono (passa a ser aba). Fica «Minha conta».
- Novo `AccountTabs(actor, own, target) []NavLink` (puro, testável): devolve as seis abas com `Current`, e o modo (`self`/`context`). A mesma função serve a `/account/*`, ao contexto root e à Saúde.
- `ProfileSettings` (ícone do cabeçalho do perfil visitado): para o dono → `/account/profile`; para root em perfil alheio → `/admin?profile=N`.
- `RootLinks` vira o grupo «Administração»: Contas e perfis, Saúde, Servidor (já existente; ajuste de rótulo/ordem).
- Rótulos: manter `nav.my_account` («Minha conta»); remover `nav.my_profile`; novos: `account.tab.*`, `admin.context.*`, `admin.accounts.*` em `langs/en.json` e `langs/pt-BR.json`.

## 9. Impacto em testes

Mudam (a verificar por `grep -rln 'admin?profile\|profilePath\|section=profiles\|nav.my_profile' internal engine`, que hoje lista estes):

- `engine/navigation_issue163_test.go` — Meu perfil sai do menu; novas asserções de ausência de controles root em `/account/*`.
- `engine/account_issue125_test.go` — abas; Conta continua; novo teste de que `/account` não mostra Meu perfil.
- `engine/profile_management_test.go`, `engine/adminui_import_test.go`, `engine/import_permissions_issue160_test.go`, `engine/profile_relay_test.go` — destino do 303 após `save`, `avatar`, `import`, `keys`, `relay` (próprio vs outro).
- `engine/admin_sidebar_123_test.go`, `engine/sidebar_122_test.go`, `engine/admin_badge_109_test.go` — links do ícone de configurações e do selo de administrador.
- `engine/auth_issue159_test.go`, `engine/sign_out_issue157_test.go`, `engine/visibility_issue161_test.go`, `engine/i18n_admin_052_test.go`, `engine/adminui_sections_124_test.go` — URLs de retorno, redirecionamentos, textos e fragmentos.
- `engine/account_tokens_162_test.go`, `internal/adminui/account_tokens_test.go`, `internal/adminui/admin_test.go`, `internal/adminweb/csrf_test.go` — token/papel em modo contexto; CSRF inalterado.
- `internal/webassets/nav_test.go` — `AccountTabs`/`AccessNavigation`.
- `internal/webui/reference_test.go` — referências de renderização das páginas admin (regenerar).

Novos: tabela por papel (visitante, comum, dono, root, terceiro) para cada rota nova e cada redirecionamento do §5; fragmento root negado a quem não é root; teste de que o POST legado continua aceito com o mesmo contrato.

## 10. Esforço (arquivos)

| Área | Arquivos |
| --- | --- |
| Rotas e autorização | `internal/adminui/routing.go`, `authorization.go`, `admin.go` (dispatch e `page`), `actions.go` (destino do 303), `login_return.go`, `internal/adminweb/return.go` |
| Shell e abas | novo `internal/adminui/account_tabs.go`; `internal/adminui/account.go` (render genérico por aba) |
| Navegação | `internal/webassets/access_nav.go` (+ testes) |
| Templates | `assets/templates/admin/account.html` (shell + abas), novo `admin/profile-cards.html` (cartões extraídos da página longa de `admin.html`), `admin/admin.html` (remove a página longa; lista root em cartões), `admin/ingest-health.html` (faixa de abas) |
| Estilo e i18n | `assets/css/admin/account.css`, `admin.css`; `langs/en.json`, `langs/pt-BR.json` |
| Docs | `docs/`, `CHANGELOG.md` (inglês) |

Estimativa: um desenho médio, ~12 arquivos de código/templates e ~14 de testes; sem mudança de schema, ingestão ou serviços (`internal/admin` fica como está).

## 11. Plano em cortes pequenos (cada um com `make check` verde)

1. **Modelo de abas**: `AccountTabs` e modo (próprio/contexto) em Go puro + testes unitários, sem mudar UI.
2. **Rota própria Perfil**: `/account/profile` (cartões Perfil e avatar + Visibilidade com aviso do relay), `IsPage`/`authorizePage`/retorno pós-login, `profileURL` para o 303; `/admin?profile=<próprio>` → 303. Menu: «Meu perfil» aponta para `/account/profile`.
3. **Abas Chaves, Histórico, Relay** próprias (`/account/keys|history|relay`) reutilizando os cartões extraídos; redirecionamentos `section=…` do próprio.
4. **Modo contexto root**: `/admin?profile=N` com cabeçalho «Você × Maria», abas e aba Conta root; remove a página longa antiga.
5. **Administração**: lista de contas e perfis em cartões + «Criar conta e perfil»; grupo Administração no menu; remove «Meu perfil» e «Saúde» do menu do comum; faixa de abas em `/admin/health`.
6. **Acabamento**: CSS responsivo, i18n completo, docs, CHANGELOG, remoção de template legado, referências de renderização. (Capturas e aceite visual ficam com o usuário no servidor.)

## 12. Decisões para o usuário

1. Aprova **A′** (um «Minha conta» com abas + Administração root; contexto root para outra pessoa) em vez da proposta original?
2. **Abas**: seis (Conta, Perfil, Chaves, Histórico, Relay, Saúde) ou cinco, juntando Chaves e Relay em «Integrações»? (Recomendo começar com seis, que mapeiam 1:1 as seções atuais.)
3. **Saúde** como aba-link (recomendado) ou só no grupo Administração/menu?
4. Aceita que `/admin?profile=<próprio>` passe a **redirecionar** para `/account/profile` (um só caminho para o próprio)?
5. Root excluindo a **própria** conta: manter pela Administração (recomendado, Zona de perigo da Conta própria continua vazia, decisão da 125).
