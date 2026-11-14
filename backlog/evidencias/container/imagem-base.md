# Imagem base do contêiner — medição e opções

**Data:** 14/11/2026 · **Imagem:** build local da 0.10 (`tunetrail:ci`) · **Tarefa:** ISSUE-102

| Item | Tamanho |
| --- | ---: |
| Imagem completa (`debian:bookworm-slim` + curl, ca-certificates, tzdata) | **188 MB** |
| Binário `/app/tunetrail` (Go puro, CGO desligado) | 17 MB |
| `/usr/share/zoneinfo` | 3,9 MB |
| `/etc/ssl/certs` | 0,6 MB |

| Opção | Estimativa | A favor | Contra |
| --- | ---: | --- | --- |
| Debian slim (atual) | 188 MB | funciona; shell e ferramentas | maior superfície e mais atualizações de pacotes |
| **Alpine (escolhida)** | ~30 MB | pequena; mantém shell para suporte e Unraid; `ca-certificates` e `tzdata` disponíveis | healthcheck e smoke do CI precisam de ajuste |
| Distroless static | ~22 MB | mínima, sem shell, usuário não root pronto | diagnóstico dentro do contêiner mais difícil |

O binário não usa a libc (CGO desligado), então musl e o resolvedor de DNS do Alpine não mudam o comportamento.
