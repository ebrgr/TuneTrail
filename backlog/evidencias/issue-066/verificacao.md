# ISSUE-066 — Verificação em 14/11/2026

Arquivos web reunidos em `assets/`, com templates público/admin separados e manifesto explícito. O executável lê uma geração do disco na inicialização, valida os arquivos e mantém um snapshot imutável. Alterações exigem reinício; não foi implementada recarga automática. `TUNETRAIL_WEB_DIR` explícito inválido falha sem fallback. Sem configuração, usa `assets/` junto ao executável real, resolvendo symlinks e sem depender do cwd.

## Verificações

- `make api.race`: todos os pacotes passaram com detector de corridas.
- `make check`: Go/vet, JavaScript e 12/12 jornadas e2e passaram, incluindo `web-assets` com binário real em distribuição descartável.
- `make ui.reference.check`: 113 arquivos idênticos à referência aceita da ISSUE-065, sem diferenças. Integra `ui.check` e o CI.
- Testes negativos: manifesto, extensões, arquivo ausente/inválido, limites de leitura, cancelamento, symlinks internos/externos, separação público/admin e templates com referências ausentes.
- Distribuição: executável via symlink e cwd `/`; binário isolado e root explícito inválido falham antes de criar o diretório de dados. Template inválido recusa partida. Alterar CSS após a partida não muda os bytes servidos.
- HTTP: URLs existentes preservadas; MIME explícito e `nosniff`; sem directory listing, HTML-fonte, brand ou licenças nas rotas. Sem introduzir Last-Modified ou Cache-Control antes da ISSUE-106.
- Docker: build local `tunetrail:issue-066` concluído. UI respondeu HTTP 200 com UID/GID 65532 e dados descartáveis em tmpfs. Assets pertencem a root, sem escrita para esse usuário; arquivos 0444. Licença de ícones presente; brand excluído da imagem. USER padrão será tratado pela ISSUE-102.
- Pacote local `.tar.gz`: binário, assets, langs e LICENSE presentes, proprietário root, diretórios/binário 0555 e demais arquivos 0444. Binário e recursos devem ser instalados e trocados juntos.
- `langs/`: contrato próprio documentado, preservando fallback do checkout e resolvendo executável real. Não existe esse fallback para assets.
- Graphify atualizado pelo script AST oficial; revisão deslop e `git diff --check` concluídas.

## Revisão

O Reviewer via Claude Opus aprovou a revisão principal e a reavaliação dos achados (ver `reviewer.md`). Foram corrigidos: gate e2e da distribuição, symlinks internos, cobertura negativa e camada duplicada de assets no Docker.

Após essa aprovação foi acrescentada a validação AST de referências a partials ausentes, com testes e integração público/admin. A tentativa de revisão complementar pelo Claude CLI foi recusada pela aprovação automática por exigir autorização explícita para enviar esse trecho de código ao serviço externo. O usuário autorizou explicitamente o envio ao Claude CLI em 14/11/2026. A revisão complementar foi concluída e APROVADA, sem correções obrigatórias (ver `reviewer-complementar.md`). As três sugestões opcionais foram posteriormente autorizadas e aplicadas: diagnóstico com nome do template chamador, testes específicos dos ramos else de if/range/with e recusa de referência a template associado sem corpo. O Claude aprovou esse diff antes do commit (`reviewer-refinamentos.md`), e `make check` passou novamente (`refinamentos-check.log`), com 12/12 jornadas e 113 arquivos idênticos à referência. A autorização resolveu o bloqueio anterior.

Nenhum push, merge, publicação ou incremento de versão foi realizado. Fontes e tipografia permanecem no escopo da ISSUE-070; HTML/CSS/JS movidos preservam os bytes anteriores.
