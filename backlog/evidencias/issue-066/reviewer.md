**APROVADO**

Reavaliei só os achados anteriores. Não alterei arquivos e não invoquei agentes.

**Médio (gate de e2e): resolvido.**
- `scripts/e2e.py:20` registra a jornada `web-assets`, que roda `validate_web_distribution.py` com o binário real. O script existe.
- `/tmp/issue066-final-check.log:76-88` mostra `PASS web-assets` e `e2e: 12/12 journeys passed`, sem nenhum `FAIL` no log.

**Baixos: resolvidos.**
- **Symlinks:** `confinedFS.Open` (`internal/webassets/assets.go:282-298`) roda `Lstat` em cada componente do caminho dentro do `os.Root` e recusa qualquer symlink, inclusive links internos de categoria e diretórios-link. O `fs.Stat` de `Load` também passa por `confinedFS.Open`, então não há caminho que pule essa checagem.
- **Testes negativos:**
  - `TestManifestCategories` (`assets_test.go:147-161`) cobre alias público→admin, manifest servido, extensão proibida, troca de palette, `..` e caminho absoluto.
  - `TestInternalSymlinkCannotExposePrivateTemplate` (`assets_test.go:164-176`) cobre o link interno.
  - O caso `escape` continua coberto.
- **Dockerfile:** `chmod` 0555/0444 é feito no estágio `backend` (`Dockerfile:17-18`) e os assets são copiados uma única vez no final, com `--chown=0:0` (`Dockerfile:27`). `.dockerignore:26` exclui `assets/brand/`.
- **Raiz em testes e dev:** está documentada em `DEVELOPMENT.md:35`. `Makefile:6-7` exporta as variáveis com `?=`, e `launch.json` (as duas configurações) e `test.yml:18-19` definem `TUNETRAIL_WEB_DIR` e `TUNETRAIL_LANGS_DIR` explicitamente.
- **Produção:** `Current()` usa `sync.Once` e carrega um snapshot imutável. `read` devolve `bytes.Clone`, e a falta de hot-reload está registrada no comentário "Restart to reload". Um override inválido falha sem fallback, e `TestExplicitRootNeverFallsBack` cobre isso.
- **i18n:** o contrato próprio, com fallback só para catálogos, está documentado em `DEPLOYMENT.md:207`, com a ressalva explícita de que esse fallback nunca vale para web assets.

**Fica uma observação que não bloqueia a aprovação.** Entre o `Lstat` dos componentes e `root.Open` existe uma janela TOCTOU: alguém que consiga escrever em `assets/` poderia trocar um arquivo por um symlink interno nesse intervalo. O `os.Root` continua impedindo a fuga da raiz, a carga acontece uma única vez na inicialização e esse atacante já precisaria de escrita na distribuição. Por isso não considero impeditivo. Se quiserem fechar a janela, comparem `os.SameFile` entre o resultado do `Lstat` e o `f.Stat()` do arquivo aberto.
