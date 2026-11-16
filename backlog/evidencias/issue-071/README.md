# ISSUE-071 — Navegação responsiva

A decisão de usar a barra inferior foi aceita pelo usuário antes desta implementação. O aceite visual do AC #4 permanece pendente. A referência técnica deste diretório não representa esse aceite.

## Comportamento e decisões

- Abaixo de 40rem: navegação inferior fixa com Início, Rankings, Histórico, Busca e Mais; área segura do dispositivo e espaço reservado até o fim do footer. O cabeçalho mantém avatar de 32px em link de 44px e nome compacto.
- De 40rem a 64rem: lateral de ícones com nomes acessíveis e tooltips. A partir de 64rem: lateral com perfil, rótulos e seletor de período; a escolha Com rótulos/Só ícones fica no painel de tema. Sem JavaScript, o padrão continua com rótulos. A preferência de rótulos é apenas uma prévia no documento atual enquanto se decide a persistência no servidor. Tooltips são suprimidos quando os rótulos estão visíveis.
- Rankings abre `/perfil/chart/top/artists`, o ranking existente de artistas. Os demais rankings continuam acessíveis pelos destaques da Home. Histórico abre `/perfil/listens`; Retrospectiva continua disponível em Mais.
- Mais é um `details` nativo: troca de perfil, tema/idioma (Palette), administração (Settings2) e Retrospectiva. Os links são reais; diálogos são aprimoramentos opcionais. Na prévia fictícia, os destinos demonstrativos usam âncoras/diálogos e não abrem administração operacional.
- A segunda barra da ISSUE-123 fica acima da barra inferior no celular. No desktop preserva os modos empurrar/sobrepor. Abrir essa barra fecha Mais; Esc devolve o foco ao gatilho. O painel de tema/idioma da ISSUE-110 permanece compartilhado.
- Período e fuso seguem nos links de Início, Rankings, Histórico e Busca, sem novo cookie. O seletor da lateral reutiliza os destinos e filtros da página; páginas sem seletor próprio usam links para a Home. URLs continuam compartilháveis e preservam a seleção ao recarregar.
- Sem JavaScript, os destinos de produção e Mais funcionam; a administração usa o fallback de documento. O painel de tema abre nativamente e o formulário de idioma continua disponível. A troca de cores ainda exige JavaScript, conforme o contrato preexistente da ISSUE-110.

## Evidências

- `antes/`: 132 PNGs gerados **antes de editar o código**, de uma exportação do HEAD `f1b15b8`, com o aceite da barra inferior já registrado.
- `depois/`: 132 PNGs da implementação inicial, preservados por decisão do usuário após a ressalva; 11 páginas × Yuu/Pearl × larguras 320, 360, 390, 768, 1024 e 1440; viewport de 900px, DPR 1. Retrospectiva é capturada integralmente pelo script existente.
- As páginas administrativas foram recapturadas após a correção final da altura da segunda barra no desktop; as demais capturas permanecem iguais, pois essa regra só afeta a segunda barra aberta.
- Os manifestos registram overflow estrito, navegador e falhas de fontes externas. Fontes Google permanecem parte do runtime; falhas de rede não são mudança desta issue.
- `responsive.json`: 598 vistas, 23 páginas × dois temas × 13 larguras, incluindo os limites 639/640/641, 1023/1024/1025 e 1439/1440/1441; fontes externas bloqueadas para testar também o fallback. Nenhuma rolagem horizontal da página.
- `estados/navigation.json`: 108 vistas (incluindo os dois modos de rótulos com JS), público/admin/prévia × dois temas × seis larguras × JavaScript ligado/desligado. Confere More por Enter, sequência de Tab, popover nativo, nomes/estado expandido na árvore de acessibilidade, alvos de toque de 44×44, alternância no painel, período nativo em Mais nas laterais de ícones, segunda barra e clearance do footer. Em desktop, confere deslocamentos de 282px/116px no modo empurrar e 224px/58px no modo sobrepor, com rótulos/só ícones respectivamente. Verifica também encaixe do heatmap inteiro em 1440px.
- `estados/desktop-yuu-labels.png` e `estados/desktop-yuu-icons.png`: as únicas duas imagens atualizadas para a ressalva, em 1440px/Yuu. As demais imagens de estados e os 132 PNGs de depois são históricos preservados, não retratam esta revisão.
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


## Ressalva — persistência no servidor pendente

A escolha saiu da lateral e foi para o painel de tema/idioma, junto de empurrar/sobrepor. A faixa de ícones agora usa os mesmos estilos de 58px em 40–64rem e no desktop sem rótulos; o nome permanece somente no cabeçalho, com “Escolher um perfil” no link da lateral. Em contêineres largos, as colunas do heatmap se distribuem no espaço disponível, evitando corte na borda direita. O layout da barra inferior foi preservado.

O `conf.toml` existente guarda configurações técnicas por `cfg.SettingsStore`, mas o esquema `tomlDocument` e `managedDefaults` não contêm preferências de UI. `SettingsStore.Save` recusa campos desconhecidos; o tema padrão vem de variável de ambiente, fora desses campos editáveis. Não há contrato existente para salvar esses dois modos por servidor/perfil.

Por instrução do usuário, a implementação da persistência foi interrompida e a dúvida foi registrada na tarefa. Não se criou arquivo/formato novo. A nova escolha de rótulos muda apenas o atributo `data-nav-labels` no documento atual, sem introduzir gravação no navegador. O modo empurrar/sobrepor mantém temporariamente o armazenamento local preexistente da ISSUE-123. Portanto, **persistência no servidor e renderização de preferências salvas sem JS ainda não estão entregues**.

Opções para decisão: arquivo novo de UI; extensão do `conf.toml` atual; preferências por perfil no banco ou configuração por ambiente. Recomendação: estender o `conf.toml` existente para defaults globais, reutilizando revisão, validação e gravação atômica, com escrita autenticada de administrador e valores aplicados pelo servidor aos atributos HTML. Banco é a alternativa adequada se a preferência precisar ser por perfil. Essa decisão de escopo precede qualquer novo campo ou endpoint.

Não executar a captura em massa para esta ressalva. `validate_navigation_071.cjs` mantém as vistas em JSON e grava somente as duas capturas de 1440px/Yuu citadas acima. AC #4 continua desmarcado.
