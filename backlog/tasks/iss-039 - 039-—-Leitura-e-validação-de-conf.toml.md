---
id: ISS-039
title: 039 — Leitura e validação de conf.toml
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - config
milestone: m-4
dependencies:
  - ISS-038
references:
  - docs-dev/issues/039-leitura-configuracao-toml-solved.md
type: feature
ordinal: 39000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/039-leitura-configuracao-toml-solved.md](../../docs-dev/issues/039-leitura-configuracao-toml-solved.md) — numeração local 039.

**Status:** Fechada e aprovada pelo usuário em 09/11/2026 · **Depende de:** contrato da 038

## Objetivo

Implementar o [contrato da 038](../../docs-dev/CONTRATO_CONFIGURACAO_038.md) para os dez campos
globais prioritários, mantendo execução sem arquivo e SQLite consolidado v3.

## Aceite

- Descobrir o diretório externo antes de ler `conf.toml`, ao lado de `tunetrail.db`;
  usar a mesma descoberta/precedência na execução e em `reset-password`.
- Aplicar ambiente não vazio > `_FILE` selecionado > TOML presente > padrão,
  distinguindo ausência, false e vazio. Arquivo `_FILE` vazio não cai no TOML;
  erro de leitura falha. Não criar TOML nem copiar valores automaticamente.
- Usar schema 1 com as dez chaves do contrato; rejeitar tipo, sintaxe, versão ou
  chave desconhecida. Preservar parser de ambiente legado quando não houver TOML;
  documentar validação estrita ao ativar arquivo e origem de cada valor.
- Validar configuração do arquivo com defaults e composição efetiva com overrides,
  inclusive URL/rate MusicBrainz, par URL/params Subsonic e fuso IANA/automático.
- Preparar helper limitado compartilhado do ping Subsonic: contexto, timeout 10 s,
  teto de resposta 8 MiB, sem redirects e validação HTTP/JSON/status. Startup deve
  orientar correção offline em erro; salvar arquivo não executa teste remoto.
- Escolher parser/estratégia compatível com preservação textual da 040; não aceitar
  perda silenciosa de comentários. Preservar erro de inicialização em chamadas
  repetidas de `cfg.Load`, sem sucesso falso causado por `sync.Once`.
- Manter relay no banco por perfil e as outras opções externas; sem schema novo,
  mudança de banco ou hot reload. Valores desta entrega são dados comuns.
- Testar matriz de fontes, defaults e falhas, restart/reset-password e compatibilidade
  com Compose/VS Code. Usar fixtures e servidores locais, sem banco operacional.

## Implementação — 09/11/2026

Leitura TOML schema 1 com BurntSushi/toml, documento original preservado para
futura edição, origem por campo e precedência de ambiente/`_FILE`. O mesmo leitor
é usado pelo servidor e por `reset-password`. Defaults legados, relay por perfil,
SQLite v3 e demais variáveis permanecem compatíveis; TOML ativo recebe validação
estrita de ambas as composições. A inicialização usa verificação Subsonic limitada
reutilizável e `cfg.Load` conserva erros para chamadas posteriores.

Testes Go cobrem fontes e campos explícitos, schema/tipos/chaves desconhecidas,
conflitos, legado sem TOML, diretório `_FILE` estável, falhas repetidas, sondagem
Subsonic e processo real com banco temporário e reset. Advisor/Sol revisou o
patch e aprovou após correções. A gravação e a tela ficam nas issues 040 e 041.
A 038 possui contrato revisado suficiente para esta implementação; permanece
aberta para aceite conforme seu próprio registro.

## Validação final

- `CGO_ENABLED=0 go test -count=1 ./...` e `go vet ./...`: passaram.
- `go test -race -count=1 ./internal/cfg ./internal/images`: passou com CGO
  habilitado apenas para o detector de corridas.
- `scripts/validate_multiuser.py` com binário temporário: jornadas em bancos
  novos e migrados v2 passaram; integridade SQLite confirmada.
- Processo real com `conf.toml` e `KOITO_CONFIG_DIR_FILE`: startup, banco v3 e
  `reset-password` no mesmo diretório. Schema inválido não criou banco. O teste
  integrado também está em `cmd/api/config_integration_test.go`.
- `go mod tidy -diff`: sem divergências; Graphify externo atualizado.

O leitor aceita configuração em arquivo somente leitura, mas o servidor ainda
precisa de diretório de dados gravável para SQLite; a 040 tratará erros de escrita
na futura edição do TOML. Não foi implementada edição administrativa nesta issue.

## Encerramento — 09/11/2026

Fechada e aprovada por solicitação explícita do usuário. As evidências acima
permanecem registradas; a persistência segue na issue 040 e a interface na 041.
<!-- SECTION:DESCRIPTION:END -->
