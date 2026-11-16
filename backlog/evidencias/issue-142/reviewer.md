# Parecer do Reviewer — ISSUE-142

Escopo: `git diff main...HEAD` (commit 7a9709e) mais a correção abaixo. `make check` verde, incluindo e2e 12/12.

- [BUG] `assets/js/media-dialog.js` (handler de `drop`): o link da tarefa de imagem era buscado com `cover.closest('.media-heading')?.querySelector('a[data-artwork-task]')`, mas a capa está no `<header class="media-heading">` e o link no `<nav class="media-actions">`, fora dele. Resultado: `link` nulo e o arrasto não fazia nada. **Corrigido** (busca em `document`); testes de contrato ganharam uma asserção de regressão e a contagem exata de um link `data-artwork-task` por página. Os testes não executam navegador: o comportamento do arrasto ainda precisa de verificação manual.
- [RISCO] `media-dialog.js`: mensagens de erro do cliente em inglês fixo, fora do catálogo `tr`/i18n. Aceitável agora; padronizar numa tarefa de textos.
- [RISCO] `artworkMaxBytes` (10 MB) duplica `MaxArtworkBytes` do servidor (`internal/admin/artwork.go`). Se um mudar, o outro diverge; o servidor continua sendo a validação final.
- [OK] Nada é enviado antes de confirmar; prévia revoga a object URL ao cancelar, fechar e após sucesso; sem endpoint novo; `ReplaceArtwork` intacto; `data-artwork-drop` só para administrador; cancelar funciona sem JavaScript.
- [OK] Sem repetição automática em falha de rede/sessão (reaproveita o fluxo do diálogo).
