# Parecer do Reviewer (Claude Opus) — ISSUE-124

Primeira rodada: **dois itens obrigatórios de corretude**, segurança aprovada.

1. **Fragmento de busca perdia `return_to`:** a URI com `partial=1` era recusada pelo `SafeReturn` e caía na página base, perdendo `tz` e `artist_query`; a ação seguinte voltava ao fuso padrão. **Corrigido:** `partial` removido antes do `SafeReturn`.
2. **Resultados da busca sumiam depois de adicionar ou remover artista** (regressão em relação ao fluxo sem JavaScript). **Corrigido:** a busca é refeita no POST da seção de artistas.

Os dois casos estão em `engine/adminui_sections_124_test.go`; o teste falha sem cada correção e passa com elas.

Conferido sem problema: CSRF (cabeçalho próprio exige preflight entre origens; falha nunca devolve fragmento); ticket ecoado só depois de `uuid.Parse`; saldo de tickets estável; abas independentes; `MediaExists` antes do fragmento; 303 sem `partial` montado pelo perfil e âncora só no próprio caminho; reexibição de valores em erro; `listen-delete` do histórico sem `data-enhance`; listeners delegados e envios serializados.

Sugestões aplicadas: aviso consumido só pela página completa ou pela seção de escutas; cabeçalhos do fragmento definidos depois da renderização; mensagens da seção sem papel ARIA próprio (a região de status anuncia uma vez); depois de uma página de erro vinda de POST completo (capa), voltar à página de gestão em vez de recarregar o POST; o `artist_query` escondido do formulário de fuso acompanha a busca. Também removidos dois arquivos de trava de um banco de teste (`engine/PjTDlAXa/`) que tinham entrado por engano no commit 3dba239 da ISSUE-123.

Não aplicada: atualizar o bloco `#selected-listen` quando a mesma escuta é excluída pela lista — o bloco só aparece ao chegar pelo histórico e a exclusão por ele volta ao histórico.
