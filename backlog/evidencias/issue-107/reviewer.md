# Parecer do Reviewer (Claude Opus) — ISSUE-107

**APROVADO**, sem itens obrigatórios.

Conferido: o fragmento roda depois de resolver o perfil e de `db.WithProfile`, com o mesmo `listening.ReadSearch` da página completa; perfil inexistente dá 404; `innerHTML` recebe HTML da mesma origem escapado pelo html/template; dois `Vary` válidos; HEAD sem corpo; limites de 2 e 200 caracteres; prévia sem `search.js`; padrão APG de listbox com grupos; `aria-activedescendant` aponta para a opção; anúncio do total real.

Sugestões opcionais:
1. Location do 303 a partir do perfil, não do caminho — **aplicada**.
2. `setActive(-1)` ao digitar, para o Enter não abrir opção da lista antiga — **aplicada**.
3. Não pedir o fragmento com menos de 2 caracteres — não aplicada: a dica vem traduzida do servidor.
4. Ignorar teclas com `isComposing` (IME) — **aplicada**.
5. Limpar o status antes de preencher, para repetir anúncios iguais — **aplicada**.
6. `aria-controls` apontando para o próprio listbox — não aplicada: o listbox só existe com resultados; o contêiner é tolerado.

Correções feitas depois do parecer, achadas pelo roteiro de navegador: o evento `close` do diálogo chega depois do fechamento; numa reabertura rápida ele apagava o campo (`search.js`) e tirava o foco do campo (`ui.js`). Os dois agora ignoram o evento quando o diálogo já está aberto de novo. Roteiro aprovado em 5 execuções seguidas.

## Segunda rodada — layout do Koito (pedido do usuário antes do aceite)

Diálogo ancorado no topo, cabeçalho fixo e resultados em cartões por tipo (grade de artistas com imagem; álbuns e faixas com miniatura), botão Buscar escondido com JavaScript.

**Um item obrigatório:** o cabeçalho fixo podia cobrir a opção ativa ao navegar com as setas (WCAG 2.4.11). **Corrigido:** `scroll-padding-top` do diálogo igual à altura do cabeçalho antes do `scrollIntoView`. O roteiro sobe da última opção até um cartão de artista acima da área visível: falha sem a correção e passa com ela.

Sugestões aplicadas: título dos grupos com `aria-hidden` (lido pelo `aria-labelledby`), cor sólida de reserva sem `color-mix`, `100dvh`, título do diálogo como `h2`.
