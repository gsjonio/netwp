# netwp

🇺🇸 [English](README.md)

**netwp** = *Internet / Rede Well Played* ("a rede, bem jogada").

Gerenciador de rede via terminal escrito em Go: descoberta ativa de dispositivos
na rede local (ARP), monitoramento ao vivo, dashboard completo, teste de banda e
inspeção de interface. Windows primeiro, portável para Linux.

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

## Compilar e executar

Requer Go 1.22+.

```powershell
go build -o netwp.exe ./cmd/netwp
.\netwp.exe            # varredura única (padrão), com RTT por dispositivo
.\netwp.exe scan --json # mesma varredura, saída JSON no stdout
.\netwp.exe monitor   # TUI ao vivo: dispositivos entrando/saindo em tempo real (q sai)
.\netwp.exe dashboard # dashboard completo: wifi + banda ao vivo + speedtest + dispositivos
.\netwp.exe speedtest # teste de download/upload
.\netwp.exe iface     # config de IP da interface ativa
.\netwp.exe iface static 192.168.1.50/24 192.168.1.1 8.8.8.8  # define IP estático (pede confirmação)
.\netwp.exe iface dhcp                                        # volta para DHCP (pede confirmação)
.\netwp.exe alias set 192.168.1.20 "TV da Sala"  # apelida um dispositivo (por IP ou MAC)
.\netwp.exe alias ls                             # lista os apelidos
.\netwp.exe alias rm 192.168.1.20                # remove um apelido
.\netwp.exe ports 192.168.1.20                   # portas abertas + RTT de um dispositivo
go test ./...
```

Para um binário menor, remova a tabela de símbolos e o DWARF
(cerca de 12 MB para 8.8 MB):

```powershell
go build -ldflags "-s -w" -o netwp.exe ./cmd/netwp
```

O scanner Windows usa a API `SendARP`: **não exige admin nem Npcap**.

### Instalar como `netwp`

O `go install` coloca o binário em `$(go env GOPATH)\bin`. Com essa pasta no
PATH, você chama como `netwp` de qualquer terminal (o Windows resolve o `.exe`
sozinho):

```powershell
go install -ldflags "-s -w" ./cmd/netwp   # -ldflags opcional, só para um binário menor
netwp             # varredura
netwp scan --json # varredura, saída JSON
netwp monitor     # monitor ao vivo
netwp dashboard   # dashboard completo
netwp speedtest   # teste de banda
netwp iface       # config de IP da interface
netwp alias set 192.168.1.20 "TV da Sala"  # apelida um dispositivo
netwp ports 192.168.1.20                   # portas abertas + RTT de um dispositivo
```

## Notas

- Os fabricantes vêm do registro IEEE MA-L completo, comprimido e embutido no
  binário (`internal/adapter/oui/data`). Atualize com o comando em `oui.go`.
- Varredura ativa pode ser vista como intrusiva em redes gerenciadas/corporativas.
  Escaneie apenas redes suas ou autorizadas.
- Os apelidos ficam em JSON em `<pasta-de-config-do-usuário>/netwp/aliases.json`,
  chaveados por MAC, então o apelido persiste mesmo quando o DHCP troca o IP do
  aparelho. O arquivo é texto puro e pode ser editado à mão.
- `alias set <ip>` resolve o MAC pelo cache do último scan (`lastscan.json`) e
  só escaneia de novo se der miss, então apelidar logo após um scan é instantâneo.
  Passe um MAC no lugar do IP para não tocar a rede.
- O teste de banda usa o endpoint público `speed.cloudflare.com`: sem chave
  de API, sem servidor próprio.
- `iface static`/`iface dhcp` chamam o `netsh` e exigem terminal
  administrador no Windows. Sempre pedem que você digite "yes" antes de
  mexer na configuração de verdade; não existe flag `--yes` para pular isso.
  Ainda não implementado no Linux.
- O painel de WiFi do dashboard lê o `netsh wlan` (aceita rótulos em inglês e
  português). Verificado em hardware real nos dois estados: desconectado e
  conectado (SSID/BSSID/canal/sinal/taxa Rx-Tx da própria conexão). Os
  rótulos em inglês ainda são só fixture, testá-los exige uma instalação
  Windows em locale inglês. Numa máquina só-cabo o painel mostra
  "disconnected".
- O scanner Linux (ARP cru via `AF_PACKET`) exige `CAP_NET_RAW` (root, ou
  `setcap cap_net_raw+ep` no binário). Foi escrito e cross-compilado
  (`GOOS=linux`) numa máquina Windows e ainda não rodou em hardware Linux
  de verdade.
- O RTT vem de um ICMP echo real por dispositivo: `IcmpSendEcho` (iphlpapi) no
  Windows, sem exigir admin; o binário `ping` do sistema nas outras
  plataformas. Um dispositivo que responde ARP mas não ICMP (com firewall)
  aparece online sem RTT.
- A sugestão de canal WiFi é uma contagem simples de congestionamento sobre os
  APs visíveis, não um planejador de RF: não considera intensidade de sinal,
  restrições de DFS nem regras regulatórias.
- Uma entrada só é marcada como "unknown" no log de atividade quando o MAC não
  tem apelido definido. Apelidar um dispositivo o marca como reconhecido nas
  próximas entradas.
- Quando o DNS reverso não retorna nada, a resolução de hostname cai para uma
  consulta reversa de mDNS e uma consulta NetBIOS NBSTAT, disputadas entre si
  com orçamento de 400ms cada. Nenhuma das duas é garantida: um dispositivo
  sem responder Bonjour/Avahi e sem suporte a NetBIOS (muitos celulares, a
  maioria das máquinas Linux sem avahi) continua sem hostname. Verificado em
  hardware real na rede do autor, incluindo um dispositivo cujo nome mDNS
  configurado é literalmente "none".
- `netwp ports <ip>` sonda um único dispositivo diretamente em vez de rodar
  uma varredura completa: as mesmas portas TCP conhecidas usadas na
  classificação, reportadas individualmente com nome, mais um RTT ICMP
  fresco. Não há histórico de portas entre execuções, só o estado atual.

## Licença

[MIT](LICENSE).
