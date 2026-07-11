# Beginner's guide: understanding your network with netwp

🇧🇷 [Português](GUIDE.pt-BR.md)

This guide explains, in plain language, the terms and information netwp
shows on screen. No prior networking knowledge needed: start here, then go
back to the [README](../README.md) for command details.

## Table of Contents

- [Who this guide is for](#who-this-guide-is-for)
- [Basic networking concepts](#basic-networking-concepts)
- [What each netwp command does](#what-each-netwp-command-does)
- [What each table column means](#what-each-table-column-means)
- [Warning signs: security and a bad network](#warning-signs-security-and-a-bad-network)
- [Frequently asked questions](#frequently-asked-questions)

## Who this guide is for

You've never set up a network, never heard of a "MAC address" or "TTL",
and want to understand what netwp is showing you. This is for you. If you
already know networking, the [README](../README.md) alone is enough.

## Basic networking concepts

Before the netwp screens make sense, a few concepts that come up
constantly:

- **Local network**: the devices connected to the same Wi-Fi or cable
  inside a home or office. netwp only sees what's inside this network, not
  the whole internet.
- **IP address**: a number like `192.168.1.20` that identifies a device
  inside the local network, similar to a house number on a street. The
  router hands these out automatically (this is called **DHCP**), which is
  why a device's IP sometimes changes on its own.
- **Subnet**: the whole "neighborhood" of IPs, written as `192.168.1.0/24`,
  meaning "every device from 192.168.1.0 to 192.168.1.255". This is the
  range netwp sweeps during a `scan`.
- **Router / gateway**: the device that connects your local network to the
  internet. netwp shows it with the "Router" class.
- **MAC address**: a serial number burned into each device's network card
  at the factory, like `aa:bb:cc:dd:ee:ff`. Unlike the IP, the MAC never
  changes. That's why netwp uses the MAC (not the IP) to recognize "this is
  still the same device" even after DHCP has changed its IP.
- **ARP**: the "who's out there?" netwp sends across the local network to
  find out which IPs have a device answering, and what MAC each one has.
  This is how `scan` works.

## What each netwp command does

- **`netwp scan`**: takes a single snapshot of the network right now,
  showing everyone who answered.
- **`netwp monitor`**: watches the network live, alerting when a device
  joins or leaves.
- **`netwp dashboard`**: the same as monitor, plus Wi-Fi, internet speed,
  and more, all on one screen.
- **`netwp ports <ip>`**: takes a close look at a single device, showing
  which ports (services) it has open.
- **`netwp events`**: shows the history of who joined and left the
  network.

## What each table column means

| Column | What it is |
| --- | --- |
| **STATUS** (●/○) | A lit green dot: online right now. A dim gray one: seen before, but didn't answer the last scan. |
| **IP** | The device's current address on the local network. Can change over time (DHCP). |
| **ALIAS** | A nickname you set yourself with `netwp alias set`, so you don't have to memorize a MAC address. |
| **RTT** | How long (in milliseconds) a "ping" takes to go out and come back from the device. Lower is better: green is fast, uncolored is fine, red is slow by local-network standards (still fast by internet standards, though). |
| **TTL** | A hint about the device's operating system, like "64 (Linux)" or "128 (Windows)". Comes for free from the same reply as RTT. It's a guess, not a certainty. |
| **CLASS** | A guess at what kind of device it is (Router, Computer, Mobile, Media, Printer, IoT). netwp guesses from services the device announces, its open ports, and its manufacturer; it's sometimes wrong or shows "Unknown". You can fix it with `netwp class set` (see the FAQ). |
| **MAC** | The MAC address explained above: the device's permanent identity. |
| **HOSTNAME** | The name the device itself announces on the network (not every device announces one). |
| **VENDOR** | The network card's manufacturer (Apple, Samsung, TP-Link...), found from the first digits of the MAC. |
| **PORTS** | Which "ports" (network services) the device has open. Shows in red when it's a sensitive port (see the warning signs below). |
| **LAST SEEN** | How long ago the device was last seen, when it's offline. |

## Warning signs: security and a bad network

Not everything highlighted is a problem, but these are worth a second
look:

- **An unknown device joined the network** (an "⚠ ... joined (unknown)"
  line in `monitor`'s activity log): a MAC that never had a nickname set.
  It could just be a guest's phone, a new device you bought, or someone
  who shouldn't be on your network. If it's legitimate, give it a nickname
  with `netwp alias set` so it stops showing as unknown.
- **The same IP now answers with a different MAC** (flagged by `netwp scan
  --diff`): this is unusual. It could just be a coincidental device swap,
  but it's also the classic signature of an attack called ARP spoofing,
  where someone on the network pretends to be another device (sometimes
  even the router) to intercept traffic. Worth confirming where the change
  came from.
- **A MAC shows up at more than one IP in the same scan**: also unusual
  and worth attention, for the same reason above.
- **Sensitive ports open** (PORTS column in red: SSH/22, SMB/445,
  RDP/3389): these are remote-access ports. On a home network, leaving
  these open is almost never intentional. Check with `netwp ports <ip>`
  whether it makes sense for that device.
- **Red RTT / slow network**: the device is taking longer than 100ms to
  answer within the local network itself. Could be weak Wi-Fi, an
  overloaded device, or just a bad reading at that moment.
- **Low-bandwidth alert** (`netwp monitor --alert-down`): actual download
  speed dropped below the threshold you set. Could be your ISP, the
  router, or another device using up all the bandwidth.

To learn more about using netwp responsibly on a network that isn't yours,
see [SECURITY.md](../SECURITY.md).

## Frequently asked questions

**Does netwp send any of my data anywhere?**
No. Data stays on your own computer (`aliases.json`, `lastscan.json`,
`events.jsonl`), aside from the network traffic the scan itself needs (ARP,
ping, speed test). See [SECURITY.md](../SECURITY.md) for details.

**Can I use netwp to "hack into" someone else's network?**
Don't run netwp against networks you don't own or don't have explicit
authorization to scan. On a corporate network this can even violate its
acceptable-use policy. See [SECURITY.md](../SECURITY.md).

**Why does a device show "Unknown" in the CLASS column? Can I fix it?**
It's Unknown because netwp couldn't find enough of a clue (no announced
service, no recognized open port, no distinctive manufacturer) to risk a
guess. "Unknown" beats a wrong guess. When you know what a device is (say,
your phone), pin it: `netwp class set 192.168.1.20 mobile`. A manual pin,
kept by MAC address, always wins over the automatic guess and survives the
device changing IP. Undo it with `netwp class rm`, list pins with `netwp
class ls`. Valid classes: router, computer, mobile, media, printer, iot.

**Why does a device's IP change sometimes?**
That's normal: the router reassigns IPs via DHCP from time to time. netwp
uses the MAC (which never changes) to keep recognizing it as the same
device, so the nickname you gave it still applies.

**My internet is slow or not working. What can netwp tell me?**
Run `netwp doctor`. It checks, in order, whether your interface has an
address, whether your router (gateway) answers, whether the internet is
reachable, whether DNS resolves, and your Wi-Fi signal. Read it top to
bottom: the first ✗ is usually the real problem, and the checks below it
tend to fail as a consequence.

**Can I turn on a computer over the network?**
Yes, if that computer has "Wake-on-LAN" turned on in its BIOS/OS settings
(off by default on most machines). Then `netwp wake <ip|mac|alias>` sends
the wake signal. It's fire-and-forget: netwp says "sent", but whether the
machine actually wakes is up to that machine's settings.

**How do I uninstall netwp?**
Run `netwp uninstall`. It asks you to confirm, then removes the local data
netwp created (your nicknames, the scan cache, the event log) and prints
how to remove the program file itself. If you liked it (or didn't), the
same screen links to where you can leave a quick review.
