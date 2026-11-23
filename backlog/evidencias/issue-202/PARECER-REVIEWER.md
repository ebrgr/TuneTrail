# ISSUE-202 — Parecer do Reviewer (23/11/2026)

Revisão do diff `main...HEAD` feita por quem implementou, sem delegar. `make check` verde com 13 de 13 jornadas do e2e, inclusive as de migração, que passam no schema 22.

**Veredito: APROVADO.** Nenhum bug. Dois riscos de operação para o usuário saber.

## Verificado

- [OK] Migração 022 aditiva: uma coluna nula com CHECK de formato (`#rrggbb` minúsculo). Perfis existentes ficam com `NULL`, e o ícone e a foto não mudam (`TestAvatarColorMigration`, que também confere o backup em schema 21).
- [OK] Só a paleta fixa de 12 cores é aceita, em `admin.SetAvatar`, antes de qualquer imagem ser gravada, então um valor recusado não deixa arquivo órfão. Recusados no teste: `red`, `#123456`, `#FFFFFF`, `#00000`, `url(x)` e `#000000" onload="x`, com o valor guardado intacto (`TestAvatarColorForm202`).
- [OK] Defesa em profundidade: o template só escreve `data-avatar-color` se o valor está na paleta (`webassets.AvatarColor`). Um valor ruim já gravado, como `#abcdef`, não chega à página.
- [OK] Contraste: as 12 cores têm 4,92:1 ou mais com o texto escolhido (a pior é o âmbar). O texto vira claro nos fundos escuros e escuro nos claros. As regras do CSS e a paleta do Go são comparadas por teste (`TestAvatarPaletteMatchesCSS`).
- [OK] Semântica da cor: campo ausente mantém a cor; vazio volta ao padrão do tema; «Usar iniciais» limpa ícone e foto e mantém a cor. Vale para o formulário e para `PUT /apis/web/v1/profiles/{slug}/avatar`.
- [OK] No navegador, em instância isolada: o select persiste depois de recarregar, o banco guarda `disc|#000000`, e a cor aparece no card do diretório (56 px), na navegação (18 px), no cabeçalho (32 px) e na lista de contas (32 px), com o ícone e as iniciais legíveis em Yuu e Pearl. Sem JavaScript o formulário envia direto (303), coberto pelo teste de formulário.
- [OK] Dois testes antigos mudaram por causa do schema 22, sem perder cobertura: `TestActiveGuardUpgrade160` roda no schema 5 e agora escreve a conta pendente por SQL, porque a loja consulta a coluna nova; e o `Down` da 022 recusa com `restore_verified_backup`, como a 021, que o `TestSchema021DevelopmentUpgrade` espera.

## Riscos

- [RISCO] A primeira execução desta versão em um banco existente migra o arquivo de schema 21 para 22 no lugar. O backup automático é feito antes, mas não há volta pelas migrações: o caminho de volta é restaurar o backup.
- [RISCO] A ISSUE-195 consolida as migrações num salto único da v2 para o schema final. A 022 passa a fazer parte dessa conta e precisa ser incluída lá.

## Fora do alcance

- Capturas de tela da lista e das cores: o aceite visual é do usuário.
- `DB_ALTERATIONS.md` (arquivo fora do Git) recebeu a seção do schema 22.
