---
id: decision-011
title: 'Banco, metadados e subida de produção: decisões de 27/11/2026'
date: '2026-11-27T15:21:00Z'
status: accepted
---
## Context

Depois das ISSUE-214, 215 e 216 (v0.79) restavam decisões sobre o banco, os metadados e a subida da produção, vindas do doc-046 §8, do doc-047 §7, do doc-048 e dos pareceres. O usuário aceitou em 27/11/2026 todas as recomendações do coordenador, com uma exceção (B4).

## Decision

| # | Decisão | Onde |
| --- | --- | --- |
| A1 | Aprovado o nível 1 do banco | ISSUE-203, 204 |
| A2 | `integrations.token_ref`, `revoked_at` e `diagnostic_enabled` ficam | ISSUE-203 |
| A3 | Parar de gravar a revisão automática do recibo aceito | ISSUE-204 |
| A4 | Pacote B antes da consolidação | ISSUE-205, 206 |
| A5 | Uma única migração de metadados antes da consolidação | ISSUE-217, 218, 195 |
| A6 | Cada issue do banco com a própria migração, consolidada depois; uma branch com migração por vez | ISSUE-203 a 206, 195 |
| A7 | A consulta por ISRC é ligada na ISSUE-218 | ISSUE-218 |
| B1 | Decidir tudo, consolidar e subir a produção uma vez | ISSUE-224, 195 |
| B2 | Ensaio com 300 mil e 1 milhão de escutas | ISSUE-195 |
| B3 | Autorizada a medição dos índices | ISSUE-207 |
| **B4** | **Exceção do usuário:** em vez de guardar para sempre, opção de limpar e excluir o conteúdo original dos recibos após um período configurado | ISSUE-208, 225 |
| B5 | Remover `relay_sent_archive` | ISSUE-203 |
| B6 | `tracks` em `occurrences` só depois da subida | ISSUE-210 |
| B7 | Antes da subida, só as integrações que reformam tabelas existentes | ISSUE-195 |
| C1 | Aposentar o relay por variáveis de ambiente | ISSUE-209 |
| C2 | Remover `LOGIN_GATE` | ISSUE-197 |
| C3 | Revogar tokens pendentes na troca de papel | ISSUE-227 |
| C4 | Abrir as pendências técnicas da 162 | ISSUE-228 |
| C5 | Teste estático do filtro por perfil | ISSUE-211 |
| C6 | Reenvio de ID externo confiável vira duplicata silenciosa | ISSUE-229 |
| C7 | Corrigir a URL do MusicBrainz como instância | ISSUE-226 |
| C8 | Não enviar as tags antigas (v0.11 a v0.67) | operação |
| C9 | Corrigir o doc-048 agora; `DB_ALTERATIONS.md` quando a consolidação fechar | doc-048, ISSUE-195 |
| C10 | Teclado e leitor de tela na auditoria de acessibilidade | ISSUE-078 |

## Consequences

- A ordem passa a ser: metadados (218, 217), banco (203, 204, 205, 206, com 207 e 208 decididas antes de fechar o esquema), 193, 195, subida da produção (ISSUE-224) e só então a 210.
- A ISSUE-195 depende também das ISSUE-217 e 218.
- Na ISSUE-225, «desligada por padrão» e «escutas nunca são excluídas» são propostas do coordenador, ainda a confirmar.
- Os comportamentos atuais citados em C3, C4 e C6 não foram conferidos no código e precisam ser verificados antes de implementar.
