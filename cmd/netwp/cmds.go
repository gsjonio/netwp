package main

import (
	"errors"
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

// runAlias dispatches the alias subcommands: set, ls, rm. Inline switch to
// match runClass/runWatch (the three store commands share this shape).
func runAlias() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return errors.New("usage: netwp alias set <ip-or-mac> <name> | alias ls | alias rm <ip-or-mac>")
	}
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls", "list":
		list := store.List()
		if len(list) == 0 {
			fmt.Println("no aliases set.")
			return nil
		}
		for _, a := range list {
			fmt.Printf("%-17s  %s\n", a.MAC, a.Name)
		}
		return nil
	case "set":
		if len(args) < 3 {
			return errors.New("usage: netwp alias set <ip-or-mac> <name>")
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		name := strings.Join(args[2:], " ")
		if err := store.Set(mac, name); err != nil {
			return err
		}
		fmt.Printf("aliased %s → %q\n", mac, name)
		return nil
	case "rm", "remove", "del":
		if len(args) < 2 {
			return errors.New("usage: netwp alias rm <ip-or-mac>")
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		if err := store.Delete(mac); err != nil {
			return err
		}
		fmt.Printf("removed alias for %s\n", mac)
		return nil
	default:
		return fmt.Errorf("unknown alias subcommand %q (use: set | ls | rm)", args[0])
	}
}

// runClass dispatches the class-override subcommands: set, ls, rm.
func runClass() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return errors.New("usage: netwp class set <ip-or-mac> <class> | class ls | class rm <ip-or-mac>\nclasses: router, computer, mobile, media, printer, iot")
	}
	store, err := openClassStore()
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls", "list":
		list := store.List()
		if len(list) == 0 {
			fmt.Println("no class overrides set.")
			return nil
		}
		for _, c := range list {
			fmt.Printf("%-17s  %s\n", c.MAC, c.Class)
		}
		return nil
	case "set":
		if len(args) < 3 {
			return errors.New("usage: netwp class set <ip-or-mac> <class> (router|computer|mobile|media|printer|iot)")
		}
		class, ok := core.ParseClass(args[2])
		if !ok {
			return fmt.Errorf("unknown class %q (use: router | computer | mobile | media | printer | iot)", args[2])
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		if err := store.Set(mac, class); err != nil {
			return err
		}
		fmt.Printf("pinned %s → %s\n", mac, class)
		return nil
	case "rm", "remove", "del":
		if len(args) < 2 {
			return errors.New("usage: netwp class rm <ip-or-mac>")
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		if err := store.Delete(mac); err != nil {
			return err
		}
		fmt.Printf("removed class override for %s\n", mac)
		return nil
	default:
		return fmt.Errorf("unknown class subcommand %q (use: set | ls | rm)", args[0])
	}
}

// runWatch dispatches the watch-list subcommands: add, ls, rm. A watched
// device triggers a highlighted alert (and a terminal bell) when it leaves
// during `netwp monitor` or `netwp dashboard`.
func runWatch() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return errors.New("usage: netwp watch add <ip-or-mac> | watch ls | watch rm <ip-or-mac>")
	}
	path, err := watchstore.DefaultPath()
	if err != nil {
		return err
	}
	store, err := watchstore.Open(path)
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls", "list":
		list := store.List()
		if len(list) == 0 {
			fmt.Println("no watched devices.")
			return nil
		}
		for _, mac := range list {
			fmt.Println(mac)
		}
		return nil
	case "add":
		if len(args) < 2 {
			return errors.New("usage: netwp watch add <ip-or-mac>")
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		if err := store.Add(mac); err != nil {
			return err
		}
		fmt.Printf("watching %s\n", mac)
		return nil
	case "rm", "remove", "del":
		if len(args) < 2 {
			return errors.New("usage: netwp watch rm <ip-or-mac>")
		}
		mac, err := resolveMAC(args[1])
		if err != nil {
			return err
		}
		if err := store.Remove(mac); err != nil {
			return err
		}
		fmt.Printf("stopped watching %s\n", mac)
		return nil
	default:
		return fmt.Errorf("unknown watch subcommand %q (use: add | ls | rm)", args[0])
	}
}

// runWake sends a Wake-on-LAN magic packet to a device named by MAC, IP, or
// alias. WoL is for devices that are asleep/offline, so an alias or a cached
// IP resolves even when the device won't answer an ARP sweep.
func runWake() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return errors.New("usage: netwp wake <ip-or-mac-or-alias>")
	}
	mac, err := resolveWakeTarget(args[0])
	if err != nil {
		return err
	}
	if err := wol.New().Wake(mac); err != nil {
		return err
	}
	fmt.Printf("sent Wake-on-LAN magic packet to %s\n", mac)
	return nil
}
