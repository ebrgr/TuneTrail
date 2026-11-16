# ISSUE-070 — fontes locais

Avaliação e execução próprias em 16/11/2026. Plano ai-memory: 01a1e5bc-e22f-7cb0-8fae-4b1d44e0b6b4.

## Inventário e distribuição

`fontes.json` registra origem, SHA-256, bytes, eixos, pesos e cobertura por arquivo. WOFF2 publicados: **983.452 bytes**. Jost vem da distribuição Google Fonts v20 (arquivos publicados originais, sem conversão); League Spartan do repositório dos autores; Inter do repositório rsms/inter. Nenhum arquivo foi convertido ou subconjuntado localmente. Jost inclui os subconjuntos originais latin, latin-ext e cyrillic em normal e italic (0/1/2 = italic cyrillic/latin-ext/latin; 3/4/5 = normal cyrillic/latin-ext/latin). Acentos portugueses precompostos ficam no latin; marcas combinantes adicionais ficam cobertas pelos subconjuntos originais. League Spartan e Inter preservam a cobertura completa do WOFF2 publicado.

OFL 1.1 verificada separadamente para Jost, League Spartan e Inter: os preâmbulos não declaram Reserved Font Names. A definição genérica de RFN no corpo da licença não constitui uma reserva. As fontes permanecem sem modificação. Textos em assets/fonts/*-OFL.txt, fora do manifesto e das rotas públicas. Dockerfile copia assets inteiro à imagem final; package_release.py inclui assets inteiro no pacote. Ambos preservam as licenças. TTF históricos mantidos, 119.064 bytes: busca por nomes/extensão não encontrou consumidor de runtime; internal/summary/image.go é protótipo desativado, recebe bytes do chamador. Nenhum TTF foi removido.

## Uso e integração

CSS: Jost é corpo/controles por herança, Inter fallback; pesos 400, 500, 600, 700 (inclui b/strong). League Spartan é headings/header-font, pesos 400, 500 (herança em mídia/admin), 600 e 700 (Home). Não há font-style italic explícito nem conteúdo em em no HTML atual; i existentes são marcadores decorativos vazios. Jost/Inter têm faces italic reais para preservar o contrato anterior. League Spartan publicado é estático 400/500/600/700, declarado com pesos individuais. Jost e Inter são variáveis, eixos verificados em fontes.json.

Google Fonts estava somente em assets/templates/shared/head.html: dois preconnects e um stylesheet, compartilhados por live/preview/admin. Não havia @import local. Todos removidos. fonts.css entra no bundle ui.css, usado nos três heads. URLs relativas fonts/* recebem SHA-256 pelo rewriter existente; alteração dos bytes invalida fonte e CSS. Nenhuma CSP encontrada no código runtime; não foi adicionada política nova. MIME font/woff2 e nosniff existentes foram verificados em HTTP (`http.json`); licenças retornam 404.

## Retrospectiva

Referência: https://github.com/gabehf/Koito/blob/v0.3.2/client/app/components/rewind/RewindTopItem.tsx (cópia textual local). O limite de título longo é **mais de 28 caracteres**, aplicado a qualquer vencedor: text-xl/lg:text-2xl versus text-2xl/lg:text-4xl. Esta entrega apenas documenta a regra, sem alterar tamanhos.

Mudança visual separada da origem: vencedor League Spartan 600 sem uppercase; rótulos League Spartan 400 uppercase; ano Jost 20/500. Estilo normal, como referência. Só seletores de rewind mudaram; h3 global segue 400 uppercase. Nenhum tamanho/line-height foi alterado.

| Largura | Título atual | Vencedor atual | Rótulo atual | Ano |
| --- | --- | --- | --- | --- |
| 390 | 28 px | 22 px | 18 px | 20 px |
| 768 | 30,72 px | 23,04 px | 18 px | 20 px |
| 1440 | 42 px | 36 px | 18 px | 20 px |

Medidas completas, caixas, line-height, pesos e fontes efetivas antes/depois: `medidas-fontes.json`. A referência histórica Koito aponta título 36 px e rótulo 16 px; mudanças de tamanho dependem do usuário, AC4 continua desmarcado. As medições de antes bloqueiam fontes externas e registram o fallback real; não presumem equivalência de glifos com a captura antes, que permitiu Google Fonts.

## Verificação

Três PNGs somente, Yuu, dados fictícios da fixture oficial: retrospectiva antes/depois e Home depois, 1440×900 CSS px, escala 1, navegador Orca. Bloqueio de fonts.googleapis.com e fonts.gstatic.com ativo nas capturas depois. Não houve acesso a dados operacionais. `reference/` é exportação determinística atual de make ui.reference, usada pelo ui.reference.check; baseline 065 preservada.

Orca não expõe CSS.getPlatformFontsForNode (comando cdp indisponível); por isso, as medidas adicionais sem imagens usam Chrome headless/Playwright e CDP, seguindo o método da 065. Rede externa inteira bloqueada nessas verificações. Público, prévia e admin testados; nenhuma tentativa externa de fonte após a mudança. Probes de português confirmam fontes customizadas reais em Jost/Inter normal e italic e League Spartan normal, sem fallback. Fontes dos seletores da retrospectiva também registradas por CDP. A prova não promete funcionamento offline de capas/integrações.

Pacote local inspecionado com tar: contém as três licenças OFL. `uso-css.json` inventaria family/weight/style efetivamente computados nas onze páginas fictícias; inclui League Spartan 500 em mídia/admin, atendido pela face Medium upstream.

make check passou (inclui ui.check e 12/12 jornadas e2e); ui.check executado também separadamente. Teste de geração verifica dependência/hash/invalidação e rejeição de fonte ausente. Sem push, merge, publish ou Done.
