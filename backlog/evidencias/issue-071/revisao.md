# ISSUE-071 — Revisão própria do diff

APROVADO para envio a aceite. Auditoria executada pelo próprio agente, sem delegação ou outra persona. Inclui a revisão de ruído/complexidade prevista na skill deslop.

[OK] assets/templates/shared/nav.html:10 — destinos reais em produção; Mais usa details nativo, com perfil/tema/admin acessíveis; links administrativos preservam o perfil e a prévia não abre administração real.
[OK] assets/css/ui/layout/shell.css:1 — três padrões em 40/64rem; reserva de safe-area e footer; foco visível; rótulos recolhíveis nativamente; tooltips não são duplicados quando há texto visível.
[OK] assets/css/admin/admin.css:222 — recuos seguem a largura real da lateral; segunda barra preserva empurrar/sobrepor e a reserva inferior no celular; a altura desktop volta a ocupar todo o viewport.
[OK] internal/webassets/nav.go:68 — período e fuso seguem em query codificada e escapada pelo template; sem cookie, estado global mutável ou nova chamada ao banco.
[OK] internal/webui/preview.go:102 — seletor usa URLs filtradas da página, preservando seleção e fuso; fallback continua disponível nas páginas sem seletor próprio.
[OK] assets/js/nav.js:20 — abrir a segunda barra fecha Mais; foco entra na barra e Esc volta ao controle visível.
[OK] assets/js/ui.js:13 — links de diálogo mantêm fallback de documento; Esc não intercepta o fechamento nativo de popovers abertos.
[OK] langs/en.json:2 — novas mensagens também presentes em pt-BR; CHANGELOG inclui a mudança em Unreleased.
[OK] scripts/validate_navigation_071.cjs:1 — provas em 72 vistas com/sem JavaScript, teclado, árvore de acessibilidade/estado expandido, 44px, footer e modos do admin; 598 vistas sem overflow no roteiro responsivo.

O grafo AST foi atualizado a partir desta worktree. A consulta DefaultPeriods confirmou os consumidores Page.Nav, LivePage.Nav e adminNav; não há mudança de autenticação, banco, API ou dependências externas.

Limitações: a árvore de acessibilidade foi verificada automaticamente no Chromium; não houve teste manual com VoiceOver. O aceite visual permanece com o usuário (AC #4 desmarcado). As demonstrações da prévia e a troca de cores continuam com os limites de JavaScript preexistentes.
