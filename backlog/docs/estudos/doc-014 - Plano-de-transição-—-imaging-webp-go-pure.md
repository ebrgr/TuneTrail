---
id: doc-014
title: Plano de transição — imaging + webp-go-pure
type: other
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-12T23:39:00Z'
tags:
  - estudo
---
> Importado de [docs-dev/PLANO_TRANSICAO_IMAGENS_GO.md](../../../docs-dev/PLANO_TRANSICAO_IMAGENS_GO.md).

# Plano de transição — imaging + webp-go-pure

Preparado em 06/11/2026, por solicitação do usuário. Este plano registra a escolha
de direção tecnológica e as issues locais. As duas fases foram concluídas; a
funcionalidade de geração de imagens sociais permanece para planejamento próprio.
Base: [investigação do codec](../../../docs-dev/ESTUDO_WEBP_GO_PURE.md) e
[estudo sem CGO](../../../docs-dev/ESTUDO_01_GO_SEM_CGO.md). Preservar SQLite consolidado v3.

## Objetivo e divisão

**Fase 1:** substituir bimg/libvips nas capas, avatares e cache por um pipeline
Go com `imaging` nas transformações e `webp-go-pure` na saída/leitura WebP.
JPEG/PNG continuam sendo decodificados por código Go. O binário normal deverá
compilar e executar sem CGO ou bibliotecas nativas de imagens.

**Fase 2:** revisar `internal/summary/image.go`, usando `imaging` para as
transformações e composições adequadas. O gerador principal está comentado;
essa revisão prepara código reutilizável e testado, sem ativar geração social
ou acrescentar rotas. A funcionalidade de imagens para redes tem escopo próprio
no roadmap e será planejada posteriormente.

