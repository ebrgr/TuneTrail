# Issue 051 — interface pública multilíngue

Implementação em 10/11/2026 · [Issue](<../../tasks/iss-051 - 051-—-Internacionalizar-interface-pública.md>) · [Estudo](<../../docs/roadmaps/doc-006 - Estudo-—-interface-multilíngue-item-3-do-roadmap.md>).

A seleção de perfis, navegação, busca, home, rankings, histórico, artistas, álbuns, faixas e retrospectiva usam mensagens dos catálogos externos `langs/en.json` e `langs/pt-BR.json`. Mensagens dos scripts de aparência e exclusão usam atributos HTML escapados. Cada renderização usa clone próprio do template, com locale específico por requisição; nomes musicais, identificadores, rotas, parâmetros e respostas da API permanecem dados independentes da tradução.

| Largura | Home em inglês | Home em português | Faixa em inglês | Faixa em português |
| --- | --- | --- | --- | --- |
| Desktop | [Captura](issue-051/desktop-en-home.png) | [Captura](issue-051/desktop-pt-br-home.png) | [Captura](issue-051/desktop-en-track.png) | [Captura](issue-051/desktop-pt-br-track.png) |
| Celular | [Captura](issue-051/mobile-en-home.png) | [Captura](issue-051/mobile-pt-br-home.png) | [Captura](issue-051/mobile-en-track.png) | [Captura](issue-051/mobile-pt-br-track.png) |

Capturas feitas com dados fictícios, JavaScript desligado e fontes externas bloqueadas. Não houve transbordamento horizontal nas larguras testadas. Com JavaScript habilitado, a confirmação e as mensagens de falha HTTP e rede da exclusão foram verificadas com respostas simuladas. A comparação visual ampla, com temas e estados adicionais, continua na 054.

Testes dirigidos cobrem rotas públicas nos dois idiomas, links iguais, nomes musicais que coincidem com textos de UI, escaping de dados, erros parciais, retrospectiva preenchida e requisições simultâneas. O Advisor apontou a falha de rede em inglês; foi corrigida e validada, com parecer final aprovado.

Limitações planejadas: `time.Month`, dias da semana, datas e horas, texto relativo, formatação de números e pluralização completa ficam na 053. A administração e suas mensagens próprias permanecem em inglês até a 052. Assim, a UI pública em português ainda pode exibir datas ou unidades com convenções inglesas enquanto a 053 estiver aberta.
