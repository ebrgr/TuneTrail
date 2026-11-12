# Corpus da issue 031

O corpus é procedural e reproduzível pelo gerador
[`corpus.py`](../../scripts/experiments/images031/corpus.py). Para evitar guardar
arquivos artificiais de 10 MiB e imagens de 40 MP no Git, ele é materializado
somente no --work-dir temporário do experimento. O manifesto com todos os hashes,
proveniência e tamanhos e as imagens menores de comparação ficam em
[comparisons/transicao-imagens](../../backlog/evidencias/transicao-imagens).

Inclui texturas de cores/gradientes e um campo procedural de tons naturais (não
é fotografia), marcadores de canto, alpha, EXIF1–8, ICC RGB criado pelo próprio
gerador, JPEG CMYK, WebP antigo gerado pela bimg, formatos adicionais locais,
animações, dados truncados e fronteiras de bytes/dimensões/pixels. Uma fixture
em memória nos testes Go cobre bounds com origem diferente de zero.
