# ISSUE-163 — Verificação

A Conta usa `/admin?section=account` até a ISSUE-125. O formulário compartilhado de saída usa o POST `/admin/logout` com CSRF e ticket de sessão até a confirmação da ISSUE-157. A entrada de login usa `/admin` com rótulo Entrar. O destino padrão do root é `/admin`.

| Papel ou estado | Verificação |
| --- | --- |
| Visitante | Diretório e leitura pública, link Entrar, ausência de ações de edição, 404 igual para perfil protegido e perfil ausente. |
| Conta comum | Login para Conta com perfil público e protegido, links Minha conta e Meu perfil no HTML bruto, 403 em acesso direto à administração global. |
| Dono | Histórico próprio, inclusive protegido, com atalho de gestão de escuta. Edição do próprio perfil com ajuda do relay associada ao checkbox. Ações globais de catálogo ausentes. |
| Conta comum em perfil de terceiro | Leitura pública sem atalhos de gestão. Acesso direto à gestão negado e leitura protegida indistinguível da ausente. |
| Root | Leitura protegida, atalhos administrativos e de catálogo, imagem protegida pela URL legada e variante do aviso do relay. |
| Sessão expirada, revogada ou rebaixada | Os testes de sessão e renderização verificam perda das permissões em novas requisições. Um ticket do menu só encerra a sessão que o recebeu, com CSRF válido. |

A matriz usa a tabela real de rotas em `engine/navigation_issue163_test.go`, além dos testes das ISSUE-109, 139, 140, 141, 159, 160 e 161. Foram conferidos GET e HEAD, os status e cabeçalhos de cache, o retorno depois do login, o POST de saída e o acesso direto às operações. A regressão da 161 compara corpo e cabeçalhos das negativas de `interest`, `aliases`, `top/albums` e `first-activity`, testa a imagem legada acessível ao root e confirma a rejeição de PATCH com `role` e `user_id`.

O teste `TestRejectedLoginRunsBcrypt159` observou uma comparação por tentativa para conta inexistente ou pendente, com o mesmo hash fictício e custo padrão. `TestAccountsMigrationPreservesData` conferiu que o hash da credencial ativa sobrevive à migração das fixtures v3 e v4 com o mesmo custo. A verificação não mede tempo decorrido.

## Navegador

O servidor isolado de `scripts/serve_roadmap06.py` iniciou em um diretório novo. A conta fictícia Beta foi ativada e recebeu proteção de leitura na cópia migrada da fixture. As capturas foram feitas em Chromium com viewport de 320, 390, 768 e 1440 px nos temas Yuu e Pearl. Cada combinação inclui diretório, 404, Conta, Meu perfil, 403, menu root e histórico protegido. Foram conferidos os limites horizontais do documento e do menu aberto. O menu do diretório também foi inspecionado pela árvore de acessibilidade do Orca. [Capturas](capturas)

Com JavaScript desativado no Chromium, Tab exibiu o atalho para o conteúdo, Enter moveu o foco para seu destino e a ordem no menu seguiu por Escolher perfil, Tema e idioma, Minha conta, Meu perfil, Sair… e Sobre. O link da Conta anunciou `aria-current="page"`. O snapshot de acessibilidade mostrou «Beta /beta Protegido» e a opção de exigir login. A jornada nativa abriu a gestão de escuta com Enter e encerrou a sessão pelo formulário da 403. A leitura foi inspecionada pela árvore de acessibilidade do navegador. A validação com um leitor de tela em execução e o aceite visual do usuário seguem pendentes.

`make ui.reference` gerou a referência em diretório novo. A comparação com a referência anterior apontou 31 mudanças entre 169 arquivos. As diferenças incluem as ações por papel, os textos e os hashes do CSS e JavaScript. A referência automatizada foi atualizada com a saída do gerador.

## Checagens automatizadas

Passaram os testes Go dos pacotes afetados e `make check`, incluindo vet, a comparação exata da referência e as 13 jornadas de ponta a ponta. O Graphify foi atualizado por extração AST da worktree.
