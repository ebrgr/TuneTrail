# Comparação de responsividade — Koito React × TuneTrail Go

**Data:** 13/11/2026 · **TuneTrail:** 0.8 (`d7cffa3`) · **Koito:** instância local em `127.0.0.1:4111` (Remix/React, perfil único, 4.224 escutas, 216 faixas) · **Tarefas:** ISSUE-106, ISSUE-107; contexto em ISSUE-056 e ISSUE-075.

## Método

- **Koito:** código do cliente lido no histórico do repositório (`git show 5c1affa^:client/...`: `SearchModal.tsx`, `ComboBox.tsx`, `EditModal/*`) e tempos da API medidos na instância 4111 (mediana de 5 requisições, loopback).
- **TuneTrail:** binário 0.8 contra a fixture `large` de `scripts/roadmap06_fixture.py` (1.009 escutas, dados fictícios, diretório descartável), login de administrador com `validate_admin_webui.Forms`, mediana de 7 requisições. Script: `measure_go_ui.py` (scratchpad da sessão, não versionado).
- Não houve medição no navegador (renderização, pintura, fontes). Ela entra na referência da ISSUE-065.

## Busca

| | Koito React | TuneTrail Go |
| --- | --- | --- |
| Interação | Busca enquanto digita (`useDeferredValue`), resultados no próprio modal; respostas antigas descartadas | Diálogo com formulário; Enter navega para `/{slug}/search?q=` |
| Requisição por busca | `GET /apis/web/v1/search?q=` → JSON | Página HTML completa (barra lateral, cabeçalho, scripts) |
| Tempo no servidor | 0,8–2,2 ms | 1,4–1,6 ms |
| Tamanho | 2,2–6,5 KB | 14,4 KB + CSS/JS sem cache |
| Ao voltar | Estado preservado no SPA | Página refeita (`no-store` impede o bfcache) |

## Edição de faixa

| | Koito React | TuneTrail Go |
| --- | --- | --- |
| Estrutura | Modal com seções independentes (apelidos, artistas, MBID) | Página `/{slug}/manage/track/{id}` com 42 formulários, 25 escutas e busca de mesclagem |
| Ação (ex.: adicionar apelido) | 1 `fetch`; só a lista afetada é redesenhada; botão em carregamento | POST → 303 → GET da página inteira: 2 requisições, 4,1 ms, 36,5 KB |
| Procurar artista para adicionar | Combobox com busca enquanto digita | Formulário `?artist_query=` recarrega a página (3,1 ms, 30,9 KB) |
| Abrir | Modal sobre a página atual | Navegação: 3,2 ms, 29,3 KB |
| Rolagem e foco | Preservados | Perdidos a cada ação (volta ao topo) |

## Arquivos estáticos

| | Koito React | TuneTrail Go |
| --- | --- | --- |
| Cache | Nomes com hash, `Cache-Control: public, max-age=31536000, immutable` | Sem `Cache-Control`, `ETag` ou `Last-Modified` em `/web-assets/*`: baixados de novo a cada navegação |
| Fontes | — | Folha do Google Fonts no `<head>`, bloqueando a renderização |
| Script no `<head>` | Módulos | `theme.js` síncrono (necessário para evitar flash de tema; pequeno) |

## Conclusão

O servidor Go é tão rápido quanto o do Koito (1–4 ms). A diferença percebida vem do **modelo de interação** — cada busca e cada ação de edição refaz a página inteira, perde rolagem e foco — e de **custos repetidos no navegador**: arquivos estáticos sem cache, fonte externa bloqueante e `no-store` nas páginas públicas. Nenhum desses pontos exige React: fragmentos HTML do próprio servidor, um script pequeno de melhoria progressiva e cabeçalhos de cache resolvem a maior parte.

## Página de faixa — `/track/23` (Koito) × `/evertonberger/track/91` (TuneTrail)

Acrescentado em 13/11/2026 a pedido do usuário. Koito lido em `client/app/routes/MediaItems/MediaLayout.tsx`; TuneTrail lido em `internal/webui/templates/media.html`, `internal/adminui/templates/manage.html` e na página servida em `192.168.88.33:4110` (somente GET). Tarefas: ISSUE-108 (barra de ações e edição por tarefa) e ISSUE-073 (cabeçalho visual).

| | Koito React | TuneTrail Go |
| --- | --- | --- |
| Ações no topo | Ícones Lucide no canto superior direito do cabeçalho (30px; 22px no celular): MusicBrainz (quando há MBID), **Adicionar escuta** (faixa), **Editar**, **Trocar imagem** (álbum/artista), **Mesclar**, **Excluir** | Um link de texto "Manage this item" ao fim do cabeçalho |
| Quem vê | Somente com usuário logado | **Todos os visitantes**; quem não é admin cai no login |
| O que cada ação abre | Um modal por tarefa sobre a própria página | Uma página única com 42 formulários: apelidos, MBID, artista principal, busca e lista de artistas, fuso, lista e exclusão de escutas, adicionar escuta, escuta selecionada, exclusão, busca e revisão de mesclagem, capa |
| Trocar imagem | Modal ou arrastar a imagem sobre a página | Formulário no fim da página de gestão |
| Cabeçalho | Capa grande (385px), tipo, título grande com posição no ranking, contagens; fundo em degradê da cor média da capa | Capa, tipo, título, artista, álbum, contagens; sem cor de destaque |
| Escuta no histórico | — | Link "Manage" por linha leva à mesma página misturada, com a escuta selecionada |

Conclusão: a navegação do TuneTrail concentra todas as ações em uma página que mistura tarefas sem relação entre si. Separar por tarefa, com ícones no topo e um diálogo para cada ação, aproxima o comportamento do original e reduz o trabalho a cada clique.
