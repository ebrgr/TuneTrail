# ISSUE-106 — implementação

Implementada na branch `feat/issue-106-cache-dos-arquivos-estaticos-e`, sobre `bec1dc2` (produto 0.11). Sem migração do banco. Os arquivos continuam externos em `assets/`, carregados no snapshot ao iniciar.

O catálogo calcula SHA-256 dos bytes finais, prepara os temas e reescreve dependências locais do CSS com hash. Templates usam chamadas literais ao helper `asset`; referências inválidas ou privadas impedem a inicialização. URLs corretas recebem cache imutável de um ano; aliases sem query revalidam com ETag; hashes desconhecidos e erros recebem no-store. GET, HEAD, condicionais e Range compartilham o mesmo handler. Licenças e templates permanecem fora das rotas públicas.

HTML público 200 usa private, no-cache e Vary Cookie/Accept-Language. Administração, ações e erros permanecem no-store. O retorno pelo histórico recria controladores e revalida sessão e contexto efetivamente resolvido pelo servidor. Logout, expiração e falha de rede retiram controles de alteração. A administração restaurada recarrega os formulários. Foi corrigida também a referência da capa alternativa para usar a URL local versionada.

## Verificações

- `make check`: aprovado sem cache de resultados Go; inclui referência textual exata, sintaxe JS, testes de tema e 12/12 jornadas e2e. Log: `implementacao-check.log`.
- `make api.race`: aprovado; log `implementacao-race.log`. Uma execução manual anterior sem o ambiente exportado pelo Makefile falhou por localização dos assets; o alvo oficial completo passou.
- Chrome 157.0.8103.51, Firefox 147.0 e Chrome atrás de proxy HTTPS: aprovados com servidores, usuários e bancos fictícios descartáveis. Relatórios `implementacao-{chrome,firefox,https}.json`. Segunda navegação sem requisições à origem para CSS/JS/JSON; bfcache real, consulta e rolagem preservados. Exercitados logout, expiração no servidor, exclusão após restauração, idioma, fuso inválido, seleção de perfil, formulários administrativos e capa alternativa. Falha offline exercitada no Chrome.
- Matriz visual: 23 páginas × 6 larguras (320, 360, 390, 768, 1024, 1440) × 2 temas = 276 comparações contra a base efetiva da 0.11 (065 + atualização 066). Texto e geometria iguais em todas. 249 imagens idênticas; 27 apresentam diferenças de rasterização, com delta máximo de 16 por canal e até 4,06% dos pixels. Não se afirma igualdade pixel a pixel. Fontes externas bloqueadas igualmente nos dois lados: esta medição não substitui o estudo tipográfico da 070. Relatórios `implementacao-renderizacao.json` e `implementacao-pixels.json`; a comparação textual dos 113 arquivos da nova referência passa no gate do projeto. Bases históricas preservadas.
- Reviewer Claude Opus: APROVADO após incorporar os itens obrigatórios. Pareceres `reviewer-inicial.md` e `reviewer-final.md`. Sugestões finais de legibilidade e um breve intervalo de reativação do botão antes da recarga são opcionais, sem bloqueio. Mantido `data-assets` por compatibilidade com validadores existentes.
- Sugestões finais do Reviewer aplicadas depois do parecer: flag `reloading` no DELETE do `live.js`, condição inalcançável removida em `assets.go`, linha em branco do `theme.js`, listeners `pageshow` do `admin.js` unificados e comentário de template sobre `data-assets`. A referência da 106 foi regenerada (29 arquivos; nas páginas, só hashes dos três JS e `data-ui-generation`). `make check` (12/12 jornadas, `differences: []`) e `make api.race` aprovados de novo. Navegadores não foram reexecutados, conforme o parecer.
- Grafo AST atualizado pelo script externo do projeto.

## Limites e publicação

Somente a geração atual é mantida; um hash antigo não entrega bytes novos. Abas antigas podem precisar de recarga, e não há garantia de recuperação se o script antigo já não puder ser carregado. Aliases legados revalidam. Fontes locais pertencem à 070. Não foram alterados tamanhos nem estilos tipográficos.

Aguardando aceite explícito do usuário. Publicação, merge, versão, tag e push dependem de autorização posterior.
