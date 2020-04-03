/*
Copyright © 2020 Cody Hatfield <cody.hatfield@me.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ipfs-ios-backup repo",
	Long:  "Initialize ipfs-ios-backup repo",
	Run: func(cmd *cobra.Command, args []string) {
		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Could not create repo path:", err)
			os.Exit(1)
		}

		_, err = os.Stat(repoRoot)
		if err == nil {
			fmt.Printf("Repo already exists at %s\n", repoRoot)
			os.Exit(1)
		}

		// Create repo
		if err := checkWritable(repoRoot); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Create backups dir
		backupDir := filepath.Join(repoRoot, "backups")
		if err := checkWritable(backupDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := initIpfsRepo(repoRoot); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Repo created at %s\n", repoRoot)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func initIpfsRepo(repoRoot string) error {
	ipfsRepoRoot := filepath.Join(repoRoot, ".ipfs")

	if err := checkWritable(ipfsRepoRoot); err != nil {
		return err
	}

	if fsrepo.IsInitialized(ipfsRepoRoot) {
		return fmt.Errorf("IPFS Repo already exists at %s", ipfsRepoRoot)
	}

	// Setup plugins
	if err := setupPlugins(ipfsRepoRoot); err != nil {
		return fmt.Errorf("Failed to setup plugins: %s", err)
	}

	// Create config
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return fmt.Errorf("Could not create config: %s", err)
	}

	customizeConfig(cfg)

	// Create the repo with the config
	err = fsrepo.Init(ipfsRepoRoot, cfg)
	if err != nil {
		return fmt.Errorf("Could not initialize repo: %s", err)
	}

	createSwarmKey(ipfsRepoRoot)

	return nil
}

func customizeConfig(cfg *config.Config) {
	// Remove Bootstrap list
	cfg.Bootstrap = make([]string, 0)

	// Enable Filestore
	cfg.Experimental.FilestoreEnabled = true
}

// Taken from go-ipfs example
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Taken from go-ipfs init
func checkWritable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := filepath.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesn't exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

// See https://github.com/Kubuxu/go-ipfs-swarm-key-gen
func createSwarmKey(ipfsRepoRoot string) error {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return fmt.Errorf("While trying to create swarm key: %s", err)
	}

	encodedKey := hex.EncodeToString(key)
	swarmKeyPath := filepath.Join(ipfsRepoRoot, "swarm.key")
	err = ioutil.WriteFile(swarmKeyPath, []byte(fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base16/\n%s\n", encodedKey)), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write swarm key: %s", err)
	}

	return nil
}
