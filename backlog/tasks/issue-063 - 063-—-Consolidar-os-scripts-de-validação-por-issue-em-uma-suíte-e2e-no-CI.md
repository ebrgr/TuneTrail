---
id: ISSUE-063
title: 063 — Consolidar os scripts de validação por issue em uma suíte e2e no CI
status: Done
assignee:
  - '@ebrgr'
created_date: '2026-11-14T00:38:00Z'
updated_date: '2026-11-14T01:38:00Z'
labels:
  - validacao
  - processo
milestone: m-9
dependencies: []
references:
  - backlog/evidencias/e2e/ISSUE_063.md
priority: high
type: chore
ordinal: 63000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Origem: doc-010 §1.3 e §8 — 37 dos 50 scripts em `scripts/` têm sufixo de issue (`validate_*_0NN`, `capture_*_0NN`) e não rodam como suíte.

Passos:
1. Inventariar os 37 scripts: o que verificam, se já existe teste Go equivalente, se dependem de React removido.
2. Classificar cada um: coberto (remover), converter (nova jornada), histórico (mover para `scripts/historico/` com nota).
3. Criar a suíte de jornadas em Go (`httptest` + SQLite descartável) cobrindo: login/sessão, perfis, ingestão ListenBrainz, páginas públicas, administração, importação/exportação, permissões e isolamento entre perfis.
4. Ligar a suíte ao `make check` e ao workflow `test.yml`; capturas com navegador ficam opcionais (`make ui.capture`).
5. Atualizar `scripts/README.md`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Inventário dos 37 scripts com destino de cada um
- [x] #2 Suíte de jornadas executa em `make check` e no CI sem rede externa
- [x] #3 Jornadas cobrem perfis, ingestão, páginas públicas, administração, importação e isolamento
- [x] #4 Scripts de issue removidos ou arquivados como históricos, sem quebrar evidências citadas no backlog
<!-- AC:END -->

## Comments

<!-- COMMENTS:BEGIN -->
author: ebrgr
created: 2026-11-14T01:25:00Z

---

Início: branch `chore/issue-063-consolidar-os-scripts-de-validacao` criada a partir de `main`.

---

author: claude
created: 2026-11-14T01:32:00Z

---

Desvio do plano, registrado para o aceite: o passo 3 previa reescrever as jornadas em Go (httptest). Os validadores Python existentes já exercitam o binário real (migração, reinício, encerramento, processos separados), somam ~1.300 linhas e 8 deles passavam; reescrevê-los em Go duplicaria cobertura com risco maior. A suíte consolidada é scripts/e2e.py, com um ponto de entrada (make e2e) dentro de make check. Os critérios de aceite são os mesmos.

---

author: ebrgr
created: 2026-11-14T01:32:00Z

---

Verificação: (1) inventário dos 37 scripts com destino e motivo em backlog/evidencias/e2e/ISSUE_063.md — 7 mantidos sem sufixo de issue, 30 em scripts/historico/ (React removido ou capturas pontuais); (2) scripts/e2e.py roda 10 jornadas (multiuser, admin, public, permissions, isolation, migration, standalone, shutdown, vscode, tools) em 127.0.0.1 com dados fictícios e serviços externos desligados; make check inclui make e2e e passou localmente (10/10, 31 pacotes Go); a suíte sai com código 1 quando uma jornada falha (testado com binário inválido); test.yml atualizado, mas o CI só será exercitado após o push, por isso o critério 2 segue aberto; (3) jornadas cobrem perfis, ingestão, páginas públicas, administração, importação/exportação e isolamento; (4) 0 links quebrados no backlog; menções em texto aos nomes antigos preservadas como histórico, com tabela antes/agora em scripts/README.md. Correções extras: validate_webui.py (lia koito.db e subia React) e serve_roadmap06.py (modo React) estavam quebrados fora do CI. Commits 39805fe e 663d3d4.

---

author: claude
created: 2026-11-14T01:33:00Z

---

Aceite: aprovada pelo usuário em 13/11/2026.

---

author: claude
created: 2026-11-14T01:33:00Z

---

Publicação: integrada ao main por fast-forward sem versão nem tag, a pedido do usuário (somente testes e ferramentas), e enviada ao GitHub; o critério 2 será marcado após o workflow Test passar no push.

---

author: claude
created: 2026-11-14T01:38:00Z

---

Verificação do critério 2: workflow Test aprovado no GitHub para o commit 4c04172 (https://github.com/ebrgr/TuneTrail/actions/runs/43541472021), com make check incluindo a suíte e2e.

---
<!-- COMMENTS:END -->
