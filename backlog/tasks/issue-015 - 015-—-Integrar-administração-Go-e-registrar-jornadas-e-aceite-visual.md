---
id: ISSUE-015
title: 015 — Integrar administração Go e registrar jornadas e aceite visual
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
references: []
type: feature
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/015-integracao-aceite-admin-go-solved.md` — numeração local 015. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](<../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>). Depende de 010–014.

## Trabalho

- Estender dispatcher Go, ponte React e manifesto de rotas juntos para documentos
  admin/gestão e actions; separar métodos e não classificar assets/API como páginas.
- Preservar bookmarks `/admin?profile=ID`, URLs manage, seleção pública, retorno
  interno, modo React padrão e opção Go. Ativar cada família apenas completa/testada.
- Integrar acesso nos Settings/Account/Administration e páginas de mídia; substituir
  links transitórios apenas quando o destino Go oferecer todas as ações previstas.
- Criar validação de jornadas em bancos v3 descartáveis e comparação React/Go em
  desktop/celular, Yuu/Pearl, abaixo da dobra, teclado/foco e alternativa sem JS.
- Registrar manifesto de rota/perfil/mode/fuso/viewport/região e dados fictícios,
  sem capturar chaves, tokens ou senhas reais.
- Revisão Advisor dos contratos de escrita e regressão; atualizar configuração,
  guias de perfis/importação/edição e resumo do roadmap após testes.
- Manter SPA/build/dependências e rotas React enquanto servirem de fallback.

## Aceite

Go público → gestão → login → ação → retorno; conta sem perfil; criar/renomear/
excluir perfil; copiar/revogar chave; configurar/testar/remover relay; importar/
exportar; editar/capa; escuta manual/exclusão e merge reverso. Duas abas/perfis,
recarga, voltar/avançar, sessão expirada, confirmação cancelada/desatualizada.

Testes Go, cliente, builds, validação multiusuário e negativas de autenticação,
CSRF e origem; falhas e sigilo em cada família. Evidências numéricas e visuais
complementares, sem afirmar paridade exata ou desempenho sem medição.
Nenhuma remoção de React, publicação ou alteração no banco operacional nesta issue.

## Agendamento visual

A orientação atual do usuário concentra a conferência de layout e as comparações
visuais de 010–015 somente ao final desta issue. Testes funcionais e de segurança
continuam obrigatórios nas entregas anteriores, sem antecipar screenshots.

## Entrega — 05/11/2026

Dispatcher, ponte React e manifesto GET/HEAD/POST atualizados. O login HTML e
JSON compartilham o mesmo limitador global; formulários têm sessão, CSRF e
tickets de envio de uso único. Layout/temas administrativos compartilham as
fontes e paletas da aplicação. Busca de artista/destino, escuta com fuso efetivo,
retorno filtrado e alternativa sem JavaScript estão integrados.

As validações Go, cliente, builds, multiusuário e jornadas administrativas
passaram. O Advisor aprovou a integração. Foram registradas **55 imagens**:
26 pares React/Go e três capturas de interação, em desktop/celular e Yuu/Pearl.
Nenhuma publicação ou mutação do banco operacional. Os servidores de teste são
encerrados ao concluir o trabalho.

[Relatório, comparações e limitações](../evidencias/roadmap-04/ISSUE_015.md).

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário ao solicitar o encerramento
de todas as issues abertas até a 015. Entrega, testes e limites conhecidos
permanecem registrados; validação transversal será estudada no roadmap 5.
<!-- SECTION:DESCRIPTION:END -->
