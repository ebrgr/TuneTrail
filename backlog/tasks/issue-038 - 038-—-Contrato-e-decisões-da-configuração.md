---
id: ISSUE-038
title: 038 — Contrato e decisões da configuração
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-14T00:20:00Z'
labels:
  - config
milestone: m-4
dependencies: []
priority: high
type: feature
ordinal: 104000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/038-contrato-configuracao.md` — numeração local 038. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Em andamento — planejamento registrado em 09/11/2026 · **Bloqueia:** 039–042

## Objetivo

Definir o contrato da configuração administrativa priorizada pelo usuário antes
de implementar TOML, persistência e interface. O contrato orientou as issues 039–042, agora implementadas; a 042 aguarda aceite.

## Plano da primeira entrega

[Contrato, inventário dirigido, decisões e matriz de testes](<../docs/design/doc-019 - Issue-038-—-contrato-e-plano-da-configuração-administrativa.md>).

- Dez campos globais: log level; URL, limite e desativação de MusicBrainz;
  desativação de Deezer/CAA; chave Last.fm; URL/params Subsonic; fuso forçado.
- Todos os valores citados serão tratados como dados comuns, visíveis/editáveis
  na administração autenticada, conforme solicitação do usuário. Sem máscaras
  ou tratamento especial de segredo. A senha de login não integra esses campos.
- Usar `KOITO_FORCE_TZ`/`time.force_tz` para o fuso da aplicação; `TZ` continua
  externo e determina o fuso local do processo. Modo automático preserva a
  precedência de query, cookie e processo.
- Relay já está no SQLite v3 em `profile_relays`, por perfil de escuta. Manter
  URL/token e gestão existentes, sem cópia no TOML nem nova coluna enabled.
- Subsonic é conexão para fetch de artistas, álbuns e imagens, não sincronização
  de escutas. Validar URL/params em conjunto e prever teste explícito do rascunho.

## Trabalho e aceite documental

- [x] Conferir leitura de ambiente/`_FILE`, defaults e consumidores dos campos.
- [x] Registrar schema TOML inicial e precedência, ausência/vazio/false e overrides.
- [x] Separar configuração global, fuso do processo e relay persistido por perfil.
- [x] Definir sessão admin existente, valores visíveis e reinício externo.
- [x] Delimitar bootstrap, conta master, diretório/rede e outras opções externas.
- [x] Planejar validações, escrita, comentários, concorrência e recuperação offline.
- [x] Revisar com Advisor e incorporar ressalvas no contrato e issues seguintes.
- [ ] Concluir a 038 após aceite do planejamento, preservando as limitações registradas.

## Verificação

Testes dirigidos existentes de configuração, fuso e relay passaram com Go sem CGO;
comandos e alcance estão no contrato. Advisor/Sol revisou o plano antes das edições.
Não foi criado leitor TOML, tela, endpoint ou migração de banco nesta etapa.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:20:00Z

---

Aceite: usuário pediu o fechamento da ISSUE-038 em 13/11/2026, a partir de Em andamento (etapa Aguardando aceite opcional). Encerrada com as verificações já registradas; sem nova execução de testes nesta etapa além do make check da publicação 0.8.

---
<!-- COMMENTS:END -->
