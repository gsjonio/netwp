package wifi

import "testing"

// Real capture from `netsh wlan show networks mode=bssid` (pt-BR), trimmed.
const networksPT = `Nome da interface: Wi-Fi 2
Há 2 redes visíveis no momento.

SSID 1 : GUSTAVO -2.4G
    Tipo de rede            : Infraestrutura
    BSSID 1                 : 86:0b:bb:1c:9e:5f
         Sinal             : 40%
         Tipo de Rádio         : 802.11ac
         Banda               : 5 GHz
         Canal            : 44
SSID 2 : GUSTAVO -5G
    Tipo de rede            : Infraestrutura
    BSSID 1                 : 8a:0b:bb:1c:9e:60
         Sinal             : 72%
         Canal            : 44
`

func TestParseNetworks(t *testing.T) {
	aps := parseNetworks(networksPT)
	if len(aps) != 2 {
		t.Fatalf("got %d APs, want 2: %+v", len(aps), aps)
	}
	if aps[0].SSID != "GUSTAVO -2.4G" || aps[0].Channel != 44 || aps[0].SignalPercent != 40 {
		t.Errorf("AP0 = %+v, want {GUSTAVO -2.4G 44 40}", aps[0])
	}
	if aps[1].SSID != "GUSTAVO -5G" || aps[1].Channel != 44 || aps[1].SignalPercent != 72 {
		t.Errorf("AP1 = %+v", aps[1])
	}
}

// Representative connected `show interfaces` block, Portuguese labels.
const interfacesPT = `    Nome                   : Wi-Fi 2
    SSID                   : Tam Oi Fibra 5G
    BSSID                  : 84:3e:92:7a:6b:70
    Estado                 : Conectado
    Tipo de rádio          : 802.11ax
    Banda                  : 5 GHz
    Canal                  : 149
    Taxa de recebimento (Mbps)  : 270
    Taxa de transmissão (Mbps)  : 300
    Sinal                  : 47%
`

func TestParseInterfacesPT(t *testing.T) {
	w := parseInterfaces(interfacesPT)
	if !w.Connected {
		t.Fatal("expected Connected=true")
	}
	if w.SSID != "Tam Oi Fibra 5G" {
		t.Errorf("SSID = %q", w.SSID)
	}
	if w.BSSID != "84:3e:92:7a:6b:70" {
		t.Errorf("BSSID = %q (colons in value must survive)", w.BSSID)
	}
	if w.Channel != 149 || w.SignalPercent != 47 {
		t.Errorf("channel/signal = %d/%d, want 149/47", w.Channel, w.SignalPercent)
	}
	if w.RxRateMbps != 270 || w.TxRateMbps != 300 {
		t.Errorf("rx/tx = %d/%d, want 270/300", w.RxRateMbps, w.TxRateMbps)
	}
	if w.Band != "5 GHz" {
		t.Errorf("band = %q", w.Band)
	}
}

// Same fields, English labels: the parser must handle both locales.
const interfacesEN = `    Name                   : Wi-Fi
    SSID                   : HomeNet
    BSSID                  : aa:bb:cc:dd:ee:ff
    State                  : connected
    Radio type             : 802.11ax
    Channel                : 36
    Receive rate (Mbps)    : 866
    Transmit rate (Mbps)   : 866
    Signal                 : 90%
`

func TestParseInterfacesEN(t *testing.T) {
	w := parseInterfaces(interfacesEN)
	if !w.Connected || w.SSID != "HomeNet" || w.Channel != 36 || w.SignalPercent != 90 {
		t.Errorf("EN parse failed: %+v", w)
	}
}

func TestParseInterfacesDisconnected(t *testing.T) {
	const disc = `    Nome     : Wi-Fi 2
    Estado   : Desconectado
`
	if w := parseInterfaces(disc); w.Connected {
		t.Errorf("disconnected interface should have Connected=false, got %+v", w)
	}
}
