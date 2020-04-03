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

var backupsPerformCmd = &cobra.Command{
	Use:   "perform [device-id]",
	Short: "Perform a backup",
	Long:  "Perform a backup",
	Args:  cobra.ExactArgs(1),
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

		// deviceID := idevice.DeviceID(args[0])

		backupDir := filepath.Join(repoRoot, "backups")

		// Perform backup
		// if err = idevice.PerformBackup(deviceID, backupDir); err != nil {
		// 	fmt.Println("Failed to perform backup:", err)
		// 	os.Exit(1)
		// }

		// Add backup to IPFS
		fmt.Println("Adding backup to IPFS...")
		backupIpfsPath, err := addBackupToIpfs(ctx, ipfs, backupDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Added backup to IPFS (%s)\n", backupIpfsPath)

		fmt.Println("Publishing latest backup path to IPNS...")
		backupIpnsEntry, err := updateLatestBackupIpns(ctx, ipfs, backupIpfsPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Latest backup path published to IPNS with name %s\n", backupIpnsEntry.Name())
	},
}

func init() {
	rootCmd.AddCommand(backupsCmd)
	backupsCmd.AddCommand(backupsPerformCmd)
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

func updateLatestBackupIpns(ctx context.Context, ipfs icore.CoreAPI, backupIpfsPath path.Path) (icore.IpnsEntry, error) {
	opts := []options.NamePublishOption{
		options.Name.AllowOffline(true),
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
