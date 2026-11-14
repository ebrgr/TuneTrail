---
id: ISSUE-111
title: 111 — Trocar Koito por TuneTrail na identidade, configuração e navegador
status: Aberta
assignee: []
created_date: '2026-11-14T01:53:00Z'
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
- [ ] #1 User-Agent e cliente Subsonic identificam o TuneTrail, sem o contato do Koito (teste)
- [ ] #2 TUNETRAIL_* e TUNETRAIL_*_FILE aceitos para as 35 variáveis; KOITO_* continua funcionando com aviso de descontinuação (testes)
- [ ] #3 Valores diferentes em TUNETRAIL_X e KOITO_X recusam a partida com mensagem clara (teste)
- [ ] #4 Administrador logado com koito_session continua logado após atualizar e recebe tunetrail_session
- [ ] #5 Preferência de início da semana preservada na troca do cookie
- [ ] #6 Exemplos, compose, Dockerfile, CI e referência de configuração usam os nomes novos; a jornada e2e roda com TUNETRAIL_* e uma jornada confirma KOITO_*
- [ ] #7 Decisões da Fase C registradas (diretório padrão, nome do projeto compose, remoção de KOITO_*)
- [ ] #8 koito.db, formato koito, créditos e histórico intactos; inventário atualizado ao final
<!-- AC:END -->
