# Parecer do Reviewer (Claude Opus) — ISSUE-123

Primeira rodada: **dois itens obrigatórios**, resto aprovado.

1. **Obrigatório — papel deduzido de `(not .Login)`.** Erros anteriores à autenticação (limite de login 429, `h.fail`) renderizavam com `Login=false` e ofereciam Conta/Perfis/Servidor a um visitante (sem vazamento de dados: os links levam ao login). **Corrigido:** `view.SignedIn()` (verdadeiro só com `User`, preenchido depois da sessão válida). Teste `TestSecondSidebarPreAuthErrorShowsVisitorItems` provoca o 429 e falha com a regra antiga.
2. **Obrigatório — item desabilitado da prévia sem nome acessível** (`<span tabindex=0 aria-label>`). **Corrigido:** `<button type="button" aria-disabled="true">`.

Sem problemas: ids únicos e `aria-controls` válido; estado coerente no bfcache; `role=button` em `<a href>` com Enter nativo e Espaço tratado; sem JS o link leva a /admin com a barra aberta; sem `:has` a barra apenas sobrepõe; z-index e dicas coerentes; faixa estática no celular; ícones do bundle e `rel=noopener`.

Sugestões opcionais:
- "Entrar" na própria página de login — **aplicada** (`aria-current`).
- No modo sobrepor da administração a barra aberta cobre ~14 px do recuo de `.admin-shell` — mantido: é o comportamento escolhido para "sobrepor".
- "(abre em nova aba)" no rótulo de Sobre — **aplicada**.
- Esc só com a barra aberta e sem `defaultPrevented` — **aplicada**.
- `?.` na escolha de layout — **aplicada** (verificação explícita).
