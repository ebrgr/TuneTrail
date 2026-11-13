# Issue 011 — administração, perfis, chaves e relay

**Atualização na issue 015:** integração Go, jornadas reais e comparação visual
registradas em [ISSUE_015.md](ISSUE_015.md). As notas abaixo descrevem a entrega
anterior à integração; o aceite final do usuário permanece pendente.

Núcleo implementado em 05/11/2026. [Issue](<../../tasks/issue-011 - 011-—-Migrar-conta-perfis-avatares-chaves-e-relay-para-Go.md>) · [Plano](<../../docs/roadmaps/doc-007 - Roadmap-4-—-administração-importação-chaves-edição-deleção-e-mesclagem.md>).

## Entrega funcional

`internal/adminui` prepara um handler HTML com login, logout, conta, seleção por
ID, perfis, avatar, chaves e relay. Os formulários usam sessão, SameOrigin,
CSRF validado antes de gravar arquivo, no-store e POST/redirect/GET. O retorno
interno preserva um perfil válido no login. Há confirmação para revogar chave,
e a chave pode ser selecionada e copiada manualmente sem JavaScript.

O handler não foi ligado ao dispatcher de produção: `/admin` em React ainda inclui
importação/exportação da issue 012. Na 015 é necessário integrar a família completa,
compartilhar o limitador de login entre HTML e JSON, testar as jornadas reais e
conferir o layout. O limitador local do handler isolado não basta para essa
integração. Exclusão de perfil pertence à 014. A issue permanece aberta até o aceite.

Os serviços compartilhados por JSON e HTML validam administrador, ID e slug atual,
escopo da chave e conflitos. Perfil e chave inicial são gravados atomicamente; a
falha da foto posterior é informada sem desfazer o perfil. O avatar usa referências
globais; fotos substituídas ficam para o coletor. A coleta e a publicação de avatar
são sincronizadas no mesmo processo; falha da consulta de associação preserva os
arquivos. Não é uma coordenação distribuída entre processos.

O relay mantém destinos HTTP(S) self-hosted. Teste não salva nem envia escuta;
teste e envio recusam redirects, limitam tempo e corpo e não expõem token ou corpo
remoto em mensagens. Envio continua melhor esforço, sem fila persistente; tentativas
após erro 5xx usam o payload completo e têm prazo limitado. Mudanças na política
podem exigir atualizar um destino anteriormente acessado por redirect.

## Validação

- `go test ./...` passou com portas locais temporárias; SQLite v3 descartável.
- `go test -race ./internal/admin ./internal/adminui ./internal/adminweb ./internal/relay ./internal/catalog` passou.
- Testes focados cobrem criação atômica e falha do banco, perfis em abas distintas,
  slug obsoleto, chave de outro perfil, confirmação/revogação, retorno de login,
  CSRF, logout, erro de foto após criação e configuração/teste/remoção do relay.
- Relay simulado cobre token inválido, redirects, resposta excessiva, timeout,
  URL inválida e repetição com payload íntegro. Nenhuma conta externa foi contatada.
- `TestAvatarPublicationAndCollectorSafety` reproduz a publicação em paralelo com
  a coleta e injeta falha na consulta; `TestProfileAvatarLifecycle` cobre foto
  compartilhada, arquivo inválido e limite. Dados e arquivos são fictícios.
- `scripts/validate_multiuser.py` passou em bases nova e migrada v2 para v3,
  incluindo importação/exportação e reinício. `scripts/validate_webui.py` passou
  em servidores locais Go e React, sem comparação visual; integridade SQLite OK.

Layout e screenshots foram intencionalmente adiados para o final da 015. Nenhum
banco operacional foi alterado, e nenhuma issue foi publicada.

## Revisão do Advisor

O Advisor (Sol, esforço alto) revisou os serviços e formulários. Os achados sobre
preservar o perfil no retorno do login, a corrida entre publicação/coleta de avatar
e os delimitadores vazios na URL do relay foram corrigidos e testados. Após nova
revisão dos ajustes, o parecer foi **aprovado**.
