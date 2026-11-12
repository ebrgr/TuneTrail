# Contrato proposto para a 032 — definido antes da medição

Versão 1, 06/11/2026. Escopo desta issue: definir e medir; estes limites não estão
aplicados à API. As divergências abaixo devem ser resolvidas explicitamente na 032.

- JPEG, PNG e WebP estático por conteúdo. A proposta recusa GIF, BMP, TIFF, SVG,
  HEIF/AVIF e WebP animado com erro de formato legível. A 031 inventaria os
  formatos aceitos na instalação atual; recusar algum deles é alteração de
  produto e deve constar na documentação/changelog da implementação.
- Manter até 10 MiB nas operações que já têm esse limite; avatar conserva limite
  próprio HTTP de 5 MiB (inclui multipart). SaveImage recebe proteção interna
  explícita de 10 MiB como teto geral, sem ampliar os limites dos chamadores.
- Dimensões positivas, cada lado até 10.000 e produto int64 até 40.000.000;
  cabeçalhos inspecionados antes da alocação, validação repetida após decode.
  Preservar os limites atuais de download/capas; não alegar que SaveImage já os
  aplica universalmente. Provas de cabeçalho não demonstram decodificação válida.
- Preservar o comportamento observado bimg de dimensões, incluindo o ramo que
  mantém o retângulo original quando AMBOS os lados são menores que o pedido.
  Inventariar o caso em que somente um lado é menor: Enlarge=false não é garantia
  universal de ausência de ampliação. Origem mantém qualidade100 lossy; derivados
  qualidade85. Testar esforço -1/2/4 do candidato antes de escolher.
- Preservar orientação JPEG EXIF1–8, alpha, MIME, UUIDs, caminhos e arquivos
  legados. Metadados não devem reaparecer na saída. ICC/CMYK requer classificação
  com referência: retirar perfil não é conversão automática para sRGB.

## Gates iniciais mensuráveis da prova (não orçamento universal)

Compatibilidade obrigatória: dimensões/alpha/orientação corretas em todos os
casos suportados; zero sucesso em entradas de formato/limites recusados;
leitura cruzada dos WebP por ambos codecs; bounds em memória sem regressão.

Referência geométrica sem perdas: PNG independente por ImageMagick auto-orient,
resize/crop central Lanczos; comparar RGB composto sobre branco e preto e alpha.
PSNR >= 30 dB por fundo para texturas e >= 28 dB para bordas/transparência é
um alerta inicial, não substituto da inspeção. Diferenças do próprio baseline
com o mesmo oráculo devem constar. Alpha da transformação pode ser interpolado;
encode deve preservá-lo exatamente em relação à imagem anterior à codificação.

Gate de investigação de custo: p95 de operação <= 3x baseline, bytes <= 1,5x
baseline por caso. Falha exige ajustar esforço/filtro ou registrar tradeoff antes
da integração; não descartar casos desfavoráveis da média.

Gates de recursos para a máquina desta prova: processo <= 768 MiB no caso de
40 MP e <= 512 MiB em carga de dois trabalhos com entrada 2400x1600. Timeout
120s por subprocesso. Medir carga 1/2/4 no mesmo processo, com tempos individuais,
vazão e pico RSS (inclui memória nativa; normalizar unidades por OS).

Proposta conservadora de início: uma transformação por vez globalmente;
limitar a espera e permitir cancelá-la. Só aumentar concorrência após medições.
O codec não tem cancelamento no meio do encode; não abandonar goroutines de
trabalho ao cancelar a requisição. Esses gates não aprovam hardware menor nem
qualidade fotográfica sem corpus fotográfico representativo.

## Parâmetros encaminhados após a medição

O gate absoluto inicial não foi cumprido nas texturas de estresse por nenhum dos
dois pipelines; mantê-lo registrado como alerta, sem tratar a execução do script
como aprovação de qualidade. Para a 032, o contrato por classe fica:

- Campo procedural de tons naturais: origem >=40 dB e derivado >=35 dB por fundo;
  perda relativa máxima de 1,5 dB na origem e 0,5 dB no derivado, comparada ao
  baseline com o mesmo oráculo. O limite maior na origem documenta a diferença
  observada de até 1,274 dB apesar de ambos resultados superarem 45 dB. Não é
  aprovação colorimétrica ou fotográfica geral.
- Texturas de estresse/alpha: registrar PSNR absoluto e exigir diferença >=
  baseline menos 0,5 dB, com dimensões e alpha corretos. CMYK/ICC têm diferenças
  classificadas e precisam de política explícita na implementação; nenhum
  gerenciador de perfis foi adicionado nesta prova.
- Parâmetros iniciais: imaging/Lanczos, codec EffortFastest (-1), origem Q100
  lossy e derivados Q85 lossy. Esforços2/4 aumentaram tempo e não foram uma
  melhora uniforme de qualidade; não escolhê-los pelo tamanho isoladamente.
- Começar com um processamento global e no máximo quatro esperas canceláveis,
  sem pixels já decodificados na espera; retornar erro de ocupação explícito
  quando exceder a capacidade. Os mapeamentos HTTP serão definidos na 033.

Esses critérios foram calibrados no corpus da 031 e precisam ser implementados
e revalidados na 032. Os alertas do gate inicial e as limitações da medição
continuam nos JSONs e no relatório, sem apagar resultados desfavoráveis.
