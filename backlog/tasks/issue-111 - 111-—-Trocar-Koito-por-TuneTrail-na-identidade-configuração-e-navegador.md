---
id: ISSUE-111
title: 111 — Trocar Koito por TuneTrail na identidade, configuração e navegador
status: Em andamento
assignee:
  - '@ebrgr'
created_date: '2026-11-14T01:53:00Z'
updated_date: '2026-11-14T03:40:00Z'
labels:
  - config
  - operacao
  - docs
milestone: m-5
dependencies: []
references:
  - backlog/evidencias/identidade/inventario-koito.md
priority: high
type: enhancement
ordinal: 111000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: pedido do usuário em 13/11/2026. Inventário completo em backlog/evidencias/identidade/inventario-koito.md: 597 ocorrências em 110 arquivos, classificadas pelo efeito da troca.

Princípio: trocar o que identifica o produto (para o usuário, para serviços externos e na configuração), manter o que nomeia o legado e o formato do Koito, e nunca quebrar uma instalação existente.

Fase A — identidade enviada a terceiros (sem compatibilidade a preservar):
1. User-Agent: de `Koito <versão> (contact@koito.io)` para `TuneTrail/<versão> ( https://github.com/ebrgr/TuneTrail )` (`internal/cfg/cfg.go:217`). Hoje o TuneTrail se identifica ao MusicBrainz, Deezer e Last.fm com o contato do projeto original. Não usar e-mail pessoal.
2. Cliente Subsonic `c=koito` → `c=tunetrail` (`internal/images/subsonic.go`, `subsonic_ping.go`).
3. Log de partida, mensagem de exportação e prefixo de temporários (`koito-form-*`) com o nome TuneTrail.
4. Modelo de issue do GitHub (`.github/ISSUE_TEMPLATE/bug-report.md`).

Fase B — configuração e navegador, com transição:
5. Variáveis: aceitar `TUNETRAIL_*` (e `TUNETRAIL_*_FILE`) para as 35 variáveis, mantendo `KOITO_*` como alternativa, no ponto único `envReader.get`. Se só o nome antigo estiver definido, funciona e registra um aviso de descontinuação uma vez por variável. Se os dois estiverem definidos com valores diferentes, recusar a partida com mensagem clara. A página de configurações mostra o nome efetivamente usado. Chaves de tradução `admin.setting.KOITO_*` passam a nomes neutros.
6. Exemplos e infraestrutura: `.env.example`, `deploy/.env.example`, `deploy/compose.yaml`, `Dockerfile` (`ARG`/`ENV`), workflows do CI, scripts de validação e `docs/.../reference/configuration.md` com os nomes novos; a documentação explica os antigos como alternativa.
7. Cookies: `koito_session` → `tunetrail_session` lendo o antigo como alternativa e substituindo-o na próxima resposta (o administrador não é desconectado); `koito_preauth` → `tunetrail_preauth`; `koito-weekstart` → `tunetrail-weekstart` com leitura do antigo uma vez (JS e `internal/webui/handlers.go`).
8. JS interno: `window.KoitoTheme` → `window.TuneTrailTheme` e evento `koito-theme-change` → `tunetrail-theme-change`.

Fase C — mudanças que dependem de decisão do usuário:
9. Diretório padrão `/etc/koito` (configuração, `Dockerfile`, compose): manter como padrão por compatibilidade (recomendado nesta tarefa) ou passar a `/etc/tunetrail` com alternativa automática quando `/etc/koito` existir.
10. Projeto `koito-sum` e serviço `koito` do compose: renomear para `tunetrail` exige parar o projeto antigo antes (`docker compose -p koito-sum down`), senão o contêiner antigo continua ocupando a porta; documentar no `DEPLOYMENT.md`.
11. Quando remover a alternativa `KOITO_*`: recomendado manter até o 1.0 (ISSUE-105).

Manter (nome Koito correto): `koito.db` e seus backups, o formato de importação/exportação `koito` e seus tipos Go, créditos no README, rodapé e LICENSE, histórico (CHANGELOG, releases, backlog, scripts históricos) e identificadores internos herdados do motor, para não dificultar atualizações do upstream.

Fora do escopo: texto e endereço do site de documentação (`docs/`, `site: https://koito.io`) ficam com a ISSUE-048; esta tarefa só atualiza os nomes de variáveis lá citados.

