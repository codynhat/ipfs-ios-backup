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
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/bootstrap"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/go-threads/common"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
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

		// Get IPFS bootstrap list
		ipfsBootstrapList := viper.GetStringSlice("ipfsBootstrapList")
		ipfsBootstrapAddrs := make([]ma.Multiaddr, len(ipfsBootstrapList))
		for i, v := range ipfsBootstrapList {
			a, err := ma.NewMultiaddr(v)
			if err != nil {
				log.Fatal(err)
			}

			ipfsBootstrapAddrs[i] = a
		}

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

		// Spawn IPFS node
		ipfs, err := createIpfsNode(ctx, ipfsRepoRoot, ipfsBootstrapAddrs)
		if err != nil {
			log.Fatalf("Failed to spawn IPFS node: %v", err)
		}

		// Load backup collection
		rawThreadID := viper.GetString("threadID")
		threadID, err := thread.Decode(rawThreadID)
		if err != nil {
			log.Fatal(err)
		}

		d, clean, err := loadBackupDB(repoPath, threadID, viper.GetBool("debug"), threadsBootstrapAddrs)
		defer clean()
		if err != nil {
			log.Fatal(err)
		}

		collection := d.GetCollection("Backup")

		log.Info("Listening for backups performed by others on the thread...")
		err = listenForBackups(ctx, d, collection, ipfs)
		if err != nil {
			log.Fatal(err)
		}

		// Run schedules
		schedules := viper.Sub("schedules")
		if schedules != nil {
			log.Info("Starting schedules")
			err = startSchedules(ctx, viper.Sub("schedules"), repoPath)
			if err != nil {
				log.Fatal(err)
			}
		}

		ptarget, err := TcpAddrFromMultiAddr(apiAddr)
		if err != nil {
			log.Fatal(err)
		}

		lis, err := net.Listen("tcp", ptarget)
		if err != nil {
			log.Fatal(err)
		}

		service, err := api.NewService(ipfs, d)
		if err != nil {
			log.Fatal(err)
		}

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
func createIpfsNode(ctx context.Context, repoPath string, bootstrapAddrs []ma.Multiaddr) (icore.CoreAPI, error) {
	// Check if swarm key exists
	swarmKeyPath := filepath.Join(repoPath, "swarm.key")
	_, err := os.Stat(swarmKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Swarm key does not exist. Refusing to start IPFS node. Try running `ipfs-ios-backup init`")
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
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	addrs := node.PeerHost.Addrs()
	for _, addr := range addrs {
		fmt.Printf("IPFS node started listening on %v/p2p/%v\n", addr, node.Identity)
	}

	// Bootstrap
	addrInfos, err := peer.AddrInfosFromP2pAddrs(bootstrapAddrs...)
	if err != nil {
		return nil, err
	}
	node.Bootstrap(bootstrap.BootstrapConfigWithPeers(addrInfos))

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

func loadBackupDB(repoRoot string, threadID thread.ID, debug bool, bootstrapAddrs []ma.Multiaddr) (*db.DB, func(), error) {
	addrInfos, err := peer.AddrInfosFromP2pAddrs(bootstrapAddrs...)
	if err != nil {
		return nil, nil, err
	}

	net, err := common.DefaultNetwork(repoRoot, common.WithNetDebug(debug), common.WithNetHostAddr(threadsAddr))
	if err != nil {
		return nil, nil, err
	}

	net.Bootstrap(addrInfos)

	d, err := db.NewDB(context.Background(), net, threadID, db.WithNewDBRepoPath(repoRoot))
	if err != nil {
		return nil, nil, err
	}

	return d, func() { d.Close() }, nil
}

// Listen for new backups made on any device in the thread
func listenForBackups(ctx context.Context, d *db.DB, collection *db.Collection, ipfs icore.CoreAPI) error {
	l, err := d.Listen(db.ListenOption{
		Type:       db.ListenAll,
		Collection: "Backup",
	})

	if err != nil {
		return err
	}

	go func() {
		defer l.Close()

		for action := range l.Channel() {
			switch action.Type {
			case db.ActionDelete:
				break
			default:
				v, err := collection.FindByID(action.ID)
				if err != nil {
					log.Errorf("error when listening to thread: %v", err)
					continue
				}

				backup := &api.Backup{}
				util.InstanceFromJSON(v, backup)

				id, err := cid.Decode(backup.LatestBackupCid)
				if err != nil {
					log.Errorf("error when listening to thread: %v", err)
					continue
				}

				log.Infof("Found new backup. Pinning %v", id)
				err = ipfs.Pin().Add(ctx, path.IpfsPath(id))
				log.Infof("Pinned %v", id)
				if err != nil {
					log.Errorf("error when listening to thread: %v", err)
					continue
				}
			}
		}
	}()

	return nil
}

func startSchedules(ctx context.Context, schedules *viper.Viper, repoPath string) error {
	s1 := gocron.NewScheduler(time.UTC)

	for name := range schedules.AllSettings() {
		schedule := schedules.Sub(name)

		rawDeviceID := schedule.GetString("deviceID")
		deviceID := idevice.DeviceID(rawDeviceID)
		periodInHours := schedule.GetUint64("periodInHours")

		onlyWhenCharging := false
		var minBatteryLevel int
		if schedule.IsSet("onlyWhenCharging") {
			onlyWhenCharging = schedule.GetBool("onlyWhenCharging")
		} else {
			minBatteryLevel = schedule.GetInt("minBatteryLevel")
		}

		s1.Every(uint64(periodInHours)).Hours().StartImmediately().Do(runScheduledBackup, ctx, deviceID, repoPath, minBatteryLevel, onlyWhenCharging)

		log.Infof("Scheduled backup for device %s (%v)", deviceID, schedule.AllSettings())
	}

	s1.StartAsync()

	return nil
}

func runScheduledBackup(ctx context.Context, deviceID idevice.DeviceID, repoPath string, minBatteryLevel int, onlyWhenCharging bool) {
	log.Infof("Backup triggered for device %s", deviceID)
	log.Infof("onlyWhenCharging is %v", onlyWhenCharging)

	log.Infof("Checking if device is on charger")

	isCharging, err := idevice.GetDeviceBatteryIsCharging(deviceID)
	if err != nil {
		log.Errorf("failed to check if device is charging: %s", err)
		return
	}

	if !isCharging {
		if onlyWhenCharging {
			log.Infof("Device is not on charger. Skipping backup.")
			return
		}

		log.Infof("Checking if battery level >= %v%%", minBatteryLevel)

		currentBatteryLevel, err := idevice.GetDeviceBatteryCurrentCapacity(deviceID)
		if err != nil {
			log.Errorf("failed to check device battery level: %s", err)
			return
		}

		if int(currentBatteryLevel) < minBatteryLevel {
			log.Warnf("Device is not charged enough (%v%% < %v%%). Skipping backup.", currentBatteryLevel, minBatteryLevel)
			return
		}
	}

	err = performBackup(ctx, deviceID, repoPath)
	if err != nil {
		log.Error(err)
		return
	}
}
