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
	"fmt"
	"os"
	"path/filepath"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "Interact with iOS backups",
	Long:  "Interact with iOS backups",
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing iOS backups",
	Long:  "List existing iOS backups",
	Run: func(cmd *cobra.Command, args []string) {
		// ctx, cancel := context.WithCancel(context.Background())
		// defer cancel()

		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Could not create repo path:", err)
			os.Exit(1)
		}

		ipfslitePath := filepath.Join(repoRoot, "ipfslite")

		_, err = queryBackups(ipfslitePath)
		if err != nil {
			fmt.Println("Failed to fetch backups:", err)
			os.Exit(1)
		}
	},
}

var backupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Perform a backup",
	Long:  "Perform a backup",
	Run: func(cmd *cobra.Command, args []string) {
		// ctx, cancel := context.WithCancel(context.Background())
		// defer cancel()

		// Find repo path
		repoRoot, err := homedir.Expand("~/.ipfs-ios-backup")
		if err != nil {
			fmt.Println("Could not create repo path:", err)
			os.Exit(1)
		}

		ipfslitePath := filepath.Join(repoRoot, "ipfslite")

		err = saveBackupCid(ipfslitePath, "test-backup")
		if err != nil {
			fmt.Println("Failed to save backup:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(backupsCmd)
	backupsCmd.AddCommand(backupsListCmd)
	backupsCmd.AddCommand(backupsCreateCmd)
}

func queryBackups(ipfslitePath string) ([]string, error) {
	// Get IPFS lite datastore
	ds, err := ipfslite.BadgerDatastore(ipfslitePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to get IPFS lite datastore: %s", err)
	}

	q, err := ds.Query(query.Query{Prefix: "/backups"})
	if err != nil {
		return nil, fmt.Errorf("Query failed: %s", err)
	}

	all, err := q.Rest()
	if err != nil {
		return nil, fmt.Errorf("Query failed: %s", err)
	}

	for _, e := range all {
		fmt.Println(e)
	}

	return nil, nil
}

func saveBackupCid(ipfslitePath string, cid string) error {
	// Get IPFS lite datastore
	ds, err := ipfslite.BadgerDatastore(ipfslitePath)
	if err != nil {
		return fmt.Errorf("Failed to get IPFS lite datastore: %s", err)
	}

	if err = ds.Put(datastore.NewKey("/backups/"+cid), []byte("Qm-test")); err != nil {
		return fmt.Errorf("Failed to save backup CID: %s", err)
	}

	return nil
}