Pode ser dividida: a Fase A é pequena e independente e pode ser publicada antes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 User-Agent e cliente Subsonic identificam o TuneTrail, sem o contato do Koito (teste)
- [ ] #2 TUNETRAIL_* e TUNETRAIL_*_FILE aceitos para as 30 variáveis ativas com alias (das 34 constantes; DATABASE_URL, WEB_UI, SQLITE_ENABLED e ENABLE_FULL_IMAGE_CACHE ficam só como KOITO_*); KOITO_* funciona com aviso de descontinuação emitido uma vez na partida (testes)
- [ ] #3 Valores diferentes entre prefixos (inclusive combinações com _FILE e chaves não lidas) recusam a partida com mensagem que não contém valores (teste)
- [ ] #4 Administrador com koito_session continua logado após atualizar; o próximo login grava tunetrail_session e expira o antigo; logout revoga as duas sessões
- [ ] #5 Troca do cookie de início da semana não provoca recarga extra da página
- [ ] #6 Imagem sem ENV KOITO_* fixas; compose mantém a interpolação compatível com deploy/.env existente; exemplos, CI e referência usam os nomes novos; e2e roda com TUNETRAIL_* e uma jornada e um smoke de contêiner confirmam KOITO_* com diretório personalizado
- [ ] #7 Decisões da Fase C registradas (diretório padrão, nome do projeto compose, prazo do fallback de cookie, KOITO_* vira erro fatal no 1.0)
- [ ] #8 koito.db, formato e arquivo de exportação koito, créditos e histórico intactos; inventário atualizado ao final
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T03:05:00Z

---

Início: branch `feat/issue-111-trocar-koito-por-tunetrail-na` criada a partir de `main`.

---

author: claude
created: 2026-11-14T03:07:00Z

---

Fase A concluída (commit 95a3317): User-Agent TuneTrail/<versão> ( https://github.com/ebrgr/TuneTrail ) para MusicBrainz, Deezer e Last.fm; cliente Subsonic c=tunetrail no ping, busca e capas; log de partida, texto do log de exportação (formato Koito mantido) e prefixo tunetrail-form-*; modelo de bug report. Testes: TestUserAgentIdentifiesTuneTrail e TestSubsonicClientName. make check verde (10/10 e2e). Mantido de propósito: 'unsupported Koito version' no importador (versão do formato Koito) e KOITO_LOG_LEVEL no modelo de bug report até a Fase B. Fases B e C seguem para revisão do Advisor antes da implementação.

---

author: claude
created: 2026-11-14T03:40:00Z

---

Parecer do Advisor (@Opus, effort=medium), 14/11/2026: REPROVADO COM RESSALVAS para as Fases B e C como estavam escritas; direção mantida. Falhas críticas (verificadas no código): (1) Dockerfile:31-33 fixa KOITO_CONFIG_DIR/BIND_ADDR/LISTEN_PORT na imagem — trocar para TUNETRAIL_* gera conflito ou troca de diretório em upgrades com KOITO_CONFIG_DIR próprio (Unraid); remover essas ENV, os padrões do código são equivalentes. (2) compose.yaml interpola KOITO_DATA_DIR e afins do deploy/.env — renomear a interpolação aponta o volume para ./data vazio; manter KOITO_* na interpolação até o 1.0 ou fallback aninhado. (3) Makefile exporta .env e os sanitizadores de teste filtram só o prefixo KOITO_ (validate_multiuser.py:68, validate_standalone.py:39, cmd/api/*_integration_test.go) — com TUNETRAIL_* no .env o servidor de teste herdaria o diretório real; filtrar lista explícita dos dois prefixos. (4) envReader.get é preguiçoso; conflitos em chaves não lidas passariam — varredura antecipada em Load. (5) aviso não pode ficar em envReader.get (recriado a cada leitura da página de configurações; Load roda antes do logger) — coletar em Load e emitir em initLogger. (6) não codificar o nome no enum Origin (quebra comparações e deixaria campo editável sobrescrito); usar mapa Sources paralelo. (7) cookie de sessão: novo tem prioridade, antigo só fallback; login grava o novo e expira o antigo; logout revoga os dois IDs. (8) sem regravação silenciosa do cookie (sem expires_at a sessão viraria de navegador). Também: 34 constantes ativas, 30 recebem alias; koito-weekstart não é preferência (recalculado), o efeito é só uma recarga extra; incluir theme.test.mjs, validate_permissions.py e .koito_perm_check_*; não renomear koito_export.json (detecção de formato na reimportação); build-arg KOITO_VERSION e ARG precisam mudar juntos ou o binário deixa de usar -X. Plano corrigido: B1 resolvedor de configuração (env_alias.go, tabela de precedência, Sources, avisos, sanitizadores, testes); B2 navegador (cookies com fallback e logout duplo, weekstart, TuneTrailTheme); B3 infra e docs (remover ENV da imagem, ARG e workflow juntos, compose com interpolação compatível, e2e com TUNETRAIL_* e jornada/smoke legados). Fase C recomendada: manter /etc/koito até o 1.0; não renomear projeto/serviço compose antes do 1.0 (dois contêineres no mesmo SQLite); KOITO_* vira erro fatal no 1.0, nunca ignorado; fallback do cookie removido 30+ dias após B2. Critérios de aceite reescritos conforme o parecer.

---
<!-- COMMENTS:END -->
