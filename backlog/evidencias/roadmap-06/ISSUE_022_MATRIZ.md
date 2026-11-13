# Issue 022 — matriz e fixture da comparação

[Issue](<../../tasks/issue-022 - 022-—-Preparar-matriz-fixture-e-protocolo-de-comparação.md>) ·
[Plano](<../../docs/roadmaps/doc-009 - Roadmap-6-—-comparação-visual-e-decisão-sobre-React.md>) ·
[Procedimento](ISSUE_022_PROTOCOLO.md) · [Modelo de manifesto](manifesto-modelo.json).
A matriz é um **plano**, não uma lista de casos aprovados. Todas as jornadas de
navegador abaixo estão **não executadas** e pertencem às issues indicadas.

## Dimensões e cobertura mínima

Modos G = Go; R = React hidratado. V = visitante; A = administrador fictício.
Viewports: D 1440×1000; M 390×844; limites 320/360 e 639/640/641,
767/768, 959/960, 1023/1024 em telas de risco. Temas Y = Yuu, P = Pearl,
B = Black; amostrar seleção e legibilidade dos demais nove. Usar escala 1,
idioma e zoom documentados, fuso `America/Sao_Paulo` salvo indicação.
Todos os pares comparáveis são G/R com JS habilitado; os casos marcados `G sem JS`
também requerem navegador com JS desativado. A referência instantânea da fixture
não é injetada no servidor. Esperar conteúdo real, imagens e fontes verificadas;
sem evidência de hidratação, o caso React não está aprovado.

| ID / issue | Rota/estado/ação | Perfil, ator e base | Cobertura mínima | Evidência e resultado esperado |
| --- | --- | --- | --- | --- |
| P01 / 023 | `/`, seleção, volta e perfil inexistente | 0/1/N, V, `zero`/`one`/base | G/R D/M Y/P/B; G sem JS | Links e destino corretos; capturas de seletor e estados vazio/erro |
| P02 / 023 | `/alpha`, troca para `/beta`, recarga, duas abas e voltar/avançar | Alpha/Beta/Empty, V, base | G/R D/M Y/P; G sem JS | Totais all-time 7/2/0, sem trocar destino; capturas e jornada |
| P03 / 023 | `/alpha/chart/top/{artists,albums,tracks}`, filtro e empate | Alpha/Beta, V, base | G/R D/M Y/P; G sem JS | Valores por perfil; controles, ordenação, overflow e recortes |
| P04 / 023 | `/alpha/listens?limit=100&page=2`, filtros e retorno | Alpha, V, `large` | G/R D/M Y; G sem JS | 1009 escutas, página 2 não vazia; empates, tempo e navegação |
| P05 / 023 | Busca pessoal e seleção via sidebar | Alpha/Beta, V, base | G/R D/M Y/P; G sem JS | Item exclusivo Beta não aparece em Alpha; avaliar dialogs/fallback |
| P06 / 023 | `/alpha/{artist,album,track}/1`, relações e histórico | Alpha/Beta, V, base | G/R D/M Y/P | Escopo da entidade, valores e longos Unicode; capa compartilhada |
| P07 / 023 | atividade e interesse por entidade | Alpha/Beta, V, base | G/R D/M Y; UTC e São Paulo | Valores em datas/intervalos comparados a oráculo independente; tabela sem JS |
| P08 / 023 | `/alpha/rewind?year=2018&month=11`, anual e vazio | Alpha/Empty, V, base | G/R D/M Y/P; G sem JS | Valores por período e fuso, opção de tempo, estados vazios |
| P09 / 023 | Imagem ausente, conteúdo longo e layout | Alpha/Beta, V, base | G/R M 320/360/390 + limites, Y/B | Sem conteúdo inacessível; imagem compartilhada disponível e outra ausente |
| A01 / 024 | `/admin`, login inválido/válido e retorno | A/V, `mutation` por modo | G/R D/M Y/P; G sem JS | Autorização e erro legível; sessão e retorno corretos |
| A02 / 024 | Perfis, avatar, conta, chaves/relay | A, `mutation` | G/R D/M Y/P; G sem JS essencial | Destino correto; capturas com campos secretos vazios/ocultos |
| A03 / 024 | Importação 5 formatos/exportação, falha parcial | A, nova cópia por ação | G/R D/M Y; G sem JS essencial | Destino/contagens; cinco formatos não implicam rollback total |
| A04 / 024 | `/{slug}/manage/{kind}/1`, aliases, artistas, capa, escuta | A, nova cópia por ação | G/R D/M Y/P; G sem JS essencial | Dados, retorno e limites de perfil; gesto de upload quando aplicável |
| A05 / 024 | Revisar/cancelar/confirmar delete/merge e plano obsoleto | A, nova cópia por ação | G/R D/M Y; G sem JS | Efeito global, deduplicação, sem repetição; banco íntegro |
| X01 / 024 | Teclado, foco/Escape, rótulos, contraste, zoom | V/A, base e `mutation` | G/R D/M Y/P/B; G sem JS | Percurso real; registrar falhas e exceções, não apenas screenshots |
| C01 / 025 | Rotas legadas, `/theme-helper`, favicon, assets e API | V/A, base | G/R GET/HEAD e ações previstas | Contrato observado/esperado, sem assumir fallback automático |
| C02 / 025 | Jornada web/admin após migração v2→v3 | V/A, cópias de fixture migrada | G/R e integridade | Migração real e backup, contagens pré/pós; não executado pela 022 |

## Fixture v1: fatos verificados

`scripts/roadmap06_fixture.py` estende a fixture 016 e cria somente um diretório
novo. Base: Alpha=7, Beta=2, Empty=0; variações zero, um e grande 1009/2/0.
Há duas pessoas na mesma faixa/instante, uma faixa sem escuta, Unicode/nome longo,
artistas e imagem compartilhados, imagem Beta propositalmente ausente, e no
cenário grande empate na fronteira do cursor de exportação 1000/1001. A imagem
compartilhada local existe nas cópias `go`, `react` e `mutation`; outras variantes
não incluem o cache de imagem e isso deve ser registrado em capturas.

Eventos DST Alpha/track 1, oráculos fixos em São Paulo:

| UTC | Data/hora local e offset |
| --- | --- |
| 2018-11-04 02:59 | 2018-11-03 23:59 -03:00 |
| 2018-11-04 03:01 | 2018-11-04 01:01 -02:00 |
| 2019-02-17 01:30 | 2019-02-16 23:30 -02:00 |
| 2019-02-17 02:30 | 2019-02-16 23:30 -03:00 |

A fixture não inclui uma base v2 migrada: a 025 deverá gerá-la e migrá-la por
inicialização real. A comparação por intervalo exige oráculo específico ainda a
construir na 023. O relatório gerado registra versão e hash; dados fictícios não
são exemplos para implantação. Os dados de sessões/now-playing não são persistidos
como estado inicial; cada jornada prepara seus próprios estados.
