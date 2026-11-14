---
id: ISSUE-048
title: 048 — Documentação e site oficial do TuneTrail
status: Aberta
assignee: []
created_date: '2026-11-12T23:39:00Z'
updated_date: '2026-11-14T00:39:00Z'
labels:
  - docs
milestone: m-15
dependencies: []
type: docs
ordinal: 48000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/048-documentacao-site-tunetrail.md` — numeração local 048. Original removido em 12/11/2026; consulte o histórico do Git.

**Status:** Aberta · **Criada:** 09/11/2026 · **Tipo:** documentação e site

Origem: encerramento aprovado das issues [043](<issue-043 - 043-—-Identidade-visual-TuneTrail.md>),
[044](<issue-044 - 044-—-Versionamento-próprio-do-TuneTrail.md>) e
[045](<issue-045 - 045-—-Links-e-diretório-do-TuneTrail.md>). Issue local, sem publicação no GitHub.

## Objetivo

Organizar e atualizar a documentação do usuário e preparar o site próprio do
TuneTrail, com identidade, navegação e informações coerentes com a aplicação
atual: interface Go/HTML/CSS/JavaScript essencial, execução sem libvips/CGO e
SQLite com esquema consolidado v3.

## Situação de partida

- O site existente usa Astro/Starlight em `docs/`; a identidade TuneTrail já foi
  aplicada parcialmente. O build da documentação ficou sem verificação na 043.
- `docs/astro.config.mjs` ainda define `site: https://koito.io`. Não há endereço
  próprio aprovado para substituí-lo neste planejamento.
- O repositório é `ebrgr/TuneTrail` e a imagem própria é
  `ghcr.io/ebrgr/tunetrail`. Confirmar tags publicadas antes de recomendá-las.
- As fontes e evidências de desenvolvimento ficam em `docs-dev/`; estudos locais,
  relatórios privados, configurações reais e bancos não são conteúdo do site.
- Este trabalho trata da documentação e do site; a etapa TOML 038–042 continua
  em espera. Não documentar funcionalidades planejadas como já disponíveis.

## Trabalho

1. Inventariar páginas e referências entre README, DEPLOYMENT, `docs/` e roadmap;
   definir o conteúdo público e a navegação, evitando instruções contraditórias.
2. Revisar introdução, instalação local, Docker/Compose e Unraid, porta 4110,
   atualização, volumes, configuração vigente e recuperação de acesso.
3. Documentar perfis e seus limites atuais, API/ingestão ListenBrainz, importação,
   exportação, relay best-effort, imagens suportadas e permissões administrativas.
   Incluir backup/restauração e migração de `koito.db` para `tunetrail.db`,
   distinguindo versão do aplicativo de versão do esquema SQLite.
4. Aplicar identidade TuneTrail no site, incluindo logo, slogan, crédito ao Koito,
   títulos, favicon, ícones, metadados e links. Preservar licença e atribuição;
   referências legadas necessárias devem ser identificadas como históricas.
5. Definir com o usuário o endereço e a hospedagem antes de configurar URL
   canônica, base path e, se aplicável, domínio/DNS. Revisar referências a
   `koito.io` e decidir sobre scripts de analytics herdados e serviços externos.
   Não comprar domínio nem alterar DNS nesta issue sem instrução específica.
6. Preparar build reproduzível e fluxo de implantação/preview apropriado ao
   destino escolhido, separado do build Go e da imagem runtime. Avaliar o setup
   Astro existente antes de propor troca de stack; documentar requisitos do site.
7. Registrar capturas desktop e celular do site e, quando úteis nos guias,
   da interface TuneTrail atual, usando dados fictícios. Conferir logo, rodapé,
   navegação, links, foco, contraste, imagens e responsividade.
8. Atualizar índice, roadmap e instruções de manutenção da documentação.

## Critérios de aceite

- [ ] Conteúdo de uso coerente com a versão atual, sem instruções de React ou
  libvips como dependências runtime e sem promessas de TOML/privacidade futuras.
- [ ] Endereço e destino de hospedagem definidos; URLs canônicas, caminhos,
  assets e links próprios configurados e verificados para esse destino.
- [ ] Instalação, configuração e backup descritos com exemplos fictícios,
  sem segredos ou caminhos pessoais; tags de imagem verificadas quando citadas.
- [ ] Build e preview do site executados com sucesso e instruções reproduzíveis.
- [ ] Identidade, navegação e capturas verificadas em desktop/celular; registrar
  resultados e limites de acessibilidade, sem alegar auditoria completa.
- [ ] Processo de publicação preparado e documentado, com as permissões
  necessárias; publicação e alterações externas somente com autorização explícita.
- [ ] README, documentação, índice de issues e roadmap apontam para referências
  consistentes; os estudos privados permanecem fora do conteúdo publicado.

## Entrega esperada

Documentação revisada, site pronto para aceite, evidências de build/preview e
capturas em `backlog/evidencias/issue-048/` quando a execução começar.
Este registro não inicia a implementação nem publica o site.
<!-- SECTION:DESCRIPTION:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: claude
created: 2026-11-14T00:39:00Z

---

Plano TuneTrail 1.0 (doc-032): Fase 6 · Operação; requisito da candidatura ao 1.0 (ISSUE-105).

---
<!-- COMMENTS:END -->
