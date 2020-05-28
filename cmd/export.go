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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type export struct {
	Addrs     []string
	ThreadKey string
	SwarmKey  string
}

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export-secrets",
	Short: "Export secrets needed to share backups with another device",
	Long:  "Export secrets needed to share backups with another device",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		repoPath := viper.GetString("repoPath")
		ipfsRepoRoot := filepath.Join(repoPath, ".ipfs")

		// Get export
		reply, err := client.Export(ctx)
		if err != nil {
			log.Fatalf("Failed to export: %s\n", err)
		}

		// Get swarm key
		swarmKeyPath := filepath.Join(ipfsRepoRoot, "swarm.key")
		f, err := os.Open(swarmKeyPath)
		if err != nil {
			log.Fatal(err)
		}

		b2 := make([]byte, 128)
		n2, err := f.Read(b2)
		if err != nil {
			log.Fatal(err)
		}

		e := export{
			Addrs:     reply.Addrs,
			ThreadKey: reply.ThreadKey,
			SwarmKey:  string(b2[:n2]),
		}

		obj, err := json.Marshal(e)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(obj))
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
