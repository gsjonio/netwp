# netwp — Português

**netwp** = *Internet / Rede Well Played* — "a rede, bem jogada".

Gerenciador de rede via terminal escrito em Go. Descoberta ativa de dispositivos
na rede local (ARP), com monitoramento, teste de banda e inspeção de interface
planejados. Windows primeiro, portável para Linux.

🇬🇧 [English version](README.en.md)

## Status

- [x] Núcleo de descoberta (ARP scan, hostname, fabricante por OUI)
- [x] Monitoramento contínuo (entrada/saída), TUI ao vivo
- [ ] Teste de banda
- [ ] Inspeção/configuração de IP da interface
- [ ] Adapter Linux (ARP cru via AF_PACKET)

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
go test ./...
```

O scanner Windows usa a API `SendARP`: **não exige admin nem Npcap**.

## Notas

- Os fabricantes vêm do registro IEEE MA-L completo, comprimido e embutido no
  binário (`internal/adapter/oui/data`). Atualize com o comando em `oui.go`.
- Varredura ativa pode ser vista como intrusiva em redes gerenciadas/corporativas.
  Escaneie apenas redes suas ou autorizadas.

## Licença

[MIT](../LICENSE).
