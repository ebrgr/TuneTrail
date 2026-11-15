# Parecer do Advisor (Claude Opus, effort medium) — ISSUE-107

**REPROVADO COM RESSALVAS.** Desenho correto (HTML do servidor, fragmento como melhoria progressiva, sem framework), mas a edição, como descrita, quebra na segunda ação.

## Falhas críticas
1. Um único `submission_token` por página: os ~42 formulários do `manage.html` usam `{{$.Submission}}`; trocar só uma seção deixa as demais com ticket consumido → 409.
2. Limite global de 4096 tickets: GET do `mediaPage` cria ticket; um combobox com debounce lotaria o mapa e bloquearia a administração. Fragmentos de consulta não podem criar ticket.
3. Envios simultâneos com o mesmo token: o segundo falha; serializar.
4. `alias-primary` muda o nome (`<h1>`, `<title>`): atualizar junto.
5. Aviso de `listen-add`/`listen-delete` (`putNotice`) precisa ser consumido (`takeNotice`) no próprio fragmento.

## Desenho
- Carregadores por seção (`loadHeader`, `loadAliases`, `loadArtists`, `loadMetadata`, `loadListens`); templates nomeados reaproveitados pela página completa (`manage/aliases`, …, `search/results`); referência de renderização inalterada.
- Rota única com `?partial=1` (GET) ou cabeçalho `X-TuneTrail-Partial: 1` (POST), no fluxo atual, depois de perfil/sessão/origem. `partial=1` sem cabeçalho, ou com `Sec-Fetch-Mode: navigate`/`Sec-Fetch-Dest: document` → 303 para a URL sem `partial`.
- Cabeçalhos: `text/html`, `X-Robots-Tag: noindex`; busca `private, no-cache` + `Vary: Cookie, X-TuneTrail-Partial`; administração `no-store`; `X-TuneTrail-Fragment: <seção>`. 401/403 sem fragmento (JS navega ao login).
- Ticket: continua de uso único por página; cada resposta de fragmento devolve um novo em `X-TuneTrail-Submission` e no HTML; o JS atualiza todos os `submission_token`; 409 → recarregar, nunca reenviar.
- Sem JS: 303 para `#<seção>` (exceto retorno para `/listens`).
- Busca: mínimo 2 caracteres, até 5 itens por tipo + "ver todos"; debounce 120 ms; `AbortController`; padrão APG combobox/listbox; bfcache como `live.js`.
- JS: `search.js` (live) e `enhance.js` (admin), < 4 KB gzip somados, verificado por teste; tempos em `backlog/evidencias/ui-responsiva/issue-107-tempos.md`.

## Divisão recomendada
- 107a — busca instantânea (pública, baixo risco).
- 107b — edição: âncora no 303, seções, envio por fetch, comboboxes de artista e mesclagem. A ISSUE-108 depende da 107b.

## Perguntas ao usuário
1. Dividir em 107a e 107b?
2. Envio de capa (multipart) pode continuar recarregando a página nesta tarefa?
3. Busca instantânea com 5 itens por tipo e "ver todos"?
