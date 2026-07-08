# netwp

**netwp** = *Internet / Rede Well Played*.

A terminal network manager written in Go: active local-network device
discovery (ARP), monitoring, bandwidth testing, and interface inspection.
Windows-first, portable to Linux.

*Gerenciador de rede via terminal em Go: descoberta ativa de dispositivos,
monitoramento, teste de banda e inspeção de interface. Windows primeiro,
portável para Linux.*

## Documentation / Documentação

- 🇬🇧 [English](docs/README.en.md)
- 🇧🇷 [Português](docs/README.pt-BR.md)

## Install / Instalação

Requires Go 1.22+ / Requer Go 1.22+.

Run from source / Rodar do código:

```powershell
go build -o netwp.exe ./cmd/netwp
.\netwp.exe
```

Install as a global `netwp` command / Instalar como comando global `netwp`:

```powershell
go install ./cmd/netwp
```

`go install` drops the binary in `$(go env GOPATH)\bin`. With that folder on
your PATH, call it as `netwp` from any terminal. Add `-ldflags "-s -w"` for a
smaller binary (about 12 MB down to 8.8 MB).

*O `go install` coloca o binário em `$(go env GOPATH)\bin`. Com essa pasta no
PATH, chame como `netwp` de qualquer terminal. Use `-ldflags "-s -w"` para um
binário menor (cerca de 12 MB para 8.8 MB).*

## Commands / Comandos

```powershell
netwp                 # scan the local network / varre a rede local
netwp monitor         # live join/leave TUI / TUI ao vivo de entrada/saída
netwp speedtest       # download/upload throughput / teste de banda
netwp iface           # active interface IP config / config de IP da interface
netwp iface static <ip>/<bits> <gateway> [dns...]   # set a static address (admin)
netwp iface dhcp      # switch back to DHCP (admin) / volta para DHCP (admin)
```

The Windows scanner uses `SendARP`: no admin rights and no Npcap required.
`iface static` / `iface dhcp` change the real network config and ask for
confirmation first. Full details in the language docs above.

*O scanner Windows usa `SendARP`: sem admin e sem Npcap. `iface static` /
`iface dhcp` alteram a config de rede real e pedem confirmação antes. Detalhes
completos nas docs por idioma acima.*

## License

[MIT](LICENSE) © 2026 Gustavo Oliveira
