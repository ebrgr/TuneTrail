# ISSUE-157 — Verificação

«Sair…» abre `GET /admin/sign-out` no menu (+), na Conta e na 403. A confirmação atende sessões comuns e root. O botão «Sair» envia o formulário para `POST /admin/logout`, que revoga a sessão atual, limpa o cookie de sessão e limpa o cookie de pré-autenticação quando presente. A resposta 303 leva ao diretório público (`/`), que oferece «Entrar» e leva ao formulário em `/admin`. Esse destino segue o reescopo de 20/11/2026.

«Cancelar» retoma um destino de leitura interno permitido ao ator. Na ausência de um destino válido, o retorno é a Conta (`/admin?section=account`) para comum e root, com o diretório como alternativa se a Conta estiver indisponível. Sessão ausente, expirada ou revogada apresenta o formulário de login com «Entre para continuar.».

| Cenário | Resultado verificado |
| --- | --- |
| GET, HEAD e prefetch | GET exibe a confirmação com `Cache-Control: private, no-store` e `Vary: Cookie`. HEAD devolve cabeçalhos com corpo vazio e prefetch recebe 204. A sessão permanece ativa nos três casos. |
| CSRF e origem | CSRF ausente, inválido ou de outro ator, assim como origem externa, recebe 403. |
| Tickets | Ticket ausente, de outra sessão, reutilizado ou emitido para edição recebe 409 enquanto a sessão está ativa. O limite global permanece em 4096. A confirmação renova o ticket de saída dedicado da sessão por 30 minutos. Páginas públicas e a 403 deixam de emitir os dados do formulário. |
| Duas abas e expiração | Abrir a segunda aba mantém o formulário utilizável. Depois que uma aba encerrar a sessão, o envio da outra retorna 401 com formulário de login e a mensagem neutra. A expiração entre renderizar a página e enviar o formulário tem a mesma resposta. |
| Sessão comum e root | Uma requisição com dois cookies de sessão é rejeitada antes de escolher o ator. A saída de root limpa os cookies da requisição e a sessão comum independente permanece válida. |
| Cancelar | Um endereço interno autorizado é aceito. URL absoluta, destino iniciado por `//`, esquema arbitrário, rota de ação e leitura sem permissão caem no fallback. |
| Idiomas e ícone | A confirmação foi verificada em pt-BR e en. O SVG anterior foi substituído pela geometria oficial do ícone `log-out` do Lucide (ISC) no catálogo local. |

## Navegador

Servidor isolado de `scripts/serve_roadmap06.py` com dados fictícios em diretório novo. Foram feitas oito capturas em Chromium, com larguras de 320, 390, 768 e 1440 px, nos temas Yuu e Pearl. A página manteve as ações dentro da área visível, sem rolagem horizontal. A árvore de acessibilidade do Orca mostrou o título, a descrição, o botão «Sair» e o link «Cancelar». [Capturas](capturas/)

Com JavaScript desativado no Chromium, o teclado abriu «Sair…» no menu (+). O foco inicial foi para «Sign out», Tab avançou para «Cancel» e Enter voltou ao diretório. A confirmação também foi enviada pelo teclado. O diretório ofereceu «Sign in» e o formulário da segunda aba mostrou o login com a explicação neutra. O aceite visual (AC4) fica pendente.

## Checagens

Passaram os testes de `engine`, `internal/adminui`, `internal/adminweb`, `internal/webassets` e `internal/webui`, além de `make check` completo com 13 de 13 jornadas de ponta a ponta.

`make ui.reference` gerou a referência da ISSUE-075 em diretório novo e incluiu a confirmação. Foram substituídas as páginas e o manifesto afetados. A comparação da referência passou em `make check`. O Graphify foi atualizado por extração AST da worktree.
