# ISSUE-074 — Parecer do Reviewer (23/11/2026)

Revisão do diff `main...HEAD` (commits 2091315 e fc88ec2) feita por quem despachou, não pelo Codex. Medido em instância isolada, com a branch construída em fc88ec2, em 390, 768 e 1440 px, tema Yuu.

**Veredito: APROVADO COM RESSALVAS.** Nenhum bug bloqueante. Quatro riscos para decidir antes do aceite.

## Verificado

- [OK] Largura dos cartões em 1440 px: a área útil passou de 1100 para 1302 px; Chaves e Relay passaram de 542 px (49%) para 1302 px (100%); Conta, Perfil, Histórico e Configuração mantêm duas colunas de 643 px, e o cartão ímpar final ocupa a linha. Em 390 e 768 px todos têm 100% e não há rolagem horizontal.
- [OK] Bloco da chave: com o detalhe aberto, a chave fica num bloco próprio (288, 544 e 1216 px de largura nos três tamanhos), sem corte, e o botão Copiar fica abaixo, com 44 px de altura.
- [OK] Hover e pressionado: o script de cobertura (corrigido para seletores com vírgula dentro de parênteses) não encontra nenhum button, summary ou link-botão sem regra `:hover` e `:active` em /account, /account/profile, /account/keys, /account/relay, /account/history, /admin/settings, /admin?section=profiles e /admin/health. Com o mouse: botão primário 229,132,106 → 232,153,130; secundário, aba e Aplicar 25,19,16 (ou 35,27,24) → 60,46,42; Copiar → 60,46,42.
- [OK] `scripts/check_web_style.py`: a exigência de hover e pressionado saiu de `.account-action` e `.account-secondary` e passou para a regra compartilhada `SHARED_STATES`, que continua sendo cobrada com teste. Não enfraquece a checagem.
- [OK] A remoção do botão Copiar genérico de `details code` em `admin.js` é segura: nenhum outro template de administração tem `<details><code>`. O botão agora é renderizado pelo servidor, fica oculto sem JavaScript e funciona após a troca de fragmento.
- [OK] O `PARECER.md` com a palavra APROVADO, escrito pelo Codex, foi removido.

## Riscos

- [RISCO] `assets/css/admin/admin.css` (regra compartilhada `.admin-main :is(button,summary,a):not(:disabled):hover`): vale para todo link da administração, inclusive links de texto dentro de parágrafos, que ganham fundo cinza ao passar o mouse. O hover em «Gerenciar perfis e relay» não foi confirmado no navegador. Decidir se links de texto devem ter fundo ou só sublinhado.
- [RISCO] Conflito com a ISSUE-201: as duas branches corrigem o mesmo envio errado de `enhance.js` (a 074 com `??`, a 201 com `||` e teste de regressão). Conflito certo em `CHANGELOG.md`, `assets/js/enhance.js` e nos arquivos de `backlog/evidencias/ISSUE-075/reference`. Publicar a 201 primeiro, rebasear a 074 ficando com a versão da 201 e regenerar a referência.
- [RISCO] Sem teto de largura, em monitores muito largos os cartões e campos de uma coluna se estendem por toda a tela. Foi o pedido do usuário; não foi medido acima de 1440 px.
- [RISCO] Peso das evidências: 102 capturas, 7,7 MB no Git. Aceitável, mas cada nova rodada de capturas soma ao histórico.

## Fora do alcance desta revisão

- O relay com exceções por destino e as telas de revisão foram cobertos pelo Codex (1.920 verificações); não os repeti.
- O aceite visual do conjunto (AC 3) é do usuário.
