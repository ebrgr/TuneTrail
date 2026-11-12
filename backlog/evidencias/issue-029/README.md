# Issue 029 — comparação da navegação administrativa

Execução em 05/11/2026 com Chrome headless, SQLite v3 fictício e rede externa
bloqueada. Capturas Yuu a 1440×1000 e 390×844; as fontes externas foram bloqueadas,
portanto as imagens mostram fontes de fallback. Os arquivos `public` são a
referência da página de Alpha; conta e perfil usam o mesmo administrador e dados
fictícios, sem fotografar senhas ou chaves.

| Página | Antes | Depois | Referência pública |
| --- | --- | --- | --- |
| Conta desktop | [Antes](before-account-desktop.png) | [Depois](after-account-desktop.png) | [Pública](after-public-desktop.png) |
| Perfil desktop | [Antes](before-profile-desktop.png) | [Depois](after-profile-desktop.png) | [Pública](after-public-desktop.png) |
| Conta celular | [Antes](before-account-mobile.png) | [Depois](after-account-mobile.png) | [Pública](after-public-mobile.png) |
| Perfil celular | [Antes](before-profile-mobile.png) | [Depois](after-profile-mobile.png) | [Pública](after-public-mobile.png) |

A navegação administrativa usa os estilos e ícones equivalentes da barra pública:
vertical no desktop e superior no celular. Com perfil selecionado, os links de
home, busca e retrospectiva apontam para esse perfil. O seletor leva à lista pública;
as configurações administrativas continuam em um diálogo próprio, com escolha de
tema e links para conta/administração. Ele não é o diálogo completo da página pública.
Sem JavaScript, links de navegação e conta permanecem; escolha de tema requer script.

`capture_sidebar_029.py` verificou as duas larguras, links com/sem perfil, tema
Pearl e persistência, Escape e retorno de foco, links sem JS e assets dos cinco
ícones de avatar. `go test ./internal/adminui ./internal/webui ./engine`, `go vet`
dos pacotes de interface e `validate_admin_webui.py` passaram; o último cobriu
login, formulários, perfis, chaves, import/export, relay local e integridade. As
capturas não constituem auditoria geral de acessibilidade ou de todos os temas.

Advisor (Sol, esforço médio) aprovou a implementação e as capturas por leitura;
seus apontamentos sobre ícones de avatar e falha de imagem foram tratados após
essa revisão. Os servidores temporários foram encerrados.
