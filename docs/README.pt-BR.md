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
.\netwp.exe speedtest # teste de download/upload
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
```

## Notas

- Os fabricantes vêm do registro IEEE MA-L completo, comprimido e embutido no
  binário (`internal/adapter/oui/data`). Atualize com o comando em `oui.go`.
- Varredura ativa pode ser vista como intrusiva em redes gerenciadas/corporativas.
  Escaneie apenas redes suas ou autorizadas.
- O teste de banda usa o endpoint público `speed.cloudflare.com`: sem chave
  de API, sem servidor próprio.

## Licença

[MIT](../LICENSE).
