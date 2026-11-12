# Retrospectiva — issue 057

Data: 10/11/2026. Referência: imagem fornecida pelo usuário, na qual os meses finais ficavam ocultos e o checkbox dependia do botão Aplicar. As capturas abaixo usam dados fictícios.

- [Desktop — português](desktop-pt-mensal.png)
- [Celular — português](mobile-pt-mensal.png)

Os doze meses permanecem visíveis; meses futuros continuam desabilitados. Mensal e Anual aparecem centralizados acima do título. O checkbox envia o formulário ao ser alterado, preservando ano, mês e fuso; com JavaScript desativado, o botão Aplicar permanece disponível. Validado por teclado, em desktop e celular, sem rolagem horizontal nas larguras testadas.

## Ajustes comparados com o Koito original

O serviço original em `127.0.0.1:4111/rewind` e a imagem fornecida serviram de referência. O ano está centralizado com setas, o título dentro do cartão, os subtítulos ao lado das imagens e as linhas usam a cor principal do tema. A faixa de datas e o fuso foram mantidos no cartão. As capturas usam dados fictícios diferentes do original.

- [Original — desktop](original-desktop.png)
- [TuneTrail — desktop, tema claro](desktop-pearl-final.png) · [tema escuro](desktop-yuu-final.png)
- [TuneTrail — celular, tema claro](mobile-pearl-final.png) · [tema escuro](mobile-yuu-final.png)

As setas foram verificadas também sem JavaScript e mantêm fuso e preferência de tempo. As cores das divisórias acompanham o tema; não houve rolagem horizontal nas resoluções testadas.

## Refinamento das linhas e listas

A data fica próxima do título, com uma linha deslocada à esquerda abaixo dela. Cada estatística possui sua própria linha deslocada, na cor principal do tema. As listas ficam limitadas à altura das imagens; o conteúdo adicional pode ser rolado por mouse, toque ou teclado, inclusive sem JavaScript. Os exemplos usam banco descartável com cinco artistas, álbuns e faixas e títulos longos.

- [Desktop claro](desktop-pearl-linhas.png) · [escuro](desktop-yuu-linhas.png)
- [Celular claro](mobile-pearl-linhas.png) · [escuro](mobile-yuu-linhas.png)

Verificado: nove linhas de estatísticas, cores do tema, altura das listas, rolagem por teclado e ausência de rolagem horizontal nas larguras testadas. As capturas anteriores permanecem como histórico.

## Ajuste de espaçamento

Reduzido o espaço entre o título, a contagem e os itens. No cenário de cinco itens com nomes longos, o conteúdo coube na altura da imagem em desktop; em celular, onde a imagem é menor, a lista continua rolável.

- [Desktop claro](desktop-pearl-compacto.png) · [escuro](desktop-yuu-compacto.png)
- [Celular claro](mobile-pearl-compacto.png) · [escuro](mobile-yuu-compacto.png)

## Ajuste responsivo posterior

Aumentado levemente o espaço entre os itens. Até 900 px os cartões exibem o conteúdo completo sem rolagem interna; em celulares estreitos a imagem fica acima do texto. As descrições anteriores representam o comportamento registrado no momento de cada captura.

Capturas finais: [320 px](responsivo-320.png) · [390 px](responsivo-390.png) · [768 px](responsivo-768.png) · [900 px](responsivo-900.png). Testado sem JavaScript, com nomes longos e cinco itens, sem corte de conteúdo nem rolagem horizontal.

## Conteúdo sempre visível

Após avaliação do usuário, removidos o limite de altura e a rolagem interna das listas em todas as larguras. Os itens podem ultrapassar a altura da imagem; os registros anteriores refletem versões intermediárias.
