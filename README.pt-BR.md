# netwp

🇧🇷 Português · 🇺🇸 [English](README.md)

[![CI](https://github.com/gsjonio/netwp/actions/workflows/ci.yml/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql)
[![Dependabot](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph)
[![Go version](https://img.shields.io/github/go-mod/go-version/gsjonio/netwp)](go.mod)
[![Release](https://img.shields.io/github/v/release/gsjonio/netwp)](https://github.com/gsjonio/netwp/releases/latest)
[![License: MIT](https://img.shields.io/github/license/gsjonio/netwp)](LICENSE)

**netwp** = *Internet / Rede Well Played* ("a rede, bem jogada").

Gerenciador de rede via terminal escrito em Go: descoberta ativa de dispositivos
na rede local (ARP), monitoramento ao vivo, dashboard completo, teste de banda e
inspeção de interface. Windows primeiro, portável para Linux.

## Sumário

- [Status](#status)
- [Arquitetura](#arquitetura)
- [Instalação](#instalação)
- [Uso](#uso)
- [Notas](#notas)
- [Licença](#licença)

## Status

- [x] Núcleo de descoberta (ARP scan, hostname, fabricante por OUI, palpite de classe)
- [x] Monitoramento contínuo (entrada/saída), TUI ao vivo
- [x] Teste de banda
- [x] Inspeção de IP da interface (somente leitura)
- [x] Configuração de IP da interface (estático/DHCP, só Windows)
- [x] Adapter Linux (ARP cru via AF_PACKET, gateway, DNS)
- [x] Apelidos persistentes de dispositivos (chaveados por MAC)
- [x] Dashboard ao vivo (WiFi, banda em tempo real, speedtest, dispositivos)
- [x] Latência por dispositivo (RTT) e latência da internet, ICMP nativo (sem admin)
- [x] Recomendação de canal WiFi por congestionamento de APs vizinhos
- [x] Alerta de dispositivo novo (entrada de MAC não reconhecido no monitor/dashboard)
- [x] Exportação JSON (`netwp scan --json`)
- [x] Fallback de hostname via mDNS/NetBIOS quando o DNS reverso não retorna nada
- [x] Detalhe de portas por dispositivo (`netwp ports <ip>`)
- [x] Portas abertas na tabela, com as sensíveis (SSH/SMB/RDP) destacadas
- [x] Auto-atualização (`netwp update`) e versão instalada (`netwp version`)

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

## Uso

| Comando | O que faz |
| --- | --- |
| *(nenhum)* / `help` / `-h` / `--help` | Mostra a ajuda |
| `scan` / `scan --json` | Varredura única, com RTT por dispositivo; `--json` pra saída legível por máquina |
| `monitor` | TUI ao vivo: dispositivos entrando/saindo em tempo real (`q` sai) |
| `dashboard` | Dashboard completo: WiFi + banda ao vivo + speedtest + dispositivos |
| `speedtest` | Teste de download/upload |
| `iface` | Inspeciona o IP da interface ativa |
| `iface static <ip>/<bits> <gw> [dns...]` | Define IP estático (pede confirmação) |
| `iface dhcp` | Volta para DHCP (pede confirmação) |
| `alias set <ip\|mac> <nome>` / `ls` / `rm <ip\|mac>` | Apelida um dispositivo / lista / remove |
| `ports <ip>` | Portas abertas + RTT de um dispositivo |
| `version` | Versão instalada |
| `update` | Atualiza pra última versão (precisa do Go) |

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

- A resolução de hostname tenta DNS reverso primeiro, depois cai para mDNS
  e NetBIOS (400ms cada). Nenhum dos fallbacks é garantido: um dispositivo
  sem responder Bonjour/Avahi nem suporte a NetBIOS simplesmente não mostra
  nome.
- O RTT é um ICMP echo real por dispositivo; um que responde ARP mas não
  ICMP (com firewall) aparece online sem RTT.
- A sugestão de canal WiFi é uma contagem simples de congestionamento sobre
  os APs visíveis, não um planejador de RF: sem sinal, DFS ou regras
  regulatórias.
- Uma máquina com mais de uma interface ativa (ex.: Ethernet e WiFi ao
  mesmo tempo) é reconhecida como "This device" em todas elas, não só na
  interface usada pra escolher a sub-rede do scan.
- O teste de banda usa o `speed.cloudflare.com` anycast, que roteia
  automaticamente pro edge mais próximo entre os ~300 da Cloudflare; o
  `netwp speedtest` mostra qual respondeu (ex.: "via Cloudflare edge: GRU").
- `netwp ports <ip>` sonda um único dispositivo diretamente (mesmas portas
  usadas na classificação, reportadas individualmente) em vez de um scan
  completo. Sem histórico de portas entre execuções, só o estado atual.
- A coluna PORTS da tabela reaproveita as portas que a sonda de classificação
  já coleta, então não custa varredura extra. SSH (22), SMB (445) e RDP
  (3389) aparecem em vermelho: expostas numa rede doméstica, normalmente não
  são intencionais. Os nomes das portas ficam um nível abaixo, no
  `netwp ports <ip>`.
- O painel DEVICES do dashboard mostra um resumo por classe do que está online
  (ex.: "2 Media · 1 Router"). "This device" e hosts não classificados ficam
  de fora, já que nenhum dos dois diz nada sobre a rede.
- Uma entrada só é marcada como "unknown" no log de atividade quando o MAC
  não tem apelido definido.

Quer contribuir? Veja [CONTRIBUTING.md](CONTRIBUTING.md). Este projeto
segue o [Código de Conduta](CODE_OF_CONDUCT.md).

## Licença

[MIT](LICENSE).
