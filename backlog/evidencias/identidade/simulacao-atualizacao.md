# Simulação da atualização 0.9 → ISSUE-111 com Docker

**Data:** 14/11/2026 · **Imagens:** `ghcr.io/ebrgr/tunetrail:0.9` (publicada) e build local da branch da ISSUE-111 (`b1a5d38`) · **Script:** `upgrade_sim.py` (scratchpad da sessão) · Projetos `sim-*`, portas 14130+, dados fictícios; o contêiner `koito-oficial` da máquina não foi tocado.

| Passo | Cenário | Resultado |
| --- | --- | --- |
| 1 | 0.9 com o `deploy/compose.yaml` e o `deploy/.env` atuais (`KOITO_*`, `/etc/koito`, projeto `koito-sum`) | Banco criado no diretório do host; 3 escutas gravadas pela API; cookie `koito_session` |
| 2 | **Só a imagem nova**, mesmo compose, `.env` e diretórios | Inicia sem erros; 3 escutas preservadas; 6 avisos de variáveis `KOITO_*` no log; diretório explícito `/etc/koito` usado (sem aviso de diretório legado); cookie antigo não aceito, novo login grava `tunetrail_session` |
| 3a | Compose novo (`tunetrail`) com o antigo ainda rodando, outra porta | **No Docker Desktop do macOS com diretório do host, não foi recusado**: travas de arquivo não atravessam o compartilhamento de arquivos do Docker Desktop. Teste à parte: com volume Docker (sistema de arquivos Linux) o segundo contêiner é recusado com "another TuneTrail server is already using this data directory". Ressalva registrada no `DEPLOYMENT.md` |
| 3b | Compose novo depois de parar o antigo, mesmo `.env` | Inicia; 3 escutas; volume em `/etc/tunetrail`; nenhum aviso `KOITO_*` (o compose novo passa `TUNETRAIL_*`) |
| 4 | `koito.db` legado (v2) montado em `/etc/koito` com a imagem nova | Migrado para `tunetrail.db` (esquema 3); backup `koito.db.pre-tunetrail-*.bak` criado; **`koito.db` com o mesmo SHA-256 de antes** |

Teste de contêiner do CI (`test.yml`) executado localmente com três variantes, todas aprovadas: nomes novos (banco em `/etc/tunetrail`, sem avisos); nomes antigos com `KOITO_CONFIG_DIR=/data` (banco em `/data`, aviso); volume em `/etc/koito` sem variável de diretório (banco em `/etc/koito`, aviso de diretório legado).
