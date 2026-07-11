# netwp

🇧🇷 Português · 🇺🇸 [English](README.md)

[![CI](https://github.com/gsjonio/netwp/actions/workflows/ci.yml/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql)
[![Dependabot](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph)
[![Go version](https://img.shields.io/github/go-mod/go-version/gsjonio/netwp)](go.mod)
[![Release](https://img.shields.io/github/v/release/gsjonio/netwp)](https://github.com/gsjonio/netwp/releases/latest)
[![License: MIT](https://img.shields.io/github/license/gsjonio/netwp)](LICENSE)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy_Me_a_Coffee-gugamenezes-FFDD00?logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/gugamenezes)

**netwp** = *Internet / Rede Well Played* ("a rede, bem jogada").

Gerenciador de rede via terminal escrito em Go: descoberta ativa de dispositivos
na rede local (ARP), monitoramento ao vivo, dashboard completo, teste de banda e
inspeção de interface. Windows primeiro, portável para Linux.

Nunca mexeu com redes? Comece pelo [guia para iniciantes](docs/GUIDE.pt-BR.md)
([EN](docs/GUIDE.md)): explica cada termo e coluna da tabela em linguagem
simples.

## Sumário

- [Features](#features)
- [Instalação](#instalação)
- [Arquitetura](#arquitetura)
- [Uso](#uso)
- [Notas](#notas)
- [Apoie o projeto](#apoie-o-projeto)
- [Licença](#licença)

## Features

**Descoberta & monitoramento** — varredura ARP ativa com hostname (DNS
reverso, depois fallback mDNS/NetBIOS), fabricante por OUI, palpite de classe,
RTT e TTL por dispositivo (com palpite de família de SO) e detalhe de portas
abertas (as sensíveis como SSH/SMB/RDP destacadas), tudo acompanhado
continuamente numa TUI ao vivo com alerta de dispositivo novo.

**Dashboard** — WiFi, banda em tempo real, speedtest e dispositivos numa única
tela ao vivo, com recomendação de canal WiFi por congestionamento de APs
vizinhos.

**Configuração de interface & rede** — inspeção de IP somente leitura em
qualquer plataforma; configuração estático/DHCP no Windows. Suporte a Linux
via ARP cru (`AF_PACKET`).

**Persistência & ferramentas** — apelidos de dispositivo que sobrevivem a
trocas de IP pelo DHCP, exportação JSON (`netwp scan --json`), e
auto-atualização (`netwp update` / `netwp version`).

## Instalação

Sem toolchain Go? Pegue um binário pronto na
[página de Releases](https://github.com/gsjonio/netwp/releases/latest)
(Windows e Linux amd64).

Requer Go 1.24+ para as opções abaixo.

### Instalação rápida (sem clonar)

O `go install` baixa o módulo, compila e coloca o binário em
`$(go env GOPATH)\bin`. Ponha essa pasta no PATH e chame como `netwp` de
qualquer terminal (o Windows resolve o `.exe` sozinho):

```powershell
go install github.com/gsjonio/netwp/cmd/netwp@latest
netwp
```

Prefira travar numa release específica em vez de `@latest` para builds
reprodutíveis, ex.: `go install github.com/gsjonio/netwp/cmd/netwp@v0.1.0`.

### Compilar a partir do código-fonte

Clone o repositório se quiser ler ou alterar o código, cross-compilar, ou
rodar a suíte de testes:

```powershell
git clone https://github.com/gsjonio/netwp.git
cd netwp
go build -o netwp.exe ./cmd/netwp
go test ./...
```

Para um binário menor, remova a tabela de símbolos e o DWARF
(cerca de 12 MB para 8.8 MB):

```powershell
go build -ldflags "-s -w" -o netwp.exe ./cmd/netwp
```

`go install -ldflags "-s -w" ./cmd/netwp` (rodando de dentro do repositório
clonado) faz a mesma coisa, direto para `$(go env GOPATH)\bin`.

### Privilégios por comando

| Comando | Windows | Linux |
| --- | --- | --- |
| `scan` · `monitor` · `dashboard` | sem privilégio | exige `CAP_NET_RAW` |
| `ports` · `speedtest` · `alias` · `version` · `update` | sem privilégio | sem privilégio |
| `iface` (só inspeção) | sem privilégio | sem privilégio |
| `iface static` / `iface dhcp` | exige terminal elevado | não implementado |

O Windows usa as APIs `SendARP`/`IcmpSendEcho` pro `scan`, então os comandos
de leitura nunca exigem admin. No Linux, conceda a capability do scanner ARP
cru uma vez em vez de rodar como root toda hora:

```bash
sudo setcap cap_net_raw+ep $(which netwp)
```

### Atualizando

Confira o que você tem com `netwp version`. Se tiver o toolchain Go
instalado (não importa como você instalou o netwp), o caminho mais fácil é:

```powershell
netwp update
```

É um wrapper fino em cima do `go install github.com/gsjonio/netwp/cmd/netwp@latest`
— o mesmo comando de baixo, só sem precisar redigitar o caminho do módulo.
Sobrescrever o binário em execução funciona até no Windows.

Fora isso, atualize do mesmo jeito que instalou:

- **Instalação rápida:** rode de novo `go install github.com/gsjonio/netwp/cmd/netwp@latest`
  (ou a tag específica que quiser). Sobrescreve o binário antigo.
- **Compilado do código-fonte:** `git pull` e recompile (`go build`/`go install`).
- **Binário pronto:** baixe o novo na
  [página de Releases](https://github.com/gsjonio/netwp/releases/latest) e
  substitua o arquivo antigo. Não tem mecanismo de auto-atualização por esse
  caminho.

## Arquitetura

Hexagonal (Ports & Adapters). O pacote `core` é domínio puro + casos de uso e
nunca importa código de SO/rede; os adapters implementam as portas e são
escolhidos em tempo de compilação por build tags.

```text
cmd/netwp        raiz de composição
internal/core    domínio + portas + casos de uso (puro)
internal/adapter arpscan · netinfo · oui (tocam o SO)
internal/tui     saída em tabela legível
```

## Uso

| Comando | O que faz |
| --- | --- |
| *(nenhum)* / `help` / `-h` / `--help` | Mostra a ajuda |
| `scan` / `scan --json` / `scan --diff` | Varredura única, com RTT por dispositivo; `--json` pra saída legível por máquina, `--diff` pra imprimir só o que mudou desde a última varredura |
| `monitor` / `monitor --alert-down=<taxa>` | TUI ao vivo: dispositivos entrando/saindo em tempo real (`q` sai); `--alert-down` avisa sobre queda na taxa de download, ex.: `--alert-down=50Mbps` |
| `dashboard` | Dashboard completo: WiFi + banda ao vivo + speedtest + dispositivos + um log de operações |
| `speedtest` | Teste de download/upload |
| `iface` | Inspeciona o IP da interface ativa |
| `iface static <ip>/<bits> <gw> [dns...]` | Define IP estático (pede confirmação) |
| `iface dhcp` | Volta para DHCP (pede confirmação) |
| `alias set <ip\|mac> <nome>` / `ls` / `rm <ip\|mac>` | Apelida um dispositivo / lista / remove |
| `class set <ip\|mac> <classe>` / `ls` / `rm <ip\|mac>` | Fixa a classe de um dispositivo quando o palpite erra (router/computer/mobile/media/printer/iot) |
| `watch add <ip\|mac>` / `ls` / `rm <ip\|mac>` | Alerta (destaque + bipe) quando um dispositivo sai durante o monitor/dashboard |
| `ports <ip>` | Portas abertas + RTT + TTL de um dispositivo |
| `wake <ip\|mac\|apelido>` | Envia um pacote Wake-on-LAN pra ligar um dispositivo |
| `doctor` | Diagnostica a conexão: interface, gateway, internet, DNS, Wi-Fi |
| `events [n]` | Mostra os últimos n eventos de entrada/saída (padrão 20) |
| `version` | Versão instalada |
| `update` | Atualiza pra última versão (precisa do Go) |
| `uninstall` | Remove os dados locais do netwp (pede confirmação); mostra como remover o binário |

```powershell
netwp scan --json | ConvertFrom-Json | Where-Object reachable
netwp alias set 192.168.1.20 "TV da Sala"
```

## Notas

Veja [SECURITY.md](SECURITY.md) pra segurança da varredura e como reportar
uma vulnerabilidade.

### Dados & armazenamento

- Os fabricantes vêm do registro IEEE MA-L completo, comprimido e embutido
  no binário (`internal/adapter/oui/data`). Atualize com o comando em
  `oui.go`.
- Os apelidos ficam em `<pasta-de-config-do-usuário>/netwp/aliases.json`,
  chaveados por MAC, então o apelido sobrevive a uma troca de IP pelo DHCP.
  Texto puro, seguro de editar à mão.
- `alias set <ip>` resolve o MAC pelo cache do último scan
  (`lastscan.json`), então apelidar logo após um scan é instantâneo. Passe
  um MAC no lugar do IP para não tocar a rede.

### Suporte por plataforma

- **Windows** é a plataforma primária, mais verificada: scan ARP via
  `SendARP`, ICMP via `IcmpSendEcho`, nenhum dos dois exige admin.
  `iface static`/`iface dhcp` exigem terminal elevado e sempre pedem um
  "yes" digitado; verificado de ponta a ponta em hardware real.
- **Linux** funciona mas é menos testado em campo: o scanner ARP cru
  (`AF_PACKET`) exige `CAP_NET_RAW` e já rodou de verdade num kernel Linux
  (WSL2), mas só contra a rede NAT padrão do WSL2, não uma LAN física
  completa. `iface static`/`dhcp` não está implementado no Linux. O CI
  compila e testa nativamente em Ubuntu a cada push.
- O painel de WiFi do dashboard aceita rótulos em inglês e português do
  `netsh wlan`; só os rótulos em português são verificados contra saída ao
  vivo.

### Como algumas coisas funcionam

Novo em termos como MAC, TTL, ou "dispositivo desconhecido"? O
[guia para iniciantes](docs/GUIDE.pt-BR.md) ([EN](docs/GUIDE.md)) explica o
que cada coisa na tela significa. Esta seção é trivia de implementação pra
quem já manja de redes.

- A resolução de hostname cai para mDNS/NetBIOS quando o DNS reverso não
  retorna nada; alguns dispositivos continuam sem nome. O mecanismo está
  no [CONTRIBUTING.md](CONTRIBUTING.md).
- RTT e TTL vêm do mesmo ICMP echo por dispositivo, então um com firewall
  (responde ARP mas não ICMP) aparece online sem nenhum dos dois.
- A sugestão de canal WiFi é uma contagem simples de congestionamento sobre
  os APs visíveis, não um planejador de RF.
- Uma máquina com mais de uma interface ativa (ex.: Ethernet e WiFi ao
  mesmo tempo) é reconhecida como "This device" em todas elas.
- O teste de banda usa o `speed.cloudflare.com` anycast; o `netwp
  speedtest` mostra qual edge respondeu.
- `netwp ports <ip>` sonda um único dispositivo diretamente em vez de um
  scan completo, sem histórico de portas entre execuções.
- O palpite de CLASS combina serviços mDNS anunciados (um Chromecast,
  impressora ou iPhone dizem o que são), depois ~29 portas sondadas, depois o
  fabricante. Quando ainda erra (um celular com MAC aleatório e sem portas
  abertas), fixe com `netwp class set <ip|mac> <classe>` — o pin manual
  sempre vence.
- `netwp wake` só liga um dispositivo que ficou com Wake-on-LAN habilitado
  (uma opção de BIOS/SO). Ele faz broadcast e não recebe resposta, então
  reporta "enviado", não "acordou". Um apelido ou IP em cache resolve mesmo
  com o alvo desligado.
- `netwp doctor` checa de cima pra baixo (interface → gateway → internet →
  DNS); o primeiro ✗ costuma ser a causa raiz e explica os de baixo.
- No `monitor`/`dashboard`, dois eventos tocam o bipe do terminal e destacam
  a linha do log: um dispositivo desconhecido entrando (sem apelido) e um
  dispositivo da lista do `netwp watch` saindo. O resto fica quieto.
- `netwp monitor --alert-down=<taxa>` (ex.: `50Mbps`) destaca a linha de
  banda quando o download cai abaixo desse limiar. Sem a flag, o monitor se
  comporta exatamente como antes.
- `netwp scan --diff` compara com a varredura anterior (identidade pelo
  MAC) e imprime só o que mudou, incluindo possíveis conflitos de IP/MAC.
- `netwp monitor`/`dashboard` gravam cada entrada/saída em
  `<pasta-de-config-do-usuário>/netwp/events.jsonl`; `netwp events [n]`
  mostra esse histórico.
- O painel DEVICES do dashboard mostra um resumo por classe do que está
  online (ex.: "2 Media · 1 Router"), sem contar "This device" e hosts não
  classificados.
- O painel LOG do dashboard (embaixo) mostra o que ele mesmo está fazendo —
  scans começando e terminando, speedtests, e mudanças de estado de
  internet/Wi-Fi. Num terminal curto ele encolhe, depois some, pra a tabela
  de dispositivos e o rodapé terem prioridade. (Diferente do painel ACTIVITY,
  que lista entradas/saídas de dispositivos.)

Quer contribuir? Veja [CONTRIBUTING.md](CONTRIBUTING.md). Este projeto
segue o [Código de Conduta](CODE_OF_CONDUCT.md).

## Apoie o projeto

O netwp é livre e de código aberto. Se ele te economiza tempo, você pode
apoiar o desenvolvimento com um café — obrigado! ☕

[![Buy Me a Coffee](https://img.shields.io/badge/Buy_Me_a_Coffee-gugamenezes-FFDD00?style=for-the-badge&logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/gugamenezes)

## Licença

[MIT](LICENSE).
