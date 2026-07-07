# netwp

A terminal network manager written in Go. Active local-network device
discovery (ARP), with monitoring, bandwidth testing and interface inspection
planned. Windows-first, portable to Linux.

*Gerenciador de rede via terminal em Go. Descoberta ativa de dispositivos na
rede local (ARP), com monitoramento, teste de banda e inspeção de interface
planejados. Windows primeiro, portável para Linux.*

## Status

- [x] Device discovery core (ARP scan, hostname, vendor by OUI) — *núcleo de descoberta*
- [ ] Continuous monitoring (join/leave) — *monitoramento contínuo*
- [ ] Bandwidth test — *teste de banda*
- [ ] Interface IP inspect/configure — *inspeção/configuração de IP*
- [ ] Linux adapter (AF_PACKET raw ARP) — *adapter Linux*

## Architecture / Arquitetura

Hexagonal (Ports & Adapters). The `core` package is pure domain + use cases and
never imports OS/network code; adapters implement its ports and are selected at
build time via Go build tags.

*Hexagonal (Ports & Adapters). O pacote `core` é domínio puro + casos de uso e
nunca importa código de SO/rede; os adapters implementam as portas e são
escolhidos em tempo de compilação por build tags.*

```text
cmd/netwp        composition root / raiz de composição
internal/core    domain + ports + use cases (pure)
internal/adapter arpscan · netinfo · oui (touch the OS)
internal/tui     legible table output / saída em tabela
```

## Build & Run / Compilar e executar

Requires Go 1.22+. / Requer Go 1.22+.

```powershell
go build -o netwp.exe ./cmd/netwp
.\netwp.exe
go test ./...
```

The Windows scanner uses the `SendARP` API: **no admin rights and no Npcap
required**. / O scanner Windows usa a API `SendARP`: **não exige admin nem Npcap**.

## Notes / Notas

- Vendor names come from the full IEEE MA-L registry, gzipped and embedded in
  the binary (`internal/adapter/oui/data`). Refresh it with the command in
  `oui.go`. / *Fabricantes vêm do registro IEEE MA-L completo, comprimido e
  embutido no binário. Atualize com o comando em `oui.go`.*
- Active scanning may be flagged as intrusive on managed/corporate networks.
  Only scan networks you own or are authorized to. / *Varredura ativa pode ser
  vista como intrusiva em redes gerenciadas. Escaneie apenas redes suas ou
  autorizadas.*
