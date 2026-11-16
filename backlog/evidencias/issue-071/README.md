# ISSUE-071 — Navegação responsiva

A decisão de usar a barra inferior foi aceita pelo usuário antes desta implementação. O aceite visual do AC #4 permanece pendente. A referência técnica deste diretório não representa esse aceite.

## Comportamento e decisões

- Abaixo de 40rem: navegação inferior fixa com Início, Rankings, Histórico, Busca e Mais; área segura do dispositivo e espaço reservado até o fim do footer. O cabeçalho mantém avatar de 32px em link de 44px e nome compacto.
- De 40rem a 64rem: lateral de ícones com nomes acessíveis e tooltips. A partir de 64rem: lateral com perfil, rótulos e seletor de período; checkbox nativo permite recolher os rótulos sem JavaScript. Tooltips são suprimidos quando os rótulos estão visíveis.
- Rankings abre `/perfil/chart/top/artists`, o ranking existente de artistas. Os demais rankings continuam acessíveis pelos destaques da Home. Histórico abre `/perfil/listens`; Retrospectiva continua disponível em Mais.
- Mais é um `details` nativo: troca de perfil, tema/idioma (Palette), administração (Settings2) e Retrospectiva. Os links são reais; diálogos são aprimoramentos opcionais. Na prévia fictícia, os destinos demonstrativos usam âncoras/diálogos e não abrem administração operacional.
- A segunda barra da ISSUE-123 fica acima da barra inferior no celular. No desktop preserva os modos empurrar/sobrepor. Abrir essa barra fecha Mais; Esc devolve o foco ao gatilho. O painel de tema/idioma da ISSUE-110 permanece compartilhado.
- Período e fuso seguem nos links de Início, Rankings, Histórico e Busca, sem novo cookie. O seletor da lateral reutiliza os destinos e filtros da página; páginas sem seletor próprio usam links para a Home. URLs continuam compartilháveis e preservam a seleção ao recarregar.
- Sem JavaScript, os destinos de produção e Mais funcionam; a administração usa o fallback de documento. O painel de tema abre nativamente e o formulário de idioma continua disponível. A troca de cores ainda exige JavaScript, conforme o contrato preexistente da ISSUE-110.

## Evidências

- `antes/`: 132 PNGs gerados **antes de editar o código**, de uma exportação do HEAD `f1b15b8`, com o aceite da barra inferior já registrado.
- `depois/`: 132 PNGs da referência técnica final, 11 páginas × Yuu/Pearl × larguras 320, 360, 390, 768, 1024 e 1440; viewport de 900px, DPR 1. Retrospectiva é capturada integralmente pelo script existente.
- As páginas administrativas foram recapturadas após a correção final da altura da segunda barra no desktop; as demais capturas permanecem iguais, pois essa regra só afeta a segunda barra aberta.
- Os manifestos registram overflow estrito, navegador e falhas de fontes externas. Fontes Google permanecem parte do runtime; falhas de rede não são mudança desta issue.
- `responsive.json`: 598 vistas, 23 páginas × dois temas × 13 larguras, incluindo os limites 639/640/641, 1023/1024/1025 e 1439/1440/1441; fontes externas bloqueadas para testar também o fallback. Nenhuma rolagem horizontal da página.
- `estados/navigation.json`: 72 vistas, público/admin/prévia × dois temas × seis larguras × JavaScript ligado/desligado. Confere More por Enter, sequência de Tab, popover nativo, nomes/estado expandido na árvore de acessibilidade, alvos de toque de 44×44, recolhimento desktop, segunda barra e clearance do footer. Em desktop, confere também deslocamentos de 296px no modo empurrar e 224px no modo sobrepor.
- `estados/`: capturas de Mais, lateral recolhida, segunda barra empurrando/sobrepondo e snapshots de acessibilidade.
- `reference/`: fixtures técnicas fictícias usadas por `make ui.reference.check`. A referência histórica da ISSUE-069 foi preservada. Os bytes exportados mantêm espaços dos templates e CRLF de SVGs existentes; a checagem de whitespace do diff exclui esse espelho gerado, pois normalizá-lo quebraria a comparação técnica.

A verificação para leitores de tela usa a árvore de acessibilidade real do Chromium (nomes, papéis e estado expandido nativo), com percurso de teclado. **Não houve teste manual com VoiceOver**. O aceite visual e a avaliação manual com leitor de tela ficam disponíveis para a revisão do usuário; AC #4 não foi marcado.

## Reprodução

Playwright fica instalado fora do repositório. Nenhuma etapa usa banco operacional.

```sh
make ui.reference OUT=/tmp/issue071-reference-novo
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/capture_ui_base.cjs /tmp/issue071-reference-novo /tmp/issue071-capturas
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/validate_navigation_071.cjs /tmp/issue071-reference-novo /tmp/issue071-estados
PLAYWRIGHT_MODULE=/tmp/tunetrail-069-tools/node_modules/playwright \
  node scripts/validate_responsive.cjs /tmp/issue071-reference-novo /tmp/issue071-responsive.json
make check
make ui.check
```

Os checks incluem testes de rotas reais com preservação de período/fuso, traduções, referência técnica, `check_web_style.py` e as 12 jornadas end-to-end. Não houve push, merge, publicação ou mudança para Done.
