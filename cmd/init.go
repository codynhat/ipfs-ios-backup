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
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/codynhat/ipfs-ios-backup/api"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/go-threads/common"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

var (
	secretsImportPath string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ipfs-ios-backup repo",
	Long:  "Initialize ipfs-ios-backup repo",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Find repo path
		repoPath := viper.GetString("repoPath")

		_, err := os.Stat(repoPath)
		if err == nil {
			log.Fatalf("repo already exists at %s\n", repoPath)
		}

		// Create repo
		if err := checkWritable(repoPath); err != nil {
			log.Fatal(err)
		}

		// Create backups dir
		backupDir := filepath.Join(repoPath, "backups")
		if err := checkWritable(backupDir); err != nil {
			log.Fatal(err)
		}

		var existingExport *export
		if secretsImportPath != "" {
			existingExport, err = importSecrets(secretsImportPath)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Importing secrets from %v\n", secretsImportPath)
		}

		// Get IPFS bootstrap list
		ipfsBootstrapList := viper.GetStringSlice("ipfsBootstrapList")

		// Get Threads bootstrap list
		threadsBootstrapList := viper.GetStringSlice("threadsBootstrapList")
		threadsBootstrapAddrs := make([]ma.Multiaddr, len(threadsBootstrapList))
		for i, v := range threadsBootstrapList {
			a, err := ma.NewMultiaddr(v)
			if err != nil {
				log.Fatal(err)
			}

			threadsBootstrapAddrs[i] = a
		}

		if err := initIpfsRepo(repoPath, existingExport, ipfsBootstrapList); err != nil {
			log.Fatal(err)
		}

		threadID, clean, err := initThreadsRepo(ctx, repoPath, existingExport, threadsBootstrapAddrs)
		defer clean()
		if err != nil {
			log.Fatalf("Could not initialize threads repo: %s", err)
		}

		fmt.Printf("Repo created at %s\n", repoPath)

		viper.Set("threadID", threadID)
		if err = viper.WriteConfig(); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Saved config to %s\n", viper.ConfigFileUsed())
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&secretsImportPath, "secrets", "", "Secrets file exported from another node")
}

func initIpfsRepo(repoRoot string, existingExport *export, ipfsBootstrapList []string) error {
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

	customizeConfig(cfg, ipfsBootstrapList)

	// Create the repo with the config
	err = fsrepo.Init(ipfsRepoRoot, cfg)
	if err != nil {
		return fmt.Errorf("Could not initialize repo: %s", err)
	}

	createSwarmKey(ipfsRepoRoot, existingExport)

	return nil
}

func customizeConfig(cfg *config.Config, ipfsBootstrapList []string) {
	// Set Bootstrap list
	cfg.Bootstrap = ipfsBootstrapList

	// Enable Filestore
	cfg.Experimental.FilestoreEnabled = true

	// Set swarm addr
	cfg.Addresses.Swarm = []string{ipfsAddr.String()}
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
func createSwarmKey(ipfsRepoRoot string, existingExport *export) error {
	var swarmKey string
	if existingExport == nil {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			return fmt.Errorf("While trying to create swarm key: %s", err)
		}

		encodedKey := hex.EncodeToString(key)
		swarmKey = fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base16/\n%s\n", encodedKey)
	} else {
		swarmKey = existingExport.SwarmKey
	}

	swarmKeyPath := filepath.Join(ipfsRepoRoot, "swarm.key")
	err := ioutil.WriteFile(swarmKeyPath, []byte(swarmKey), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write swarm key: %s", err)
	}

	return nil
}

func initThreadsRepo(ctx context.Context, repoRoot string, existingExport *export, bootstrapAddrs []ma.Multiaddr) (thread.ID, func(), error) {
	addrInfos, err := peer.AddrInfosFromP2pAddrs(bootstrapAddrs...)
	if err != nil {
		return thread.Undef, nil, err
	}

	net, err := common.DefaultNetwork(repoRoot, common.WithNetDebug(true), common.WithNetHostAddr(threadsAddr))
	if err != nil {
		return thread.Undef, nil, err
	}

	net.Bootstrap(addrInfos)

	var id thread.ID
	var d *db.DB
	if existingExport == nil {
		id = thread.NewIDV1(thread.Raw, 32)
		d, err = db.NewDB(ctx, net, id, db.WithNewDBRepoPath(repoRoot))
		if err != nil {
			return thread.Undef, func() { net.Close() }, err
		}

		_, err = d.NewCollection(db.CollectionConfig{
			Name:   "Backup",
			Schema: util.SchemaFromInstance(&api.Backup{}, false),
		})

		if err != nil {
			return thread.Undef, func() { d.Close(); net.Close() }, err
		}

		fmt.Printf("Created thread %s\n", id)
	} else {
		key, err := thread.KeyFromString(existingExport.ThreadKey)
		if err != nil {
			return thread.Undef, func() { net.Close() }, err
		}

		for _, rawAddr := range existingExport.Addrs {
			mctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			addr, err := ma.NewMultiaddr(rawAddr)
			if err != nil {
				return thread.Undef, func() { net.Close() }, err
			}

			id, err = thread.FromAddr(addr)
			if err != nil {
				log.Warnf("Could not dial addr %v: %v", addr, err)
				continue
			}

			// Try to connect
			threadComp, err := ma.NewComponent(thread.Name, id.String())
			if err != nil {
				log.Warnf("Could not dial addr %v: %v", addr, err)
				continue
			}
			peerAddr := addr.Decapsulate(threadComp)
			addri, err := peer.AddrInfoFromP2pAddr(peerAddr)
			if err != nil {
				log.Warnf("Could not dial addr %v: %v", addr, err)
				continue
			}
			if err = net.Host().Connect(ctx, *addri); err != nil {
				log.Warnf("Could not dial addr %v: %v", addr, err)
				continue
			}

			// If successful add DB
			cc1 := db.CollectionConfig{
				Name:   "Backup",
				Schema: util.SchemaFromInstance(&api.Backup{}, false),
			}

			d, err = db.NewDBFromAddr(mctx, net, addr, key, db.WithNewDBRepoPath(repoRoot), db.WithNewDBCollections(cc1))
			if err != nil {
				log.Warnf("Could not create db %v: %v", addr, err)
				continue
			}

			fmt.Printf("Joined thread %s\n", id)

			break
		}

		if d == nil {
			return thread.Undef, func() { net.Close() }, fmt.Errorf("could not create DB from existing export")
		}
	}

	return id, func() { d.Close(); net.Close() }, nil
}

func importSecrets(filePath string) (*export, error) {
	s, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	b2 := make([]byte, s.Size())
	n2, err := f.Read(b2)
	if err != nil {
		return nil, err
	}

	exportJ := string(b2[:n2])

	e := export{}

	json.Unmarshal([]byte(exportJ), &e)

	return &e, nil
}
