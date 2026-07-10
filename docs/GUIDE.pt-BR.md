# Guia para iniciantes: entendendo sua rede com o netwp

🇺🇸 [English](GUIDE.md)

Este guia explica, em linguagem simples, os termos e as informações que o
netwp mostra na tela. Não é preciso saber nada de redes de antemão: comece
por aqui e depois volte para o [README](../README.pt-BR.md) para ver os
comandos com detalhe.

## Sumário

- [Para quem é este guia](#para-quem-é-este-guia)
- [Conceitos básicos de rede](#conceitos-básicos-de-rede)
- [O que cada comando do netwp faz](#o-que-cada-comando-do-netwp-faz)
- [O que significa cada coluna da tabela](#o-que-significa-cada-coluna-da-tabela)
- [Sinais de alerta: segurança e rede ruim](#sinais-de-alerta-segurança-e-rede-ruim)
- [Perguntas frequentes](#perguntas-frequentes)

## Para quem é este guia

Você nunca configurou uma rede, nunca ouviu falar de "endereço MAC" ou
"TTL", e quer entender o que o netwp está te mostrando na tela. É para
você. Se você já manja de redes, o [README](../README.pt-BR.md) sozinho já
é suficiente.

## Conceitos básicos de rede

Antes das telas do netwp, alguns conceitos que aparecem o tempo todo:

- **Rede local**: os aparelhos conectados no mesmo Wi-Fi ou cabo dentro de
  casa (ou escritório). O netwp só enxerga o que está dentro dessa rede,
  não a internet inteira.
- **Endereço IP**: um número tipo `192.168.1.20` que identifica um
  aparelho dentro da rede local, meio como o número de uma casa numa rua.
  O roteador distribui esses números automaticamente (isso se chama
  **DHCP**), e por isso o IP de um aparelho às vezes muda sozinho.
- **Sub-rede**: o "bairro" inteiro de IPs, escrito como `192.168.1.0/24`,
  ou seja "todo aparelho de 192.168.1.0 até 192.168.1.255". É essa faixa
  que o netwp varre num `scan`.
- **Roteador / gateway**: o aparelho que liga sua rede local à internet. No
  netwp ele aparece com a classe "Router".
- **Endereço MAC**: um número de série gravado de fábrica na placa de rede
  de cada aparelho, tipo `aa:bb:cc:dd:ee:ff`. Ao contrário do IP, o MAC não
  muda. É por isso que o netwp usa o MAC (não o IP) para reconhecer "ainda
  é o mesmo aparelho" mesmo depois que o DHCP trocou o IP dele.
- **ARP**: o "quem está aí?" que o netwp manda pela rede local para
  descobrir quais IPs têm um aparelho respondendo, e qual o MAC de cada
  um. É assim que o `scan` funciona.

## O que cada comando do netwp faz

- **`netwp scan`**: tira uma foto única da rede agora, mostrando todo
  mundo que respondeu.
- **`netwp monitor`**: fica de olho na rede ao vivo, avisando quando um
  aparelho entra ou sai.
- **`netwp dashboard`**: a mesma coisa do monitor, só que com Wi-Fi,
  velocidade de internet e mais informação tudo numa tela só.
- **`netwp ports <ip>`**: olha só um aparelho de perto, mostrando quais
  portas (serviços) ele deixa abertas.
- **`netwp events`**: mostra o histórico de quem entrou e saiu da rede.

## O que significa cada coluna da tabela

| Coluna | O que é |
| --- | --- |
| **STATUS** (●/○) | Bolinha verde acesa: está online agora. Cinza apagada: já foi visto antes, mas não respondeu na última varredura. |
| **IP** | O endereço do aparelho na rede local agora. Pode mudar com o tempo (DHCP). |
| **ALIAS** | Um apelido que você mesmo define com `netwp alias set`, para não precisar decorar o endereço MAC. |
| **RTT** | Quanto tempo (em milissegundos) uma "batidinha" (ping) leva para ir e voltar até o aparelho. Quanto menor, melhor: verde é rápido, sem cor é razoável, vermelho é lento para os padrões de rede local (ainda assim rápido perto dos padrões de internet). |
| **TTL** | Uma pista sobre o sistema operacional do aparelho, tipo "64 (Linux)" ou "128 (Windows)". Vem de graça na mesma resposta do RTT. É só um palpite, não uma certeza. |
| **CLASS** | Um palpite de que tipo de aparelho é (Router, Computer, Mobile, Media, Printer, IoT). O netwp adivinha pelas portas abertas e pelo fabricante; às vezes erra ou fica "Unknown". |
| **MAC** | O endereço MAC explicado acima: a identidade permanente do aparelho. |
| **HOSTNAME** | O nome que o próprio aparelho anuncia na rede (nem todo aparelho anuncia um). |
| **VENDOR** | O fabricante da placa de rede (Apple, Samsung, TP-Link...), descoberto pelos primeiros números do MAC. |
| **PORTS** | Quais "portas" (serviços de rede) o aparelho deixa abertas. Aparece em vermelho quando é uma porta sensível (veja a seção de alertas). |
| **LAST SEEN** | Há quanto tempo o aparelho foi visto pela última vez, quando está offline. |

## Sinais de alerta: segurança e rede ruim

Nem tudo que aparece destacado é um problema, mas vale uma segunda olhada:

- **Um aparelho desconhecido entrou na rede** (aviso "⚠ ... joined
  (unknown)" no log de atividade do `monitor`): é um MAC que nunca teve
  apelido definido. Pode ser só o celular de uma visita, um aparelho novo
  que você comprou, ou alguém que não deveria estar na sua rede. Se for
  legítimo, dê um apelido com `netwp alias set` para ele parar de aparecer
  como desconhecido.
- **O mesmo IP passa a responder com um MAC diferente** (aviso do `netwp
  scan --diff`): isso é estranho. Pode ser só uma coincidência de troca de
  aparelho, mas também é a assinatura clássica de um ataque chamado ARP
  spoofing, onde alguém na rede finge ser outro aparelho (às vezes até o
  roteador) para interceptar tráfego. Vale confirmar de onde veio essa
  mudança.
- **Um MAC aparece em mais de um IP na mesma varredura**: também incomum
  e digno de atenção, pelo mesmo motivo acima.
- **Portas sensíveis abertas** (coluna PORTS em vermelho: SSH/22, SMB/445,
  RDP/3389): são portas de acesso remoto. Numa rede doméstica, quase nunca
  é intencional deixar isso aberto. Confira com `netwp ports <ip>` se faz
  sentido para aquele aparelho.
- **RTT vermelho / rede lenta**: o aparelho está demorando mais de 100ms
  para responder dentro da própria rede local. Pode ser Wi-Fi fraco,
  aparelho sobrecarregado, ou só uma medição ruim naquele instante.
- **Alerta de banda baixa** (`netwp monitor --alert-down`): a velocidade
  real de download caiu abaixo do valor que você configurou. Pode ser o
  provedor, o roteador, ou outro aparelho consumindo a banda toda.

Para saber mais sobre uso responsável do netwp numa rede que não é sua,
veja o [SECURITY.md](../SECURITY.md).

## Perguntas frequentes

**O netwp manda algum dado meu para algum lugar?**
Não. Os dados ficam só no seu computador (`aliases.json`, `lastscan.json`,
`events.jsonl`), fora o próprio tráfego de rede necessário para fazer a
varredura (ARP, ping, teste de velocidade). Veja
[SECURITY.md](../SECURITY.md) para detalhes.

**Posso usar o netwp para "invadir" a rede de outra pessoa?**
Não use o netwp em redes que não são suas ou que você não tem autorização
explícita para escanear. Em redes corporativas isso pode até violar
políticas de uso. Veja [SECURITY.md](../SECURITY.md).

**Por que um aparelho aparece como "Unknown" na coluna CLASS?**
Porque o netwp não conseguiu nenhuma pista suficiente (nem porta aberta
reconhecida, nem fabricante característico) para arriscar um palpite. É
melhor "Unknown" do que um palpite errado.

**Por que o IP de um aparelho muda de vez em quando?**
Isso é normal: o roteador reatribui IPs por DHCP de tempos em tempos. O
netwp usa o MAC (que não muda) para continuar reconhecendo que é o mesmo
aparelho, então o apelido que você deu continua valendo.
