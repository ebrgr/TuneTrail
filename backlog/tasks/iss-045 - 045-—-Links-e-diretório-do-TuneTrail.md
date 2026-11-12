---
id: ISS-045
title: 045 — Links e diretório do TuneTrail
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - docs
  - operacao
milestone: m-5
dependencies: []
references:
  - docs-dev/issues/045-links-diretorio-tunetrail-solved.md
type: feature
ordinal: 45000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de [docs-dev/issues/045-links-diretorio-tunetrail-solved.md](../../docs-dev/issues/045-links-diretorio-tunetrail-solved.md) — numeração local 045.

**Status:** Fechada e aprovada pelo usuário em 09/11/2026

## Objetivo

Alinhar links do repositório e referência futura da imagem ao nome TuneTrail, preservando créditos e histórico, e renomear a pasta local sem perder alterações.

## Aceite

- Atualizar links próprios e exemplos de imagem; manter referências ao Koito upstream e alertar que a imagem nova depende de publicação.
- Confirmar que o VS Code usa caminhos relativos ao workspace e preservar dados, histórico Git e alterações locais.
- Validar links locais, configuração e testes pertinentes; registrar eventuais limitações para mover o diretório ou publicar o site.

## Resultado

Links ativos do projeto, templates, metadados OCI e destino da imagem no Compose foram alinhados a `ebrgr/TuneTrail` e `ghcr.io/ebrgr/tunetrail`. O domínio do site de documentação (`koito.io`) permanece até ser definido e configurado um endereço do TuneTrail; referências históricas de imagens e ao upstream foram preservadas. O Advisor confirmou o risco de divergência de GHCR e recomendou preservar o nome operacional Compose, banco, configuração e upstream.

A pasta foi movida integralmente para `/Users/evertonberger/Projects/TuneTrail`. O VS Code usa `${workspaceFolder}` e precisa reabrir essa pasta. `make version.check`, testes de versionamento e testes dos templates passaram; os remotos foram conferidos após a mudança. A consulta remota do GitHub não retornou conteúdo no ambiente; a existência da imagem nova não foi verificada nem houve publicação.

## Encerramento — 09/11/2026

Fechada e aprovada por solicitação explícita do usuário. A mudança de diretório
e os links próprios já foram aplicados. A definição do endereço oficial da
documentação, a substituição de `site: https://koito.io`, a revisão de links do
site e a preparação de sua publicação passam à
[issue 048](../../docs-dev/issues/048-documentacao-site-tunetrail.md). Nenhum domínio, implantação ou
acesso remoto foi confirmado por este encerramento.
<!-- SECTION:DESCRIPTION:END -->
