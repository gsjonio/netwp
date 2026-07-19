package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gsjonio/netwp/internal/adapter/watchstore"
	"github.com/gsjonio/netwp/internal/adapter/wol"
	"github.com/gsjonio/netwp/internal/core"
)

// updateModule is the same module path documented in the README's install
// instructions, kept in one place so `netwp update` can't drift from it.
const updateModule = "github.com/gsjonio/netwp/cmd/netwp@latest"

// runUpdate re-runs the same `go install` the README tells people to use by
// hand, so updating doesn't require remembering or retyping the module path.
// This needs the Go toolchain; a binary downloaded from Releases has no
// self-update path (see SECURITY.md/README "Updating").
//
// Overwriting the running binary works even on Windows: go install builds to
// a temp file and renames it into place, and Windows allows renaming a file
// out from under its own running image (verified against a live `netwp
// monitor` process during development).
func runUpdate() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found on PATH; download the latest binary instead from https://github.com/gsjonio/netwp/releases/latest")
	}
	fmt.Printf("updating: go install %s\n", updateModule)
	cmd := exec.Command("go", "install", updateModule)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	fmt.Println("done. run \"netwp version\" to confirm.")
	return nil
}

// feedbackURL opens a pre-titled GitHub issue so an uninstalling user can
// leave a review without hunting for where to click.
const feedbackURL = "https://github.com/gsjonio/netwp/issues/new?labels=feedback&title=My%20netwp%20review"

// runUninstall removes netwp's local data (aliases, scan cache, event log)
// after a typed confirmation, then prints how to remove the binary. It never
// deletes the binary itself: a running program can't reliably delete its own
// executable on Windows, and a clear instruction beats a silent surprise.
func runUninstall() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "netwp")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("no netwp data found at %s.\n", dir)
	} else {
		fmt.Printf("This removes netwp's local data (aliases, scan cache, event log):\n  %s\nType \"yes\" to continue: ", dir)
		if !promptYes() {
			fmt.Println("aborted. Nothing was removed.")
			return nil
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		fmt.Println("removed.")
	}

	fmt.Printf(`
To remove the binary too:
  go clean -i github.com/gsjonio/netwp/cmd/netwp
  (or delete netwp from your Go bin directory: go env GOPATH)

Thanks for trying netwp. If you'd like to leave a review or feedback:
  %s
`, feedbackURL)
	return nil
}

// aliasSet nicknames a device. nameParts are joined with spaces so an unquoted
// multi-word name still works. The alias is keyed by MAC, so it survives DHCP.
func aliasSet(idOrMac string, nameParts []string) error {
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	name := strings.Join(nameParts, " ")
	if err := store.Set(mac, name); err != nil {
		return err
	}
	fmt.Printf("aliased %s → %q\n", mac, name)
	return nil
}

func aliasList(asJSON bool) error {
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	list := store.List()
	if asJSON {
		return printJSON(aliasesJSON(list))
	}
	if len(list) == 0 {
		fmt.Println("no aliases set.")
		return nil
	}
	for _, a := range list {
		fmt.Printf("%-17s  %s\n", a.MAC, a.Name)
	}
	return nil
}

func aliasRemove(idOrMac string) error {
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	if err := store.Delete(mac); err != nil {
		return err
	}
	fmt.Printf("removed alias for %s\n", mac)
	return nil
}

// classSet pins a device's class when the automatic guess is wrong; the pin is
// kept by MAC and always wins over the guess.
func classSet(idOrMac, className string) error {
	class, ok := core.ParseClass(className)
	if !ok {
		return fmt.Errorf("unknown class %q (use: router | computer | mobile | media | printer | iot)", className)
	}
	store, err := openClassStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	if err := store.Set(mac, class); err != nil {
		return err
	}
	fmt.Printf("pinned %s → %s\n", mac, class)
	return nil
}

func classList(asJSON bool) error {
	store, err := openClassStore()
	if err != nil {
		return err
	}
	list := store.List()
	if asJSON {
		return printJSON(classesJSON(list))
	}
	if len(list) == 0 {
		fmt.Println("no class overrides set.")
		return nil
	}
	for _, c := range list {
		fmt.Printf("%-17s  %s\n", c.MAC, c.Class)
	}
	return nil
}

func classRemove(idOrMac string) error {
	store, err := openClassStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	if err := store.Delete(mac); err != nil {
		return err
	}
	fmt.Printf("removed class override for %s\n", mac)
	return nil
}

// openWatchStore opens the persistent watch list at its default path. A watched
// device triggers a highlighted alert (and a terminal bell) when it leaves
// during `netwp monitor` or `netwp dashboard`.
func openWatchStore() (*watchstore.Store, error) {
	path, err := watchstore.DefaultPath()
	if err != nil {
		return nil, err
	}
	return watchstore.Open(path)
}

func watchAdd(idOrMac string) error {
	store, err := openWatchStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	if err := store.Add(mac); err != nil {
		return err
	}
	fmt.Printf("watching %s\n", mac)
	return nil
}

func watchList(asJSON bool) error {
	store, err := openWatchStore()
	if err != nil {
		return err
	}
	list := store.List()
	if asJSON {
		return printJSON(watchedJSON(list))
	}
	if len(list) == 0 {
		fmt.Println("no watched devices.")
		return nil
	}
	for _, mac := range list {
		fmt.Println(mac)
	}
	return nil
}

func watchRemove(idOrMac string) error {
	store, err := openWatchStore()
	if err != nil {
		return err
	}
	mac, err := resolveMAC(idOrMac)
	if err != nil {
		return err
	}
	if err := store.Remove(mac); err != nil {
		return err
	}
	fmt.Printf("stopped watching %s\n", mac)
	return nil
}

// runWake sends a Wake-on-LAN magic packet to a device named by MAC, IP, or
// alias. WoL is for devices that are asleep/offline, so an alias or a cached
// IP resolves even when the device won't answer an ARP sweep.
func runWake(target string) error {
	mac, err := resolveWakeTarget(target)
	if err != nil {
		return err
	}
	if err := wol.New().Wake(mac); err != nil {
		return err
	}
	fmt.Printf("sent Wake-on-LAN magic packet to %s\n", mac)
	return nil
}
