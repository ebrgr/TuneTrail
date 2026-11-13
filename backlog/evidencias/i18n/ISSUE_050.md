# Issue 050 — fundação de idiomas

10/11/2026 · [Issue](<../../tasks/issue-050 - 050-—-Fundação-e-escolha-de-idioma.md>) · [Estudo](<../../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>).

Catálogos externos `langs/en.json` e `langs/pt-BR.json`, com `go-i18n/v2` 2.5.1 (MIT) e `x/text/language` já presente. O binário não incorpora os arquivos. São carregados no startup; caminho explícito `TUNETRAIL_LANGS_DIR` tem prioridade. A imagem Docker inclui `/app/langs`. Edição exige reiniciar.

A POC adotou go-i18n pela seleção CLDR, mensagens e suporte a catálogos; `x/text/message` também foi considerado no estudo. A versão 2.5.1 está no cache local e declara Go 1.18, compatível com Go 1.25; em duas provas mínimas equivalentes, ambas carregando o mesmo JSON externo e imprimindo o mesmo título, os executáveis locais com `-trimpath -ldflags="-s -w"` mediram 3.447.458 bytes (go-i18n) e 2.382.626 bytes (x/text/message), macOS arm64. A comparação mostra custo aproximado de programas mínimos, não diferença comprovada no binário completo nem medição de memória. O binário local do TuneTrail com a implementação, sem stripping, mede cerca de 25,9 MB; extração automática de HTML não foi demonstrada. O formato JSON atual suporta mensagens simples com parâmetros nomeados; variantes plurais nos arquivos ficam para 053.

| Tamanho | Inglês | Português brasileiro |
| --- | --- | --- |
| Desktop 1440×900 | [Antes da troca](issue-050/desktop-en.png) | [Após a troca](issue-050/desktop-pt-br.png) |
| Celular 390×844 | [Antes da troca](issue-050/mobile-en.png) | [Após a troca](issue-050/mobile-pt-br.png) |

Capturas em instância temporária com dados fictícios, fontes externas bloqueadas e JavaScript desligado. O seletor foi usado pelo formulário, sem transbordamento horizontal nas duas larguras. O login/admin também exibiu o seletor. A primeira entrega traduz somente a seleção de perfis e o próprio seletor; a interface restante continua em inglês até 051–053, inclusive em páginas com `lang` da preferência escolhida. Essa limitação de acessibilidade e cobertura deve ser corrigida nessas issues; isto não é uma auditoria completa.

Testes dirigidos cobrem negociação, cookie, origem, retorno local, catálogos inválidos/ausentes, fallback, placeholders, FIFO e solicitações simultâneas em idiomas diferentes. O Advisor reviu a implementação, apontando os ajustes de diretório, FIFO e chaves estáveis; foram incorporados. Os dados do banco operacional não foram usados nem alterados.

Validação final: `CGO_ENABLED=0 go test -count=1 ./...`, testes com `-race` nas áreas de i18n/web/admin/engine, `go vet`, builds Linux amd64/arm64 sem CGO e build Docker passaram. O contêiner temporário carregou `/app/langs` e respondeu em pt-BR; foi removido. A imagem local de teste permanece disponível.

## Seletor dentro das configurações — 10/11/2026

O seletor foi retirado do corpo visível das páginas e movido para os diálogos de configurações público e administrativo. Sem JavaScript, uma seção recolhida de configurações apresenta o mesmo formulário. Testado com troca de idioma nos dois caminhos e nas duas larguras. O cookie e a resolução automática pelo navegador foram mantidos.

| Tamanho | Configurações públicas | Configurações administrativas |
| --- | --- | --- |
| Desktop | [Captura](issue-050-settings/desktop-public.png) | [Captura](issue-050-settings/desktop-admin.png) |
| Celular | [Captura](issue-050-settings/mobile-public.png) | [Captura](issue-050-settings/mobile-admin.png) |

As capturas da seção anterior registram a posição inicial, antes deste ajuste. Advisor/Sol revisou a mudança e aprovou após a inclusão da alternativa sem JavaScript.
