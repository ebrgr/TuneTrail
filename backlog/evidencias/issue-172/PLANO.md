# Plano ISSUE-172 — fase 1: ensaio com cópia do banco do usuário (legado AINDA LIGADO)

Decisões do usuário (21 e 22/11/2026): cortar o legado sem as 3 semanas de sombra; antes de desligar o modo legado, rodar todos os testes com o interruptor ligado numa **cópia do banco do usuário**, acrescentar muitas escutas de Inbox e Outbox e subir uma versão para o usuário conferir. A remoção do escritor legado (fase 2) só começa depois que o usuário conferir e disser que pode seguir.

## Fase 1 (esta)
1. Cópia segura do banco de desenvolvimento do usuário: `.local/koito/tunetrail.db` (em uso pela instância em 127.0.0.1:4110, schema v18; 80 escutas, 2 API keys, 1 relay configurado). Copiar com `sqlite3 ... ".backup"` (nunca copiar o arquivo ao vivo), para um diretório fora do repositório (`~/Documents/TuneTrail-ensaio-172/`), nunca versionado e nunca enviado a serviço externo. **Não tocar no banco original nem parar a instância da porta 4110.**
2. **Neutralizar toda saída de rede na cópia**: relay (destino, allowlist, tokens), Subsonic/Navidrome ou qualquer integração; apontar o relay para um servidor falso local (um mock do ListenBrainz em 127.0.0.1) e trocar/anular qualquer token na cópia. Nenhum dado do usuário pode sair da máquina.
3. Ligar o interruptor mestre e a ingestão ativa por ocorrência para os perfis da cópia (descubra as variáveis de ambiente e flags: `profile_ingest_flags`, `KOITO_*`); migrar a cópia até o schema atual (v20) com o backup automático.
4. **Rodar todos os testes com o interruptor ligado**: a suíte Go completa (`-race` onde couber), `make check` e os E2E, repetidos com a ingestão ativa por padrão (por variável de ambiente ou fixture que ligue as flags). Reportar cada falha com a causa; não corrigir código de produto sem registrar.
5. **Carga sintética de Inbox e Outbox sobre a cópia**: milhares de escutas pela API ListenBrainz com 3 ou mais API keys distintas (labels diferentes) e vários clientes, uma importação por arquivo, escutas com metadados completos, incompletos e faltantes, reenvios e duplicatas (mesmo segundo, ID externo confiável, linhas repetidas de arquivo), pendências de revisão com o Revisar ligado e desligado, falhas permanentes (resolvedor forçado a falhar), e o relay entregando ao mock (com falhas e pausa simuladas). Medir os limiares aprovados: p95 de ACK < 200 ms, visibilidade < 10 s, < 0,1% de 503, até 15 escutas/s no perfil piloto.
6. Conferir contagens e integridade: nenhuma escuta aceita perdida; contagens por perfil e digests antes e depois; o legado e o novo produzem as mesmas contagens para o mesmo tráfego.
7. **Subir uma versão para o usuário conferir** (porta 4120, `127.0.0.1`, mesma cópia já carregada, sem rede externa, com o mock do relay rodando) e deixá-la no ar; informar login, URL e o roteiro do que olhar (central de saúde Inbox/Outbox, Pendências, Falhas permanentes, escutas, rankings, exportação). Não derrubar a instância 4110 do usuário.
8. Relatório em `backlog/evidencias/issue-172/ENSAIO.md` (sem dados pessoais: sem tokens, sem chaves, sem nomes reais de músicas se puder agregar).

## Restrições
- Fase 1 NÃO remove o escritor legado, o interruptor, as flags nem o índice legado; nenhuma mudança de produção, exceto correções triviais que o ensaio exigir e estiverem registradas (prefira reportar).
- Commits só de scripts e evidências do ensaio (sem dados). Sem push, sem merge, sem publish, sem Done.
- Advisor (effort high) é obrigatório antes do aceite da fase 2.

## Pendências de decisão do usuário (levar ao relatório)
Padrão da flag Revisar; manter o reagendamento do timeout do MusicBrainz ou voltar a degradar.
