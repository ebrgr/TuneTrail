# Issue 052 — administração multilíngue

Data: 10/11/2026. Implementada, aguardando aceite. [Issue](<../../tasks/issue-052 - 052-—-Internacionalizar-administração.md>).

A navegação administrativa, login, perfis, configuração, importação e exportação, chaves, relay, edição e confirmação de exclusão/mesclagem usam inglês ou português brasileiro conforme a preferência do visitante. As mensagens são carregadas dos arquivos externos `langs/`; textos de catálogo, nomes de variáveis, valores e formatos de API permanecem intactos. Avisos após redirecionamento são traduzidos no idioma da página de destino.

## Comparação visual

Capturas com dados fictícios, JavaScript desativado, Chrome, 1440 × 900 e 390 × 844. Cada comparação apresenta a mesma página nos dois idiomas; não foi detectada rolagem horizontal nas telas verificadas.

| Tela | Desktop inglês | Desktop português | Celular inglês | Celular português |
| --- | --- | --- | --- | --- |
| Login | [ver](issue-052/desktop-en-US-login.png) | [ver](issue-052/desktop-pt-BR-login.png) | [ver](issue-052/mobile-en-US-login.png) | [ver](issue-052/mobile-pt-BR-login.png) |
| Perfil | [ver](issue-052/desktop-en-US-profiles.png) | [ver](issue-052/desktop-pt-BR-profiles.png) | [ver](issue-052/mobile-en-US-profiles.png) | [ver](issue-052/mobile-pt-BR-profiles.png) |
| Configuração | [ver](issue-052/desktop-en-US-settings.png) | [ver](issue-052/desktop-pt-BR-settings.png) | [ver](issue-052/mobile-en-US-settings.png) | [ver](issue-052/mobile-pt-BR-settings.png) |
| Edição de faixa | [ver](issue-052/desktop-en-US-track.png) | [ver](issue-052/desktop-pt-BR-track.png) | [ver](issue-052/mobile-en-US-track.png) | [ver](issue-052/mobile-pt-BR-track.png) |
| Revisão de exclusão | [ver](issue-052/desktop-en-US-review.png) | [ver](issue-052/desktop-pt-BR-review.png) | [ver](issue-052/mobile-en-US-review.png) | [ver](issue-052/mobile-pt-BR-review.png) |

## Verificação

- `CGO_ENABLED=0 go test -count=1 ./...` e `CGO_ENABLED=1 go test -race -count=1 ./internal/adminweb ./internal/adminui ./engine ./cmd/api`: aprovados.
- `scripts/validate_admin_webui.py` com banco temporário: aprovado, incluindo login, permissões, importação, exportação e integridade SQLite.
- `go vet` nos pacotes administrativos e engine, `node --check internal/webui/static/admin.js` e `git diff --check`: aprovados.
- Testes verificam isolamento de idioma entre requisições simultâneas, formulário e ações invariantes, aviso após redirecionamento, acessibilidade, proteção CSRF, limitação de login, respostas JSON e `no-store`.
- Datas, tempos relativos e plurais permanecem no escopo da issue 053; a revisão visual abrangente permanece na 054.
