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

		fmt.Printf("Repo created at %s\n", repoRoot)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
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