`imaging` não substitui renderização de texto: manter `font`, `font/opentype`
e `math/fixed` onde necessários. Além disso, a biblioteca declara dependência
de `x/image` e importa BMP/TIFF. Portanto, retirar libvips/CGO não implica
retirar o módulo `golang.org/x/image`. Fontes:
[go.mod da imaging](https://github.com/disintegration/imaging/blob/master/go.mod),
[I/O e orientação](https://github.com/disintegration/imaging/blob/master/io.go),
[operações de imagem](https://github.com/disintegration/imaging/blob/master/README.md).
Fixar versões e registrar licenças durante a prova; não usar atualização flutuante.

## Sequência executável

| Ordem | Issue | Entrega |
| --- | --- | --- |
| 1 | [031 — contrato, corpus e parâmetros](../../../docs-dev/issues/031-contrato-corpus-imagens-go-solved.md) | Contrato observado da bimg, política de entrada/recursos, corpus e medições reproduzíveis da combinação escolhida. |
| 2 | [032 — pipeline Go](../../../docs-dev/issues/032-pipeline-imaging-webp-go-pure-solved.md) | Componente compartilhado com inspeção, limites, orientação, corte/escala e encode, sem registro WebP duplicado. |
| 3 | [033 — cache e administração](../../../docs-dev/issues/033-integracao-cache-capas-avatares-go-solved.md) | Download, capas e avatares usando o componente e preservando contratos de persistência e HTTP. |
| 4 | [034 — distribuição sem CGO](../../../docs-dev/issues/034-build-distribuicao-sem-cgo-solved.md) | Retirada da bimg/libvips, build/debug/CI/Docker/documentação e evidências finais da fase 1. |
| 5 | [035 — preparação da retrospectiva](../../../docs-dev/issues/035-refatoracao-summary-imaging-solved.md) | Refatoração das transformações/composição antigas, fontes preservadas e testes locais, após a conclusão da fase 1. |

Dependências: **031 → 032 → 033 → 034 → 035**. Cada issue exige registro dos
resultados e limitações em `docs-dev/comparisons/transicao-imagens/`, criado
quando houver execução. Os benchmarks anteriores avaliaram um protótipo com
`x/image/draw`, não `imaging`; não reaproveitar seus números como aprovação da
nova combinação.

## Progresso

031 concluída e aprovada pelo Advisor como estudo em 06/11/2026.
[Contrato, medições e limitações](../../../docs-dev/comparisons/transicao-imagens/ISSUE_031.md):
parâmetros iniciais Lanczos/EffortFastest, Q100/Q85 lossy; um processamento global
e quatro esperas canceláveis propostos para a 032. A política de JPEG/PNG/WebP
estático restringe formatos adicionais e exige documentação na implementação.
Os critérios de qualidade calibrados após a prova foram revalidados na 032 para
o corpus documentado, sem aprovação geral de fotografias ou perfis de cor.
032 concluiu o componente Go isolado, com validação no Go 1.25 e análise de
segurança. [Resultados e limitações](../../../docs-dev/comparisons/transicao-imagens/ISSUE_032.md).
033 integrou cache, capas e avatares ao pipeline Go, com testes e
[capturas](../../../docs-dev/comparisons/transicao-imagens/ISSUE_033.md). A 034 retirou bimg do
go.mod e atualizou o build. A [validação da distribuição](../../../docs-dev/comparisons/transicao-imagens/ISSUE_034.md)
concluiu a fase 1. A [035](../../../docs-dev/comparisons/transicao-imagens/ISSUE_035.md) preparou
os auxiliares da retrospectiva. A [037](../../../docs-dev/comparisons/transicao-imagens/ISSUE_037.md)
corrigiu a corrida de coleta e publicação automática.
[Revisão Advisor das 032/033](../../../docs-dev/comparisons/transicao-imagens/ADVISOR_032_033.md):
reprovado com ressalvas de admissão e cancelamento compartilhado, sem bloqueador
comprovado para iniciar a 034.

## Contrato a preservar e decisões da prova

- Manter UUIDs, URLs `/image/<id>/<tamanho>.webp`, MIME WebP e leitura dos arquivos
  atuais. Não alterar banco, limpar cache ou recodificar o acervo automaticamente.
  Arquivos antigos e novos coexistirão durante a transição.
- Medir o comportamento atual de corte central, imagens pequenas/retangulares,
  não ampliação, origem até 1200 e derivados 64/128/300/640/1000. Não usar `Fill`
  com tamanho fixo se isso ampliar imagens pequenas que a bimg preservava.
- Qualidade 100 para origem e 85 para derivados são pontos iniciais, com
  `Lossless` desligado; validar aparência, tempo e bytes para escolher o esforço.
  Definir na 031 critérios mensuráveis e justificativa das diferenças antes de
  integrar a mudança. Não prometer equivalência pixel a pixel ou ganhos.
- Suporte mínimo desejado: JPEG, PNG e WebP estático, por conteúdo. A 031 deve
  inventariar outros formatos hoje aceitos e registrar o destino de cada um.
  Importar a imaging registra formatos adicionais; a lista permitida do produto
  deve ser aplicada explicitamente antes do decode.
- Aplicar limites de bytes por operação, cabeçalho e dimensões antes de alocar
  pixels; validar novamente os limites após orientação/transformação. Preservar
  limites vigentes até decisão registrada. Recursos simultâneos precisam de
  controle global além de singleflight por ID/tamanho.
- WebP animado deve ser identificado antes de `std.Decode`, que materializa
  quadros no commit estudado. Escolher entre rejeição explícita e processamento
  limitado comprovado; a escolha não pode ficar implícita. Definir GIF também.
- Testar JPEG EXIF 1–8, alpha e imagens com bounds não iniciados em zero.
  A orientação automática da imaging examinada lê EXIF JPEG; não presumir
  orientação de metadados WebP/PNG nem conversão ICC/CMYK sem prova.
- Há um import `x/image/webp` no summary. Ele precisa ser retirado/substituído
  **na 032**, junto da introdução do novo codec, para impedir registro duplicado.
  Essa pequena compatibilização da fase 1 não antecipa a refatoração da 035.

## Validação e distribuição

A prova deve usar fixtures e bancos descartáveis, cache legado copiado, testes
de arquivos inválidos, geração concorrente, cancelamento e publicação atômica.
O codec não oferece interrupção garantida durante encode: não lançar goroutine
por requisição que apenas abandone trabalho ao cancelar. Registrar o alcance
real do cancelamento e da espera por capacidade.

Depois da integração, executar testes Go, web/admin e ingestão, gerar comparações
das imagens transformadas e amostrar páginas com capas/avatares em desktop e
celular. Registrar qualidade visual e recursos sem dados pessoais. Testar rollback
lendo saídas novas com a implementação anterior; fallback para o binário antigo
não deve depender de alterar o esquema nem de recodificar todo o cache.

O build normal usará `CGO_ENABLED=0`, no Go declarado pelo projeto e nas imagens
Linux amd64/arm64 publicadas atualmente; conferir também o ambiente local de
depuração. O job `-race` poderá manter CGO e compilador C conforme sua exigência,
sem libvips. Docker deve preservar TLS, fusos, permissões, sinais e healthcheck;
Node continua restrito aos testes JS da interface e documentação.

## Limites da fase 2

A `resize` antiga distorce para largura/altura exatas; `Fit` e `Fill` têm
semânticas diferentes. A composição antiga usa coordenadas do source em
`draw.Draw`, e uma conversão direta para `Overlay` altera o posicionamento.
Definir explicitamente retângulo, posição, proporção, alpha e clipping antes da
substituição, incluindo pontos negativos. Corrigir acesso aos pixels que ignora
`Bounds().Min` e fechar arquivos/font faces no código alterado.

Separar cálculo estatístico de renderização e usar leitores/fontes fornecidos
explicitamente ou assets embutidos, evitando caminhos dependentes do cwd.
Não alterar a retrospectiva HTML, autenticação, visibilidade ou regras numéricas.
Não remover `x/image` se permanecer necessário direta ou transitivamente.

## Revisão

Advisor, Sol esforço médio: a proposta inicial de troca direta foi reprovada
com ressalvas. As condições de registro único, recursos, formatos, cache,
qualidade, fontes e composição foram incorporadas às issues deste plano.
A revisão foi estática; não representa implementação ou teste da combinação.
Após revisar este plano e as issues 031–035 com as condições incorporadas, o
Advisor emitiu parecer **aprovado, sem objeção ao design proposto**. As issues
permanecem locais em Markdown, conforme a orientação do usuário para esta etapa.

## Correção após revisão

[036 concluída](../../../docs-dev/comparisons/transicao-imagens/ISSUE_036.md): admissão administrativa,
contextos independentes e respostas HTTP corrigidos e testados. A 034 concluiu a
distribuição; o risco histórico de publicação automática e coleta foi separado na
[037](../../../docs-dev/issues/037-publicacao-automatica-coleta-imagens-solved.md).

## Retrospectiva preparada

[035 concluída](../../../docs-dev/comparisons/transicao-imagens/ISSUE_035.md): escala com imaging,
composição com coordenadas explicitadas, fontes/leitores injetados e testes sem
CGO. A geração social continua desativada e requer planejamento próprio.

## Publicação automática protegida

[037 concluída](../../../docs-dev/comparisons/transicao-imagens/ISSUE_037.md): reservas por UUID
coordenam produtores e coletor dentro do processo, sem reter a trava durante
rede ou banco. O caso de múltiplos processos compartilhando cache requer
coordenação separada.
