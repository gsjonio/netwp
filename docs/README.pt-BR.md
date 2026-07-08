# netwp (Português)

**netwp** = *Internet / Rede Well Played* ("a rede, bem jogada").

Gerenciador de rede via terminal escrito em Go. Faz descoberta ativa de
dispositivos na rede local (ARP); monitoramento, teste de banda e inspeção de
interface estão planejados. Windows primeiro, portável para Linux.

🇬🇧 [English version](README.en.md)

## Status

- [x] Núcleo de descoberta (ARP scan, hostname, fabricante por OUI, palpite de classe)
- [x] Monitoramento contínuo (entrada/saída), TUI ao vivo
- [x] Teste de banda
- [x] Inspeção de IP da interface (somente leitura)
- [x] Configuração de IP da interface (estático/DHCP, só Windows)
- [x] Adapter Linux (ARP cru via AF_PACKET, gateway, DNS)

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
.\netwp.exe            # varredura única (padrão)
.\netwp.exe monitor   # TUI ao vivo: dispositivos entrando/saindo em tempo real (q sai)
.\netwp.exe speedtest # teste de download/upload
.\netwp.exe iface     # config de IP da interface ativa
.\netwp.exe iface static 192.168.1.50/24 192.168.1.1 8.8.8.8  # define IP estático (pede confirmação)
.\netwp.exe iface dhcp                                        # volta para DHCP (pede confirmação)
go test ./...
```

O scanner Windows usa a API `SendARP`: **não exige admin nem Npcap**.

### Instalar como `netwp`

O `go install` coloca o binário em `$(go env GOPATH)\bin`. Com essa pasta no
PATH, você chama como `netwp` de qualquer terminal (o Windows resolve o `.exe`
sozinho):

```powershell
go install ./cmd/netwp
netwp            # varredura
netwp monitor    # monitor ao vivo
netwp speedtest  # teste de banda
netwp iface      # config de IP da interface
```

## Notas

- Os fabricantes vêm do registro IEEE MA-L completo, comprimido e embutido no
  binário (`internal/adapter/oui/data`). Atualize com o comando em `oui.go`.
- Varredura ativa pode ser vista como intrusiva em redes gerenciadas/corporativas.
  Escaneie apenas redes suas ou autorizadas.
- O teste de banda usa o endpoint público `speed.cloudflare.com`: sem chave
  de API, sem servidor próprio.
- `iface static`/`iface dhcp` chamam o `netsh` e exigem terminal
  administrador no Windows. Sempre pedem que você digite "yes" antes de
  mexer na configuração de verdade; não existe flag `--yes` para pular isso.
  Ainda não implementado no Linux.
- O scanner Linux (ARP cru via `AF_PACKET`) exige `CAP_NET_RAW` (root, ou
  `setcap cap_net_raw+ep` no binário). Foi escrito e cross-compilado
  (`GOOS=linux`) numa máquina Windows e ainda não rodou em hardware Linux
  de verdade.

## Licença

[MIT](../LICENSE).
