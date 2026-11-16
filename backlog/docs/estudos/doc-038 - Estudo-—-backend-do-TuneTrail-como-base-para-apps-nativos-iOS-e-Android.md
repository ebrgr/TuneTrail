---
id: doc-038
title: Estudo — backend do TuneTrail como base para apps nativos iOS e Android
type: other
created_date: '2026-11-16T01:04:00Z'
updated_date: '2026-11-16T01:04:00Z'
tags:
  - estudo
---
# Estudo — backend do TuneTrail como base para apps nativos iOS e Android

Data: 16/11/2026. Base inspecionada: `main` na v0.24. Leitura estática de `engine/routes.go`, `engine/middleware/`, `internal/adminweb/` e `internal/models/user.go`. Não foram chamados endpoints nem conferidos os formatos exatos das respostas JSON.

## Resposta curta

Sim. O backend em Go está no projeto e já expõe uma API JSON utilizável por um app nativo. Um app **somente de leitura** (estatísticas, histórico, "tocando agora", busca) funciona hoje sem login. Para um app completo faltam autenticação por dispositivo, contrato estável da API e a adaptação aos perfis privados (ISSUE-100).

## O que existe

O mesmo binário serve as páginas HTML (templates em Go; o front React foi removido na ISSUE-030) e três superfícies de API:

| Superfície | Caminho | Autenticação | Uso pelo app |
| --- | --- | --- | --- |
| API web JSON, leituras | `/apis/web/v1/profiles/{slug}/…`: perfil, `stats`, `top/tracks`, `top/albums`, `top/artists`, `listens`, `listen-activity`, `now-playing`, `search`, `summary`, detalhes e `interest` de artista, álbum e faixa | Nenhuma (perfis públicos hoje) | Sim, hoje |
| API web JSON, gestão | mesma base: editar, mesclar, excluir, chaves de API, importar, exportar | Cookie `tunetrail_session` (HttpOnly, SameSite Strict) de **administrador** | Possível, desconfortável |
| Compatível com ListenBrainz | `/apis/listenbrainz/1/submit-listens` e `validate-token` | Chave de API por perfil, `Authorization: Token <chave>` | Sim, hoje (envio de escutas) |
| Imagens | `/image/{id}/{filename}` | Nenhuma | Sim |

- Login: `POST /apis/web/v1/login` com `{username, password, remember_me}` responde 204 e define o cookie; limite de 10 tentativas por minuto.
- A verificação de origem das mutações (`adminweb.SameOrigin`) só rejeita quando o cabeçalho `Origin` existe e não coincide, ou quando `Sec-Fetch-Site` é `cross-site`. Apps nativos não enviam esses cabeçalhos e passam. O CORS é regra de navegador e não afeta app nativo.
- Existem usuários com papel `admin` ou `user`; as rotas de gestão exigem `admin`.

## Lacunas para um app completo

1. Sem autenticação por token: só cookie de administrador. Não há token por dispositivo, revogação de um celular específico nem renovação de sessão.
2. A chave de API de perfil vale somente nos endpoints do ListenBrainz (`IngestionKey`); não lê estatísticas e não tem escopos. A ISSUE-121 estuda escopos de chave e autenticação por proxy.
3. Sem contrato estável: sem OpenAPI nem versionamento; os formatos herdados do Koito podem mudar; algumas rotas são "transitórias". A ISSUE-104 prevê OpenAPI.
4. Perfis privados (ISSUE-100) vão exigir autenticação nas leituras hoje livres.
5. Sem tempo real: "tocando agora" é por consulta repetida; a ISSUE-084 planeja Server-Sent Events.
6. Servidor próprio: o app precisa pedir o endereço do servidor; no iOS, HTTP sem TLS exige exceção de segurança do sistema.

Licença do repositório: MIT; consumir ou reaproveitar o código não traz restrição relevante.

## Capturar escutas

- Android permite que um app observe a reprodução de outros aplicativos (notificação de mídia), então um app nativo pode ser um scrobbler completo.
- iOS não permite; um app de iPhone só registra o que ele mesmo toca. O restante depende de apps externos que enviem pelo ListenBrainz.
(Conhecimento geral das plataformas, não testado neste projeto.)

## Caminhos

- **A. PWA primeiro (ISSUE-076):** a interface já é responsiva; o PWA torna o TuneTrail instalável nas duas plataformas com pouco trabalho, sem app novo.
- **B. App nativo só de leitura:** consome as leituras atuais; viável já, mas preso a um formato não documentado e ao modelo de perfis públicos.
- **C. App nativo completo:** exige antes as lacunas 1 a 4.

## Recomendação

1. Fazer a ISSUE-076 (PWA) e verificar se atende.
2. Se o app nativo continuar valendo, concluir antes o contrato e a autenticação: OpenAPI (104), token por dispositivo com escopos (121) e perfis privados (100).
3. Escolher a tecnologia depois. Para visualização, um framework único para as duas plataformas costuma bastar; um scrobbler Android pede código nativo.

## Tarefas abertas a partir deste estudo

Ver as ISSUE-145 (contrato e autenticação por dispositivo), 146 (app nativo de leitura) e 147 (scrobbler Android nativo).
