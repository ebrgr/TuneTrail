# Inventário de "Koito" no código e no funcionamento

**Data:** 13/11/2026 · **Versão:** TuneTrail 0.8 (`bc2995e`) · **Tarefa:** ISSUE-111

Levantamento com `git grep -i koito`, sem `backlog/` e `scripts/historico/`: **597 ocorrências em 110 arquivos**. Classificação pelo efeito de uma troca.

## A. Identidade enviada a terceiros — trocar já

| Onde | Hoje | Efeito |
| --- | --- | --- |
| `internal/cfg/cfg.go:217` (User-Agent usado por MusicBrainz, Deezer e Last.fm) | `Koito <versão> (contact@koito.io)` | O TuneTrail se identifica com o contato do projeto original; limites de taxa e abusos de instalações TuneTrail são atribuídos ao mantenedor do Koito. Contraria a política de User-Agent do MusicBrainz. |
| `internal/images/subsonic.go:49-51`, `subsonic_ping.go:22` | parâmetro de cliente Subsonic `c=koito` | O Navidrome registra as requisições como vindas do "koito". |
| `engine/engine.go:53` | log de partida `Koito <versão>` | Logs do usuário mostram o nome antigo. |
| `internal/export/export.go:65`, `internal/adminweb/forms.go:172` | mensagem de log e prefixo `koito-form-*` de temporários | Cosmético. |
| `.github/ISSUE_TEMPLATE/bug-report.md` | "Koito version", `KOITO_LOG_LEVEL` | Modelo de issue do repositório. |

## B. Interface de configuração e navegador — trocar com período de transição

| Onde | Hoje | Cuidado |
| --- | --- | --- |
| `internal/cfg/cfg.go` (35 variáveis) e `toml.go` (`KEY_FILE`) | `KOITO_*` | Instalações existentes (inclusive Docker e Unraid) usam esses nomes. Todas as leituras passam por `envReader.get`, o ponto único para aceitar `TUNETRAIL_*` e manter `KOITO_*` como alternativa. Já existe `TUNETRAIL_LANGS_DIR`. |
| `.env.example`, `deploy/.env.example`, `deploy/compose.yaml`, `Dockerfile`, `.github/workflows/*.yml`, scripts de validação | `KOITO_*`, `KOITO_VERSION`, `KOITO_IMAGE_TAG`, `KOITO_DATA_DIR`... | Exemplos e CI passam a usar os nomes novos. |
| `langs/*.json`, `internal/adminui/settings.go` | chaves `admin.setting.KOITO_*` | Só nomes de chave de tradução; o usuário não vê. |
| `internal/adminweb/cookies.go`, `engine/handlers/auth.go` | cookie `koito_session` | Trocar sem transição desconecta o administrador uma vez. |
| `internal/adminweb/csrf.go:18` | cookie `koito_preauth` | Curta duração; troca direta. |
| `internal/webui/static/live.js`, `internal/webui/handlers.go:99` | cookie `koito-weekstart` | Preferência do visitante; ler o antigo uma vez e regravar com o nome novo. |
| `theme.js`, `ui.js`, `admin.js` | `window.KoitoTheme`, evento `koito-theme-change` | Interno aos nossos scripts; troca direta. |
| `internal/cfg/toml.go:240`, `Dockerfile`, `compose.yaml` | diretório padrão `/etc/koito` | Volumes existentes estão montados nesse caminho. Mudar o padrão exige alternativa ou migração; decisão do usuário. |
| `deploy/compose.yaml:1,4` | projeto `koito-sum`, serviço `koito` | Renomear o projeto cria contêineres novos; o antigo continua no ar e ocupa a porta se não for parado antes. |
| `.env.example` | `KOITO_CONFIG_DIR=.local/koito` | Diretório de dados local de desenvolvimento. |

## C. Manter — o nome Koito está correto

| Onde | Motivo |
| --- | --- |
| `koito.db`, `koito.db.pre-tunetrail-*.bak` (`internal/db/sqlite`, testes, `DEPLOYMENT.md`) | Nome do banco legado, origem da migração. |
| Formato de importação/exportação `koito` (`ImportKoitoFile`, `KoitoExport`, `KoitoListen`...) | Nome do formato JSON do Koito; mantém a interoperabilidade com o projeto original. |
| `README.md`, rodapé (`footer.based_on` + `koito-original.svg`), `LICENSE` | Crédito ao projeto original. |
| `CHANGELOG.md`, `releases/`, `backlog/`, `scripts/historico/`, `scripts/experiments/` | Registro histórico. |
| Identificadores internos de Go ligados ao motor herdado | Renomear dificulta trazer atualizações do upstream (princípio do ROADMAP); fora do escopo quando não visíveis ao usuário. |

## D. Documentação do site — ISSUE-048

`docs/src/content/docs` tem 109 ocorrências em 7 arquivos (44 "Koito" como nome do produto, 55 em `reference/configuration.md`, sobretudo nomes de variáveis) e `docs/astro.config.mjs` usa `site: "https://koito.io"`. Os nomes de variáveis acompanham a ISSUE-111; o texto e o endereço do site ficam com a ISSUE-048.

## Estado após a ISSUE-111 (14/11/2026)

**481 ocorrências em 97 arquivos** (antes: 597 em 110). Todas as restantes são intencionais:

| Grupo | O que fica | Até quando |
| --- | --- | --- |
| Alternativa de configuração | constantes canônicas `KOITO_*` em `internal/cfg/cfg.go` (os nomes externos `TUNETRAIL_*` são derivados), leitura de `KOITO_*` com aviso, marca **(DEPRECATED)** na página de configurações, testes e smoke de CI legados | 2.0 (decisão do usuário) |
| Interpolação do compose | `KOITO_DATA_DIR`, `KOITO_ADMIN_*`, `KOITO_IMAGE_TAG`, `KOITO_HOST_*`, `KOITO_TIMEZONE` no `deploy/.env` | 2.0 |
| Caminho legado | `/etc/koito` usado quando contém o banco ou é o único volume montado, com aviso | 2.0 |
| Chaves obsoletas | `KOITO_WEB_UI`, `KOITO_DATABASE_URL`, `KOITO_SQLITE_ENABLED`, `KOITO_ENABLE_FULL_IMAGE_CACHE` (só recusam valores antigos) | sem prazo |
| Banco legado | `koito.db` e `koito.db.pre-tunetrail-*.bak` (origem da migração, nunca alterada) | permanente |
| Formato de importação Koito | opção "Koito" e arquivos `koito*` aceitos ao lado de "TuneTrail" (mesmo esquema JSON); tipos Go `KoitoExport` mantidos para facilitar atualizações do upstream | permanente |
| Créditos e histórico | README, rodapé, `koito-original.svg`, CHANGELOG antigo, `releases/`, `backlog/`, scripts históricos | permanente |
| Testes | fixtures e testes que provam que os nomes antigos são recusados (cookies) ou aceitos com aviso (variáveis) | acompanham as regras |
| Site de documentação | texto que ainda chama o produto de Koito e `site: https://koito.io` | ISSUE-048 |

Trocado: User-Agent e cliente Subsonic; logs; modelo de bug report; variáveis (`TUNETRAIL_*`); cookies `tunetrail_session`, `tunetrail_preauth`, `tunetrail-weekstart`; `window.TuneTrailTheme`; diretório padrão `/etc/tunetrail`; projeto e serviço compose `tunetrail`; arquivo `tunetrail_export.json` e formato de importação TuneTrail; cliente "TuneTrail Web UI" nas escutas adicionadas pela interface; imagem sem variáveis fixas.
