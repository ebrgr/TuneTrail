# 044 — Versionamento próprio do TuneTrail

**Status:** Fechada e aprovada pelo usuário em 09/11/2026

## Objetivo

Adotar TuneTrail 0.3 e fazer `make version.bump` incrementar o número após `0.` (0.4, 0.5, …, 0.10), sem mudar automaticamente para 1.0. Preservar como históricas as versões e imagens anteriores de Koito/SUM.

## Aceite

- Ajustar versão incorporada, validação de sincronização, incremento e comparação com o commit anterior; aceitar a transição inicial a partir do formato SUM-RC.
- Atualizar defaults de build e implantação, CI, documentação e changelog, sem anunciar publicação de imagem inexistente.
- Verificar incremento, versões com mais de um dígito, transição legada, rejeição de regressão, integridade das referências a Koito 0.3.2 e histórico, e execução do aplicativo.
- Registrar resultado no roadmap; não publicar, criar tag ou fazer commit automaticamente.

## Implementação e verificação

Adotado `0.3` em `internal/version/VERSION`, com incrementos numéricos de `0.x`, sem avanço automático para 1.0. Docker/Compose, documentação e CI foram sincronizados; tags SUM antigas permanecem históricas. O Advisor apontou riscos de transição e substituição acidental de `0.3.2`, tratados no script.

`make version.check`, verificação do commit pai e testes Python em repositórios temporários passaram. `make check` executou `go vet` com sucesso e passou em vários pacotes Go, mas testes que abrem sockets foram bloqueados pelo ambiente (`bind: operation not permitted`), impedindo o término da suíte completa. Não houve publicação.

## Encerramento — 09/11/2026

Fechada e aprovada por solicitação explícita do usuário. A checagem
`python3 scripts/version.py` confirmou TuneTrail 0.6 sincronizado e os quatro
testes de `test_version.py` passaram na revisão de 09/11/2026. As limitações das
execuções anteriores permanecem históricas; não foi realizada nova publicação,
criação de tag ou incremento de versão neste encerramento.
