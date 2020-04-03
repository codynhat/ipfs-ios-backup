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
	"fmt"
	"os"
	"path/filepath"

	"github.com/codynhat/ipfs-ios-backup/idevice"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "Interact with iOS backups",
	Long:  "Interact with iOS backups",
}

var backupsEnableCmd = &cobra.Command{
	Use:   "enable [device-id]",
	Short: "Enable backups for a device",
	Long:  "Enable backups for a device",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deviceID := idevice.DeviceID(args[0])

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Failed to find repo path:", err)
			os.Exit(1)
		}

		ipfsRepoRoot := filepath.Join(repoRoot, ".ipfs")

		// Spawn IPFS node
		ipfs, err := createIpfsNode(ctx, ipfsRepoRoot)
		if err != nil {
			fmt.Println("Failed to spawn IPFS node:", err)
			os.Exit(1)
		}

		// Pair device
		fmt.Println("Pairing device...")
		if err := idevice.PairDevice(deviceID); err != nil {
			fmt.Println("Failed to pair device:", err)
			os.Exit(1)
		}
		fmt.Println("Device is paired.")

		// Enable backup encryption
		fmt.Println("Determining if backup encryption is enabled...")
		willEncrypt, err := idevice.GetDeviceWillEncrypt(deviceID)
		if err != nil {
			fmt.Println("Failed to determine if backup encryption is enabled:", err)
			os.Exit(1)
		}

		if !willEncrypt {
			fmt.Println("Backup encryption is not enabled. Enabling...")
			if err := idevice.EnableBackupEncryption(deviceID); err != nil {
				fmt.Println("Failed to enable backup encryption:", err)
				os.Exit(1)
			}
		}
		fmt.Println("Backup encryption is enabled.")

		// Create IPNS key
		key, err := getIpnsKeyForDevice(ctx, ipfs, deviceID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if key == nil {
			fmt.Println("Generating IPNS key...")
			key, err = createBackupIpnsKey(ctx, ipfs, deviceID)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Generated IPNS key (%s -> %s)\n", key.Name(), key.Path())
		} else {
			fmt.Printf("IPNS key exists (%s -> %s)\n", key.Name(), key.Path())
		}
	},
}

var backupsPerformCmd = &cobra.Command{
	Use:   "perform [device-id]",
	Short: "Perform a backup",
	Long:  "Perform a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deviceID := idevice.DeviceID(args[0])

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Failed to find repo path:", err)
			os.Exit(1)
		}

		ipfsRepoRoot := filepath.Join(repoRoot, ".ipfs")

		// Spawn IPFS node
		ipfs, err := createIpfsNode(ctx, ipfsRepoRoot)
		if err != nil {
			fmt.Println("Failed to spawn IPFS node:", err)
			os.Exit(1)
		}

		backupDir := filepath.Join(repoRoot, "backups")

		// Get IPNS key
		key, err := getIpnsKeyForDevice(ctx, ipfs, deviceID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if key == nil {
			fmt.Println("IPNS key does not exist for device. Have backups for this device been enabled?")
			os.Exit(1)
		}

		// Perform backup
		if err = idevice.PerformBackup(deviceID, backupDir); err != nil {
			fmt.Println("Failed to perform backup:", err)
			os.Exit(1)
		}

		// Add backup to IPFS
		fmt.Println("Adding backup to IPFS...")
		backupIpfsPath, err := addBackupToIpfs(ctx, ipfs, backupDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Added backup to IPFS (%s)\n", backupIpfsPath)

		fmt.Println("Publishing latest backup path to IPNS...")
		backupIpnsEntry, err := updateLatestBackupIpns(ctx, ipfs, backupIpfsPath, key)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Latest backup path published to IPNS (%s -> %s)\n", backupIpnsEntry.Name(), backupIpnsEntry.Value())
	},
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups that exist",
	Long:  "List backups that exist",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Failed to find repo path:", err)
			os.Exit(1)
		}

		ipfsRepoRoot := filepath.Join(repoRoot, ".ipfs")

		// Spawn IPFS node
		ipfs, err := createIpfsNode(ctx, ipfsRepoRoot)
		if err != nil {
			fmt.Println("Failed to spawn IPFS node:", err)
			os.Exit(1)
		}

		// Get all IPNS keys
		keys, err := ipfs.Key().List(ctx)
		if err != nil {
			fmt.Printf("Failed to get IPNS keys: %s\n", err)
			os.Exit(1)
		}

		if len(keys) == 0 {
			fmt.Println("No backups found.")
			return
		}

		fmt.Println("Backups found:")
		fmt.Println("[device-id]:\n\t [IPNS path] -> [IPFS path]\n")
		for _, v := range keys {
			if v.Name() != "self" {
				path, err := ipfs.Name().Resolve(ctx, v.Path().String())
				ipfsPath := "no backup found"
				if err != nil {
					ipfsPath = fmt.Sprintf("ERROR: %s", err)
				}
				if path != nil && path.String() != "" {
					ipfsPath = path.String()
				}

				fmt.Printf("%s:\n\t %s -> %s\n", v.Name(), v.Path(), ipfsPath)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(backupsCmd)
	backupsCmd.AddCommand(backupsEnableCmd)
	backupsCmd.AddCommand(backupsPerformCmd)
	backupsCmd.AddCommand(backupsListCmd)
}

func addBackupToIpfs(ctx context.Context, ipfs icore.CoreAPI, backupDir string) (path.Path, error) {
	backupDirNode, err := getUnixfsNode(backupDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to find backup: %s", err)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Nocopy(true),
	}
	cidDirectory, err := ipfs.Unixfs().Add(ctx, backupDirNode, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to add backup to IPFS: %s", err)
	}

	return cidDirectory, nil
}

func getIpnsKeyForDevice(ctx context.Context, ipfs icore.CoreAPI, deviceID idevice.DeviceID) (icore.Key, error) {
	keys, err := ipfs.Key().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get IPNS keys: %s", err)
	}

	deviceIDRaw := string(deviceID)
	for _, v := range keys {
		if v.Name() == deviceIDRaw {
			return v, nil
		}
	}

	return nil, nil
}

func createBackupIpnsKey(ctx context.Context, ipfs icore.CoreAPI, deviceID idevice.DeviceID) (icore.Key, error) {
	opts := []options.KeyGenerateOption{
		options.Key.Size(2048),
		options.Key.Type(options.RSAKey),
	}

	key, err := ipfs.Key().Generate(ctx, string(deviceID), opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate IPNS key: %s", err)
	}

	return key, nil
}

func updateLatestBackupIpns(ctx context.Context, ipfs icore.CoreAPI, backupIpfsPath path.Path, key icore.Key) (icore.IpnsEntry, error) {
	opts := []options.NamePublishOption{
		options.Name.AllowOffline(true),
		options.Name.Key(key.Name()),
	}

	ipnsEntry, err := ipfs.Name().Publish(ctx, backupIpfsPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to publish to IPNS: %s", err)
	}

	return ipnsEntry, nil
}

// See https://github.com/ipfs/go-ipfs/blob/master/docs/examples/go-ipfs-as-a-library/main.go
func createIpfsNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Setup plugins
	if err := setupPlugins(repoPath); err != nil {
		return nil, fmt.Errorf("Failed to setup plugins: %s", err)
	}

	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  false,
		Routing: libp2p.NilRouterOption,
		Repo:    repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}
