# Roadmap 3 — evidências do estudo e plano de comparação

**Status em 04/11/2026: núcleos 006–008 e integração 009 implementados; comparações visuais registradas
na 009; issues 006–009 encerradas por aceite do usuário em 05/11/2026.**
[Entrega e testes da 006](ISSUE_006.md) · [Entrega e testes da 007](ISSUE_007.md) ·
[Entrega e testes da 008](ISSUE_008.md) · [Integração e comparações da 009](ISSUE_009.md).
Plano e inventário: [ROADMAP_03_MIDIA_RETROSPECTIVA.md](<../../docs/roadmaps/doc-005 - Roadmap-3-—-artistas-álbuns-faixas-e-retrospectiva.md>).

[analysis-checks.json](analysis-checks.json) registra uma execução isolada que
chamou o helper Go `db.TimeframeToTimeRange` e reproduziu as fórmulas atuais de
médias e intervalos do interesse. Janeiro/2026 resulta em denominador 30, diante
de 31 dias civis; fevereiro resulta em 27, diante de 28. Um exemplo de interesse
mostra timestamp 110 contado na última divisão com limite final publicado 109.
Esses cálculos do estudo não abriram bancos, fizeram HTTP ou executaram a suíte
do produto. As validações posteriores da 006–008 estão nos relatórios acima.
Não interpretar esses cálculos como correções ou testes integrados de summary.

As capturas React/Go realizadas na 009 estão no relatório de integração e no
[manifesto](screenshots.json); os detalhes e limitações do aceite constam nele.
A [página React de artista capturada no passo 2](../roadmap-02/screenshots/mixed-react-media.jpg)
é referência histórica da convivência, não prova de migração de artista.

Na implementação, criar `screenshots/` e `screenshots.json` com pares sobre cópias
equivalentes de SQLite v3 descartável. Registrar data, rota/query, perfil, modo,
tema, fuso, viewport e região/rolagem. Não incluir abas do usuário ou credenciais.

| Grupo | Capturas necessárias |
| --- | --- |
| Artista | Cabeçalho, top tracks/histórico, gráficos e álbuns, incluindo mais de 99 |
| Álbum | Cabeçalho normal/compilação, faixas, histórico e gráficos |
| Faixa | Múltiplos artistas, álbum, histórico e execução atual correspondente |
| Rewind | Mensal/anual, tempos ligados/desligados, janeiro e estado vazio |
| Falhas | Sem capa, título longo, item sem relação e bloco indisponível |
| Gestão | Entrada Go → React e retorno após ação em banco de teste |

Cobrir desktop/celular e Yuu/tema claro, rolagem abaixo da dobra e foco/teclado.
As issues [006](<../../tasks/issue-006 - 006-—-Migrar-páginas-públicas-de-artistas-álbuns-e-faixas-para-Go.md>),
[007](<../../tasks/issue-007 - 007-—-Migrar-atividade-e-interesse-das-páginas-de-mídia-para-HTML-SVG.md>),
[008](<../../tasks/issue-008 - 008-—-Migrar-retrospectiva-mensal-anual-e-validar-seus-cálculos.md>) e
[009](<../../tasks/issue-009 - 009-—-Integrar-mídia-rewind-Go-preservar-gestão-React-e-registrar-aceite.md>) organizam a execução.
