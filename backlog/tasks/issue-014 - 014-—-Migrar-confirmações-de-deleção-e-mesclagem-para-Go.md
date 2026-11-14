---
id: ISSUE-014
title: 014 — Migrar confirmações de deleção e mesclagem para Go
status: Done
assignee: []
created_date: '2026-11-12T23:39:00Z'
labels:
  - admin
milestone: m-0
dependencies: []
type: feature
ordinal: 80000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
> Importada de `docs-dev/issues/014-delecao-mesclagem-go-solved.md` — numeração local 014. Original removido em 12/11/2026; consulte o histórico do Git.

Status: **fechada e resolvida**. Issue local, sem publicação.
Plano: [roadmap 4](<../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>). Depende de 010 e composição 011/013.

## Trabalho

- Plano de impacto global para excluir/mesclar artista/álbum/faixa, mostrando
  origem, destino, ID sobrevivente, relações/escutas/perfis afetados e capa.
- Confirmação em duas fases nas novas actions HTML: validar no backend e
  revalidar identidade, existência e impacto no mesmo tx; conflito retorna 409.
  JSON existente conserva payloads, compartilhando autorização e invariantes do
  tx; exigir plano prévio no JSON seria mudança coordenada própria.
- Integrar exclusão de perfil com nome e contagem já exigidos atomicamente;
  distinguir preservação do catálogo de exclusão global de mídia.
- Validar IDs positivos/distintos/existentes no tx e manter consultas transacionais;
  completar correções da 010 com rollback de falhas injetadas e concorrência.
- Definir/testar política de aliases/artistas/MBIDs/capa do sobrevivente; preservar
  deduplicação de faixa somente dentro do mesmo perfil e segundo.
- Remover arquivos/caches após commit e checar associação global; não oferecer
  rollback/undo fictício de exclusão concluída nem usar history.back após apagar.
- Retornar à página pessoal do sobrevivente apenas se relacionada ao perfil;
  caso contrário home/histórico válidos. Confirmar direção inversa sem estado obsoleto.
- Manter API global de purga existente; não criar nova tela de purga como paridade.

## Aceite

Dois perfis com catálogo comum e mesmo timestamp; merge em ambos sentidos, self
merge de todos os tipos, origem/destino ausente ou alterado, IDs inválidos e planos
obsoletos. Duplicata pessoal reduz contagem esperada; pessoas diferentes conservam
as próprias escutas. Verificar aliases, relações, capa e IDs finais.

Exclusão de artista com relações/cascatas distintas, álbum, faixa, escuta e último
perfil; nome/contagem desatualizados; falha de commit/limpeza e ingestão concorrente.
Sessão/CSRF obrigatórios inclusive confirmação final. GET/cancelar não altera nada.
Executar operações destrutivas somente em SQLite v3 descartável.

## Andamento — 05/11/2026

A revisão Go isolada apresenta origem/destino, metadados, relações, capas,
perfis e escutas, inclusive cascata real de artista e duplicatas por perfil.
A confirmação usa sessão, CSRF e plano de uso único; o SQLite recalcula o
fingerprint e executa a operação no mesmo tx. Mudanças e conflitos WAL exigem
nova revisão. A exclusão de perfil confirma nome e contagem atuais.

A limpeza global indevida foi removida dos merges e da exclusão de faixa:
mídias não relacionadas e destinos sem escutas são preservados. Imagens órfãs
aguardam o coletor existente; falha de limpeza não reverte operação já concluída.
O fingerprint é conservador e inclui o catálogo e histórico inteiros: alteração
não relacionada pode exigir revisão nova, e o custo cresce com a base.

A interface ainda não está ligada ao dispatcher; integração, busca de destino
mais rica, aceite completo e comparação visual continuam na 015.
Detalhes e resultados: [relatório](../evidencias/roadmap-04/ISSUE_014.md).

## Integração na 015 — 05/11/2026

A administração foi ligada ao modo Go; os contratos e jornadas são verificados
pelo servidor real e pelas capturas comparativas. As pendências de integração
descritas no andamento anterior são históricas. Consulte
[entrega e limites da 015](../evidencias/roadmap-04/ISSUE_015.md).

## Encerramento — 05/11/2026

Fechada e resolvida por aceite explícito do usuário ao solicitar o encerramento
de todas as issues abertas até a 015. Entrega, testes e limites conhecidos
permanecem registrados; validação transversal será estudada no roadmap 5.
<!-- SECTION:DESCRIPTION:END -->
