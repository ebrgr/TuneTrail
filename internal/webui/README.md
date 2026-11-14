# Prévia visual Go — roadmap 1

Camada de apresentação experimental e isolada. Templates, estilos, ícones e
fixtures são incorporados ao binário. Não importa banco, configuração da API,
autenticação, importadores nem relay. Usa somente a biblioteca padrão e o pacote
interno de versão. Os handlers reais continuam em `engine/`; a interface de produção é Go.

Na raiz do projeto:

```sh
make ui.preview
# Abrir http://127.0.0.1:14111/ ou http://127.0.0.1:14111/demo
```

A porta deve estar livre; não há troca automática. O comando aceita somente IPs
literais de loopback. Encerre com Ctrl+C. Não carrega `.env`; variáveis `TUNETRAIL_*` e `KOITO_*`
não escolhem banco nem credenciais nesta prévia.

Rotas Go: `/` (seleção), `/demo` (histórico fictício), `/luna` e `/iara` (vazios).
A seleção demonstra avatar por ícone, iniciais e imagem gerada localmente.
`/demo?state=empty`, `error`, `long` ou `broken` demonstra os estados adicionais.
`period=day|week|month|year|all_time` altera somente o estado visual do seletor;
não aplica uma consulta. Controles de mídia e busca explicam esse limite em um
diálogo. Administração não aceita credenciais ou gravações.

Arquivos:

- `preview.go`: modelos/fixtures, renderização, assets e fixtures.
- `templates/`: documento, estrutura, cards e diálogos com escape de `html/template`.
- `static/`: CSS explícito, dois scripts locais, imagem fictícia e SVGs Lucide.
- `palettes.json`: 12 paletas incorporadas à interface Go.
- `preview_test.go`, `theme.test.mjs`: renderização, isolamento da prévia e aparência.
- `../../cmd/ui-preview/`: comando de execução com restrição a loopback.

Os SVGs foram extraídos do `lucide-react` já instalado no projeto. A licença ISC
está em `static/icons/LICENSE`. As capas são SVGs fictícios produzidos pelo handler;
o avatar WebP foi gerado localmente com formas geométricas. Não são dados pessoais.

Verificações:

```sh
make ui.check
go test -race ./internal/webui ./cmd/ui-preview
CGO_ENABLED=0 go build -o .local/ui-preview ./cmd/ui-preview
```

Node é necessário somente para os testes JavaScript; a prévia Go compilada não depende de Node, libvips ou CGO em execução.
Templates/JS/CSS são incorporados: reinicie `go run` após editar os arquivos.

[Plano](<../../backlog/docs/roadmaps/doc-002 - Roadmap-1-—-layout-fontes-temas-barra-lateral-e-cards.md>) ·
[Capturas e resultados](../../backlog/evidencias/roadmap-01/README.md) ·
[Issue local resolvida](<../../backlog/tasks/issue-001 - 001-—-Prévia-visual-da-interface-em-Go-roadmap-1.md>)

## Administration integration (roadmap 4)

`internal/adminui` owns protected documents and POST forms. `engine/routes.go`
instantiates one handler in Go mode and shares a global login limiter with JSON.
`test_assets/admin-web-ui-routes.json` covers methods; the original route manifest
covers document navigation. The public UI and admin share
themes/fonts/assets, while templates and authorization remain separate.

Run `scripts/validate_admin_webui.py --binary <build> --work-dir <new-temp-dir>`
for real-server journeys using disposable SQLite v3. Screenshots are captured by
`scripts/capture_admin_webui.cjs` against the paired servers from
`validate_webui.py --serve`. Playwright/Chrome are test tools, not runtime
dependencies.
