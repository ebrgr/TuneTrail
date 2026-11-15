# Parecer do Reviewer (Claude Opus) — ISSUE-109

Primeira rodada: **um item obrigatório**, resto aprovado.

1. **Obrigatório — `assets/js/live.js` (navegação de histórico sem bfcache).** Sem bfcache, o navegador pode montar o documento a partir do cache HTTP sem revalidar; `pageshow.persisted` é false e `activate(false)` confiaria num `data-admin` antigo após logout. **Corrigido:** `activate(performance.getEntriesByType('navigation')[0]?.type === 'back_forward')`, que revalida com o HEAD (sem laço: depois de `location.reload()` o tipo é `reload`). Verificado com `historico_sem_bfcache.py` (Chrome sem bfcache): sessão ativa mantém selo e controles sem recarregar; após logout, nenhum selo, `data-admin` ou controle. Nesta execução o Chrome buscou o HTML novo; a correção cobre o caso em que ele reaproveita o cache.

Sem problemas: `oneCookie` rejeita cookie duplicado; UUID nulo/inválido recusados; sessão expirada filtrada; leitura sem efeito colateral; papel admin exigido; `X-TuneTrail-Admin` só reflete o cookie da própria requisição; 200 com `private, no-cache` e `Vary: Cookie`, erros com `no-store`; HEAD pelo mesmo `render`; nome acessível sem duplicação; alvo de 44 px.

Sugestões opcionais: (1) registrar falha do banco ao ler a sessão — **aplicada**; (2) a consulta roda antes do 404, custo mínimo — mantida; (3) teste de HEAD e de cookie duplicado — **aplicada**; (4) linha do `live.js` dentro do limite de 160 — confirmada; (5) `title` pouco útil no toque — aceito, o `sr-only` cobre leitores de tela.
