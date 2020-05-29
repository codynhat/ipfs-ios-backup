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
	"path/filepath"

	"github.com/codynhat/ipfs-ios-backup/idevice"
	"github.com/golang/protobuf/ptypes"
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

		// Pair device
		fmt.Println("Pairing device...")
		if err := idevice.PairDevice(deviceID); err != nil {
			log.Fatalf("Failed to pair device: %v\n", err)
		}
		fmt.Println("Device is paired.")

		// Enable backup encryption
		fmt.Println("Determining if backup encryption is enabled...")
		willEncrypt, err := idevice.GetDeviceWillEncrypt(deviceID)
		if err != nil {
			log.Fatalf("Failed to determine if backup encryption is enabled: %v", err)
		}

		if !willEncrypt {
			fmt.Println("Backup encryption is not enabled. Enabling...")
			if err := idevice.EnableBackupEncryption(deviceID); err != nil {
				log.Fatalf("Failed to enable backup encryption: %v", err)
			}
		}
		fmt.Println("Backup encryption is enabled.")
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

		err := performBackup(ctx, deviceID, repoPath)
		if err != nil {
			log.Fatal(err)
		}
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
			log.Fatalf("Failed to restore backup: %v", err)
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
			log.Fatalf("Failed to get backups: %s\n", err)
		}

		if len(backups.Backups) == 0 {
			fmt.Println("No backups found.")
			return
		}

		fmt.Println("Backups found:")
		fmt.Printf("[device-id] -> [IPFS cid]\n\n")
		for _, v := range backups.Backups {
			fmt.Printf("%s -> %s\n\tLast Backup At: %v\n", v.DeviceID, v.BackupCid, ptypes.TimestampString(v.UpdatedAt))
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

func performBackup(ctx context.Context, deviceID idevice.DeviceID, repoPath string) error {
	log.Infof("Performing backup for device %s", deviceID)

	backupDir := filepath.Join(repoPath, "backups")

	// Perform backup
	if err := idevice.PerformBackup(deviceID, backupDir); err != nil {
		return fmt.Errorf("failed to perform backup: %s", err)
	}

	// Add backup to IPFS
	log.Infof("Adding backup to IPFS")
	reply, err := client.AddBackup(ctx, backupDir)
	if err != nil {
		return err
	}
	log.Infof("Added backup to IPFS (%s)", reply.BackupCid)

	log.Infof("Updating latest backup CID")
	updateReply, err := client.UpdateLatestBackup(ctx, string(deviceID), reply.BackupCid)
	if err != nil {
		return err
	}
	log.Infof("Latest backup cid saved (%s)", updateReply.Backup.BackupCid)

	return nil
}
