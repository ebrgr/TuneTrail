# Issue 022 — executar fixture e capturas futuras

[Fixture/matriz](ISSUE_022_MATRIZ.md) · [Modelo de manifesto](manifesto-modelo.json).
A 022 preparou o ambiente; **não executou comparações nem criou screenshots**.
Os comandos abaixo exigem dependências já instaladas e banco/diretório descartável.
O binário real da API usa CGO/libvips. O servidor usa portas loopback escolhidas
pelo sistema, ignora variáveis `KOITO_*` herdadas, desativa fontes musicais externas
e encerra ambos processos ao receber SIGINT/SIGTERM. Não lê `.env` como configuração
ou toca no banco operacional. Não executar este fixture em produção.

## Reproduzir e encerrar

Na raiz, com `client/build/client` já construído quando se desejar comparar SPA:

```sh
GOCACHE=/private/tmp/koito-go-cache GOPROXY=off go build -o /private/tmp/koito-comparacao ./cmd/api
python3 scripts/test_roadmap06_fixture.py
python3 scripts/serve_roadmap06.py --binary /private/tmp/koito-comparacao --work-dir /private/tmp/koito-visual-novo
```

O caminho `--work-dir` **não pode existir**; crie outro nome em cada execução.
O comando imprime os dois endereços e as contagens, e mantém os servidores até
Ctrl+C. `--smoke` inicia e encerra em seguida. Se um servidor falhar, o `finally`
encerra os iniciados. Inspecione/limpe apenas o diretório descartável criado.

Para criar apenas as bases sem iniciar servidores:

```sh
python3 scripts/roadmap06_fixture.py --output-dir /private/tmp/koito-fixture-nova
```

A fixture usa usuário `validation-admin` e senha **fictícia** `validation-password`
e chaves fictícias da fixture anterior. Credenciais são públicas e só servem para
bancos de teste. O script cria `base.sqlite3`, cópias `go`, `react`, `mutation`,
`zero`, `one`, `large` e `fixture-report.json`; o diretório tem modo restrito.
O usuário já existe no banco para que o bootstrap não acrescente outro perfil.
As cópias `go`/`react` contêm a mesma imagem local. Para cada ação mutável, crie
uma fixture inteiramente nova e use uma cópia por modo; não reteste a base já
alterada. Variantes zero/one/large têm bancos independentes; para jornada pareada
com essas variantes, crie duas fixtures novas e use a mesma variante de cada uma.
A inicialização real v2→v3 não é fabricada pela fixture v3; seguir 025.

## Captura e registro nas issues 023–025

1. Registrar commit e alterações locais, versões do navegador/Go/Node, comandos de
   build, hash do relatório da fixture e se a SPA construída corresponde à árvore.
   Não registrar valores de cookies, chaves, senhas ou tokens.
2. Utilizar apenas origens `127.0.0.1` das duas instâncias; bloquear requisições
   para outros hosts, inclusive releases. Se fontes locais conhecidas forem
   disponibilizadas, conferir família efetivamente carregada; caso contrário,
   registrar que a fonte remota foi bloqueada e qual fallback foi usado. Não
   declarar paridade de fonte quando não for possível confirmá-la.
3. Fixar viewport em CSS, escala, zoom, tema, idioma, fuso e movimento reduzido.
   Checar route/estado esperados; na SPA, comprovar hidratação com interação que
   atualiza estado visível. Aguardar `document.fonts.ready`, verificar status e
   família de fonte efetivamente disponível, e conferir `complete`/`naturalWidth`
   das imagens esperadas; anotar falhas de imagem deliberadas. Aguardar conteúdo
   final/estado da jornada, sem fazer sleeps fixos passarem como prontidão. Uma
   falha de prontidão é resultado inconclusivo, não uma captura aprovada.
4. Servidor Go não recebe relógio fixo pela fixture; usar períodos explícitos e
   dados fixos, ou controlar ambos os relógios por mecanismo testado. Anotar toda
   dependência restante do relógio real. Fixar apenas `Date` do browser não basta.
5. Para cada caso da matriz, preencher o [manifesto](manifesto-modelo.json):
   ator, modo, perfil, rota/query **sanitizada**, resultado esperado/observado,
   JS/no-JS, estado, tema, viewport, fuso, região, rolagem, fonte, imagem, zoom,
   arquivo e comparação. Usar nomes sem segredos. Rolar seções separadamente e
   registrar overflow tanto da página quanto de tabelas com rolagem própria.
6. Conferir filtros, links e mutações de fato; screenshots não provam hidratação,
   permissão, foco ou integridade. Em mutações validar contagens e SQLite com
   `PRAGMA integrity_check` e `PRAGMA foreign_key_check` na cópia correspondente.
   Classificar diferenças em defeito, alteração deliberada ou diferença visual
   aceita; não copiar um bug React para atingir paridade.
7. Armazenar somente imagens e manifesto sanitizados em `screenshots/` e comparar
   pares da mesma fixture/estado. Tokens, chaves e senhas jamais devem aparecer;
   revelar chave não deve ser fotografado. Marcar o que não foi executado.

O script antigo `capture_admin_webui.cjs` usa timeout curto de fontes e espera
fixa; não serve como comprovação automática desses critérios sem ajuste e revisão.
