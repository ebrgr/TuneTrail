# Parecer do Advisor (Claude Opus, effort medium) — segunda barra de administração

Demanda do usuário (14/11/2026): ao clicar em configuração, abrir uma segunda barra de ícones ao lado da atual (telas largas) ou abaixo dela (telas estreitas), com login do administrador, perfis, configuração do servidor e, depois, saúde (ISSUE-083) e logs (ISSUE-112).

**REPROVADO COM RESSALVAS**: a ideia é boa, mas os critérios #2–#4 da 110 descrevem o submenu popover e precisam ser reescritos.

## Falhas críticas
1. `popover="auto"` fecha com qualquer clique fora e vai para a top layer; não serve para uma barra que fica aberta nas páginas `/admin`.
2. Sair exige POST com CSRF; fora da administração, Conta é link para `/admin?section=account`.
3. Os `data-tip` da primeira barra (`left: calc(100% + 22px)`) cairiam sobre a segunda; deslocar pela largura total ou ajustar o z-index.
4. Perfis → `/admin?section=profiles`; Servidor → `/admin/settings`.

## Desenho recomendado
- Disclosure com aprimoramento progressivo: gatilho Settings2 como `<a href="/admin" data-nav-toggle aria-controls="nav-admin">`; o JS compartilhado o torna `role="button"` com `aria-expanded` e alterna `hidden`. Sem JS, o clique abre `/admin`, onde o servidor entrega a barra aberta.
- Estado só no servidor: aberta em `Kind=admin`; fechada e sem persistência nas páginas públicas.
- Teclado: ordem do DOM para Tab; Esc fecha e devolve o foco; `aria-current="page"` no item ativo.
- Segunda `<nav id="nav-admin" class="sidebar sidebar--secondary">` irmã da primeira, no mesmo partial; a ISSUE-071 a transforma no destino "Mais" do celular por CSS.
- Layout: `--nav-w` 58px → 116px com `[data-nav-open]`; `.app-content` e `admin.css` usam a variável; remover o `padding-left:116px` morto de `.admin-sidebar`. Abaixo de 40rem, faixa horizontal abaixo da barra superior.

## Itens
| Item | Visitante | Admin | Ícone |
|---|---|---|---|
| Entrar (`/admin`) | sim | não | LogIn |
| Conta | não | sim, com selo | CircleUserRound ou UserRound |
| Sair (POST + CSRF) | não | só na administração | LogOut |
| Perfis | não | sim | UsersRound ou UserRoundCog |
| Servidor | não | sim | Server ou ServerCog |
| Saúde (083) / Logs (112) | quando existirem | | HeartPulse/Activity, ScrollText |

Selo ShieldCheck no gatilho e em Conta com sessão válida. Prévia: só aviso.

## Contrato
`Nav{Kind, Profile, Admin, AdminOpen, Current, CSRF}`.

## Plano
1. 110 só com tema: Palette, modal de tema, `theme-picker.js`, remoção do `settings-dialog` e do `#admin-settings`; Settings2 vira temporariamente link para `/admin`.
2. Nova ISSUE: segunda barra de administração (depende da 110, alimenta a 071).

## Perguntas
1. "Sobre" na segunda barra ou no rodapé do modal de tema?
2. A segunda barra empurra o conteúdo ou se sobrepõe nas páginas públicas?
3. Separar a segunda barra numa ISSUE própria?
