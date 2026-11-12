# Evidências — roadmap 5

Estudo em 05/11/2026. [Plano](<../../docs/roadmaps/doc-008 - Roadmap-5-—-comportamento-permissões-fusos-e-isolamento-entre-perfis.md>).

## Verificação inicial

Executado sobre a árvore local, sem alteração de runtime:

```sh
GOCACHE=/private/tmp/koito-go-cache go test ./internal/webquery ./internal/summary ./internal/adminweb ./internal/adminui ./internal/db/sqlite ./internal/webui ./internal/relay ./engine
```

Os sete pacotes internos passaram (alguns usando cache). O engine encontrou a
restrição de bind de porta no sandbox; repetido com permissão para servidor local:

```sh
GOCACHE=/private/tmp/koito-go-cache go test ./engine
```

Engine passou em 3,609 s, usando suas bases/servidores de teste. Não foi uma
execução nova de toda a matriz proposta nem dos validadores Python ou do cliente.
Os resultados históricos da 015 permanecem no seu próprio relatório.

## Próximas evidências

Na implementação 016–021, registrar casos e oráculos, resultados de banco/cache,
configurações fictícias, comandos e limites. Capturas existentes da etapa 4 não
são comprovação dos casos temporais/concorrentes novos. Nunca incluir senhas,
chaves, tokens ou o banco operacional.

## Revisão Advisor

Sol, esforço alto: **APROVADO COM RESSALVAS**. Nenhuma falha crítica de
autenticação/isolamento comprovada. Cinco defeitos temporais reproduzidos entram
na 018; tabela no plano. Os demais riscos continuam hipóteses de validação.

O Advisor também executou, com cache temporário e downloads desabilitados:
`go test ./internal/admin ./internal/adminweb ./internal/db ./internal/webquery ./internal/relay`.
Todos passaram. Isso não substitui os novos casos nem a execução da 021.

## Reproduções temporais preservadas

[Fonte exata do diagnóstico](reproduce_time.go.txt) e
[comando, saída completa e oráculos](reproduce_time_results.txt). A saída mostra
os cinco resultados incorretos; exit code 0 significa execução do diagnóstico,
não aprovação dos casos. O programa usa banco em memória e foi removido após a
execução; a cópia `.txt` não integra o build. Não é ainda a regressão da 018.

Ambiente inicial: Go 1.26.3, darwin/arm64; Node v24.15.0 identificado, sem nova
execução do cliente neste estudo.

## Issue 016 — matriz e fixture

[Matriz de rotas, atores, escopos, oráculos e lacunas](ISSUE_016_MATRIZ.md).
Executar `python3 scripts/roadmap05_fixture.py`: fixture pequena Alpha/Beta/Empty
3/2/0 e variante grande 1005/2/0; checagens de esquema v3, duplicata, empate na
fronteira de export, `integrity_check` e `foreign_key_check` passaram (SQLite
3.54.1). Banco criado em diretório temporário e removido ao término.

`GOCACHE=/private/tmp/koito-go-cache GOPROXY=off go test ./internal/adminweb
./internal/adminui ./internal/webquery ./internal/webui ./internal/db/sqlite`
passou com cache. `python3 -m py_compile scripts/roadmap05_fixture.py` passou.
A validação inicial verifica a fixture diretamente com SQLite; não executa as
jornadas HTTP nem injeta o relógio de referência no servidor.

Advisor (Sol, esforço médio): **APROVADO** após revisão da fixture e da matriz;
ressalvas sobre precedência, erros e diferença de busca Go/React incorporadas.

[Issue 017 — permissões, sessões, origem e segurança dos logs](ISSUE_017.md):
correções e evidências separadas entre HTTP real, testes de router e parser.

[Issue 018 — fusos, intervalos e agregados civis](ISSUE_018.md): regressões,
comparação HTML/JSON, jornadas e limites de datas históricas.

[Issue 019 — isolamento, ingestão e cache](ISSUE_019.md): resultados,
correções e limites da matriz por perfil.

[Issue 020 — concorrência, falhas e I/O](ISSUE_020.md): correções, testes,
medições locais e cenários que ainda precisam de validação.

[Issue 021 — integração e aceite transversal](ISSUE_021.md): matriz consolidada,
resultados executados, decisões, pendências e checklist para a parte 6.
