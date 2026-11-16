# ISSUE-141 — Parecer do coordenador (Claude Sonnet, esforço médio)

Revisão do diff `main..HEAD` (commit `79d7de9`) feita pelo coordenador, por decisão do usuário; sem Reviewer externo.

**Parecer: APROVADO, sem itens obrigatórios.**

## O que foi conferido
- **Segurança:** a exclusão continua passando por sessão, origem, CSRF, ticket de uso único e escopo de perfil (`listen-delete` da ISSUE-139); a resposta 204 só confirma o instante removido (`X-TuneTrail-Listen-Deleted`) e o cliente só a aceita quando coincide com a linha aberta. O cabeçalho `X-TuneTrail-History-Refresh` só tem efeito com sessão de administrador (`AllowEmptyMedia = p.Admin && cabeçalho`); sem sessão a página vazia continua 404 (coberto em `TestListenDialog141`).
- **Visitante:** o link "Gerenciar" agora só é emitido para administrador (`{{if $page.Admin}}`); antes era público. O × só nasce por JavaScript e só em sessão revalidada.
- **Exclusão rápida antiga:** removido o `DELETE` do `live.js`; nenhum caminho apaga sem a janela de confirmação.
- **Isolamento:** testes com dois perfis, CSRF, ticket, perfil e slug forjados, retorno de outro perfil, origem e sessão expirada.
- **Visual:** janela com detalhes (faixa, artistas e álbum com links, data e hora com fuso, cliente) e o aviso de que não existe desfazer; Yuu e Pearl, desktop e celular, sem sobreposição.
- Testes executados no worktree: `engine`, `internal/webui`, `internal/adminui` (filtro Listen/Task/Dialog/Media) passam; `check_web_style.py` sem apontamentos; nenhuma credencial nos arquivos versionados.

## Observações (não bloqueiam)
1. Depois de excluir a última escuta de uma faixa, álbum ou artista, a janela atualiza a página mostrando o estado vazio; recarregar a página (GET normal) dá 404, como já acontecia antes desta tarefa. Conferir se esse é o comportamento desejado.
2. `CurrentURL` da página de mídia passou de `/listens?...` filtrado para a própria página de mídia (decisão da ISSUE-139, retorno a partir da mídia); o valor continua validado por `SafeHistoryReturn`.
3. O texto em português usa "reprodução" na janela e "escuta" em outros pontos; é apenas de vocabulário.
