package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	configDir  = "/etc/nftabler"
	nftBinPath = "/usr/sbin/nft"
)

func applyFile(path string) error {
	// Open the rule file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	// Run: nft -f -
	cmd := exec.Command(nftBinPath, "-f", "-")
	cmd.Stdin = file
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nft error: %w, output: %s", err, out)
	}
	log.Printf("Applied %s successfully", filepath.Base(path))
	return nil
}

func applyIfRuleFile(path string, d fs.DirEntry) error {
	if d.IsDir() {
		return nil
	}
	if filepath.Ext(path) != ".nft" {
		return fmt.Errorf("checked path is not a config file: %s", path)
	}
	return applyFile(path)

}

// Check directory and call applyIfRulefile on all files
func walkFiles() error {
	return filepath.WalkDir(configDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		applyIfRuleFile(path, d)
		return nil
	})
}

func main() {
	// Load any pre‑existing files
	if err := walkFiles(); err != nil {
		log.Fatalf("initial load failed: %v", err)
	}

	// Set up inotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("fsnotify: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(configDir); err != nil {
		log.Fatalf("add watch: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		log.Printf("shutting down")
		cancel()
	}()

	// Event loop
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				// We care about Create, Write, Rename (new file) and Remove (maybe cleanup)
				if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
					continue
				}
				// Small debounce – a ConfigMap update writes a temp file then renames
				time.Sleep(100 * time.Millisecond)
				_, err := os.Stat(ev.Name)
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						// file was removed before we could stat it
						continue
					}
					log.Printf("stat %s: %v", ev.Name, err)
					continue
				}
				walkFiles()
			case err := <-watcher.Errors:
				log.Printf("watcher error: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Block forever (or until SIGTERM)
	<-ctx.Done()
}
