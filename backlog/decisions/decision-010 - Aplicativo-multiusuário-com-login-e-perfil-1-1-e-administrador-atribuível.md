---
id: decision-010
title: Aplicativo multiusuário com login e perfil 1:1 e administrador atribuível
date: '2026-11-17T17:17:00Z'
status: proposed
---
## Context

O modelo anterior (doc-021/doc-022 e doc-010 §7) separava perfis públicos sem login de uma conta administrativa. O usuário definiu na ISSUE-100 um aplicativo multiusuário com um login por perfil, administrador atribuível e proteção opcional das escutas. As decisões complementares estão no comentário de 17/11/2026 da tarefa. Contrato formal: doc-041, `backlog/docs/design/doc-041 - Aplicativo-multiusuário-contas-administrador-e-visibilidade-das-escutas-ISSUE-100.md`.

Este registro permanece **proposed** até aceite explícito posterior do usuário; as escolhas já registradas não significam aceite desta redação nem implementação concluída.

## Decision

1. Manter conta e perfil separados, vinculados 1:1 por `listening_profiles.user_id NOT NULL UNIQUE`, com criação/exclusão da dupla em transação. Apenas admin cadastra e exclui contas/perfis.
2. Usar o papel existente `admin` para o checkbox Administrador (root), atribuível a qualquer conta somente por outro admin. Usuário comum gerencia suas próprias escutas, chaves e importação/exportação; root gerencia todos os perfis e o servidor. Capas, aliases, merges e catálogo compartilhado continuam exclusivos do root. Proteger o último administrador contra exclusão/rebaixamento na mesma transação que altera o papel ou a conta.
3. A raiz `/` lista todos os perfis com nome/avatar. `require_login=0` permite leitura pública e é o padrão para perfis existentes e novos. Quando marcado, somente dono ou admin lê escutas e derivados. Toda gestão exige sessão e autorização no servidor. Aplicar gate central em páginas, API, busca, widget, imagens e cache; identidade do diretório continua pública.
4. Manter ingestão por chave de máquina e tokens atuais. A proteção local **não pausa recebimento nem relay de saída**. Escutas protegidas podem continuar sendo publicadas no destino externo: consequência expressamente aceita pelo usuário e a informar na interface.
5. Preservar bcrypt, custo e validação atuais, sessões HttpOnly, CSRF/tickets e revogação transacional de sessões na troca de senha já existentes. Não introduzir Argon2id, mínimo de 15 caracteres, lista de senhas comuns, expiração periódica, bloqueio, bucket por usuário ou análise de origem/IP. Não adicionar limite de tentativas por usuário. A permanência do limitador compartilhado já existente, 10/min quando habilitado, é a única pergunta pendente; recomenda-se mantê-lo como está, sem novos controles.
6. Recuperar acesso sem e-mail por token temporário de uso único emitido pelo root, limitado a definir a senha da conta; reset usa o padrão atual e revoga sessões. Perfis existentes sem usuário recebem conta própria e ativação por token; não compartilhar senha administrativa. Preservar conta/hash do admin, IDs, escutas e chaves atuais. Adaptar bootstrap e coordenar a numeração da migração com ISSUE-080/133 sem editar SQL já aplicado.

## Consequences

Os contratos de acesso anteriores são revisados; decisões 008/009 de identidade musical e ingestão permanecem válidas. Login comum não pode abrir operações globais; esconder controles na interface não substitui autorização. A visibilidade pública das identidades e a continuidade do relay são intencionais, e a proteção local não desfaz cópias externas já publicadas.

Não adotar conta com vários perfis, tabela N:N de pertencimento, senha extra de visualização, senha root compartilhada ou API key administrativa. Não reformar a política de senhas nesta entrega. O doc-041 §10 propõe etapas e critérios de implementação sem criar novas issues; ISSUE-125–128/157/153 devem ser coordenadas para evitar duplicação.

Pendência única: manter como está ou remover o limitador compartilhado de login existente? Recomendação: manter, preservando seu comportamento/configuração atuais. Se removido, não substituí-lo por outro mecanismo. O trecho do AC2 da ISSUE-100 sobre limite de tentativas precisa ser alinhado à resposta antes de implementar; nenhum AC foi marcado por esta documentação.
