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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		reply, err := client.GetKeyForDevice(ctx, string(deviceID))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		key := reply.Key

		if key == nil {
			fmt.Println("Generating IPNS key...")
			reply, err := client.CreateKeyForDevice(ctx, string(deviceID))
			key = reply.Key
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Generated IPNS key (%s -> %s)\n", key.Name, key.Path)
		} else {
			fmt.Printf("IPNS key exists (%s -> %s)\n", key.Name, key.Path)
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
		repoPath := viper.GetString("repoPath")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		backupDir := filepath.Join(repoPath, "backups")

		// Get IPNS key
		reply, err := client.GetKeyForDevice(ctx, string(deviceID))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		key := reply.Key

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
		backupIpfsPath, err := client.AddBackup(ctx, backupDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Added backup to IPFS (%s)\n", backupIpfsPath)

		fmt.Println("Publishing latest backup path to IPNS...")
		backupIpnsEntry, err := client.UpdateLatestBackup(ctx, string(deviceID), backupIpfsPath.BackupPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Latest backup path published to IPNS (%s -> %s)\n", backupIpnsEntry.Entry.Name, backupIpnsEntry.Entry.Value)
	},
}

var backupsRestoreCmd = &cobra.Command{
	Use:   "restore [device-id]",
	Short: "Restore a backup",
	Long:  "Restore a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deviceID := idevice.DeviceID(args[0])
		repoPath := viper.GetString("repoPath")

		backupDir := filepath.Join(repoPath, "backups")

		// Restore backup
		if err := idevice.RestoreBackup(deviceID, backupDir); err != nil {
			fmt.Println("Failed to restore backup:", err)
			os.Exit(1)
		}
	},
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups that exist",
	Long:  "List backups that exist",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Get all backups
		backups, err := client.ListBackups(ctx)
		if err != nil {
			fmt.Printf("Failed to get backups: %s\n", err)
			os.Exit(1)
		}

		if len(backups.Backups) == 0 {
			fmt.Println("No backups found.")
			return
		}

		fmt.Println("Backups found:")
		fmt.Printf("[device-id]:\n\t [IPNS path] -> [IPFS path]\n\n")
		for _, v := range backups.Backups {
			fmt.Printf("%s:\n\t %s -> %s\n", v.Key.Name, v.Key.Path, v.IpfsPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(backupsCmd)
	backupsCmd.AddCommand(backupsEnableCmd)
	backupsCmd.AddCommand(backupsPerformCmd)
	backupsCmd.AddCommand(backupsListCmd)
	backupsCmd.AddCommand(backupsRestoreCmd)
}
