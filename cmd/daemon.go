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
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/codynhat/ipfs-ios-backup/api"
	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	"github.com/codynhat/ipfs-ios-backup/idevice"
	"github.com/go-co-op/gocron"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the ipfs-ios-backup daemon",
	Long:  "Run the ipfs-ios-backup daemon",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Find repo path
		repoPath := viper.GetString("repoPath")

		ipfsRepoRoot := filepath.Join(repoPath, ".ipfs")

		// Spawn IPFS node
		ipfs, err := createIpfsNode(ctx, ipfsRepoRoot)
		if err != nil {
			fmt.Println("Failed to spawn IPFS node:", err)
			os.Exit(1)
		}

		// Run schedules
		err = startSchedules(ctx, viper.Sub("schedules"), repoPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ptarget, err := TcpAddrFromMultiAddr(addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		lis, err := net.Listen("tcp", ptarget)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		service := api.NewService(ipfs)

		grpcServer := grpc.NewServer()
		pb.RegisterAPIServer(grpcServer, service)
		grpcServer.Serve(lis)
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func TcpAddrFromMultiAddr(maddr ma.Multiaddr) (addr string, err error) {
	if maddr == nil {
		err = fmt.Errorf("invalid address")
		return
	}
	ip4, err := maddr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		return
	}
	tcp, err := maddr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return
	}
	return fmt.Sprintf("%s:%s", ip4, tcp), nil
}

// See https://github.com/ipfs/go-ipfs/blob/master/docs/examples/go-ipfs-as-a-library/main.go
func createIpfsNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Check if swarm key exists
	swarmKeyPath := filepath.Join(repoPath, "swarm.key")
	_, err := os.Stat(swarmKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Swarm key does not exist. Refusing to start IPFS node. Try running `ipfs-ios-desktop init`")
	}

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

func startSchedules(ctx context.Context, schedules *viper.Viper, repoPath string) error {
	s1 := gocron.NewScheduler(time.UTC)

	for rawDeviceID := range schedules.AllSettings() {
		deviceID := idevice.DeviceID(rawDeviceID)

		schedule := schedules.Sub(rawDeviceID)
		periodInHours := schedule.GetUint64("periodInHours")

		onlyWhenCharging := false
		var minBatteryLevel int
		if schedule.IsSet("onlyWhenCharging") {
			onlyWhenCharging = schedule.GetBool("onlyWhenCharging")
		} else {
			minBatteryLevel = schedule.GetInt("minBatteryLevel")
		}

		s1.Every(uint64(periodInHours)).Hours().StartImmediately().Do(runScheduledBackup, ctx, deviceID, repoPath, minBatteryLevel, onlyWhenCharging)

		fmt.Printf("Scheduled backup for device %s (%v)\n", deviceID, schedule.AllSettings())
	}

	s1.Start()

	return nil
}

func runScheduledBackup(ctx context.Context, deviceID idevice.DeviceID, repoPath string, minBatteryLevel int, onlyWhenCharging bool) error {
	fmt.Printf("Backup triggered for device %s\n", deviceID)
	return performBackup(ctx, deviceID, repoPath)
}
