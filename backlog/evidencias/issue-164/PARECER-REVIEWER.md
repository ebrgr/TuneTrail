# ISSUE-164 — Parecer do Reviewer (23/11/2026)

Revisão do commit 1f13d7d feita por quem despachou, sem delegar. Lido o diff completo de scripts, testes, guias e evidências, e conferidas no código as afirmações específicas do guia operacional.

**Veredito: APROVADO.** Nenhum bug. Nenhuma tarefa nova foi aberta, porque a matriz não encontrou desvio de permissão ou de visibilidade. Quatro riscos de manutenção.

## Verificado

- [OK] A matriz cobre os nove papéis e estados em HTML, JSON e serviços, e diz de onde vem cada resultado (testes Go, jornadas do e2e e o teste de navegador com JavaScript ligado e desligado). O 404 de perfil protegido é comparado com o de perfil ausente no corpo e nos cabeçalhos.
- [OK] Os scripts de validação existentes ficaram mais exigentes, não menos: a conta migrada agora exige um administrador ativo, o backup precisa ter permissão `0600`, e a restauração é provada subindo o servidor sobre o backup restaurado (antes só se abria o arquivo com SQLite). A constante de versão passou a vir dos nomes dos arquivos de migração, então acompanha as migrações novas sem edição manual.
- [OK] A migração do schema 2 ao 22 preservou 72 escutas e o digest agregado é o mesmo na entrada, depois da migração e depois de restaurar o backup e subir de novo. As evidências trazem só contagens, versões e um digest, sem nomes, títulos, usuários ou tokens.
- [OK] O guia `multiuser-operations.md` bate com o código: o subcomando `tunetrail reset-password` existe (`cmd/api/main.go`), os tokens expiram em 24 h (ativação) e 30 min (recuperação) (`account_tokens.go`), e a senha exige ao menos 8 caracteres e no máximo 72 bytes (`admin/auth.go`).
- [OK] Não afirma que `TUNETRAIL_LOGIN_GATE` protege as estatísticas (ISSUE-197), e diz que o relay continua enviando escutas de perfil protegido, a consequência já aceita.
- [OK] As correções em `profiles.md`, `installation.md` e `DEPLOYMENT.md` trocam afirmações que já estavam falsas («perfis e escutas são públicos», «perfis privados não existem») pela situação atual. O parágrafo da cor do avatar (ISSUE-202) foi preservado.

## Riscos

- [RISCO] O guia, o `DEPLOYMENT.md` e o `MIGRACAO.md` falam em «schema 22» e «0.72». A ISSUE-190, em andamento em paralelo, sobe para o schema 23. Quem publicar depois precisa atualizar esses números nos guias; o `MIGRACAO.md` pode ficar como registro histórico do que foi medido.
- [RISCO] O teste de navegador (`validate_multiuser_browser.py`) exige Playwright e Chromium, que não estão nas dependências do repositório nem no `make check`. Está documentado, mas fica manual.
- [RISCO] A inspeção de teclado e de leitor de tela ficou fora da execução, como o próprio relatório diz.
- [RISCO] O digest agregado inclui as credenciais e chaves da cópia real. É um hash, não reversível, e a cópia ficou fora da worktree; ainda assim, quem publicar o relatório deve saber que o valor descreve uma amostra real.

## Fora do alcance

- O aceite é do usuário.
