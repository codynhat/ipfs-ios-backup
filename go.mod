module github.com/codynhat/ipfs-ios-backup

go 1.14

require (
	github.com/go-co-op/gocron v0.1.2-0.20200429025551-8c7e3da6cc03
	github.com/golang/protobuf v1.4.0
	github.com/hsanjuan/ipfs-lite v1.1.13
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-ds-badger v0.2.4
	github.com/ipfs/go-ipfs v0.5.1
	github.com/ipfs/go-ipfs-config v0.5.3
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/interface-go-ipfs-core v0.2.7
	github.com/libp2p/go-libp2p v0.8.3
	github.com/libp2p/go-libp2p-connmgr v0.2.1
	github.com/libp2p/go-libp2p-core v0.5.3
	github.com/libp2p/go-libp2p-host v0.0.3
	github.com/libp2p/go-libp2p-kad-dht v0.7.11
	github.com/libp2p/go-libp2p-peerstore v0.2.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.2.1
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.4.0
	github.com/textileio/go-threads v0.1.18
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.23.0
)

exclude github.com/libp2p/go-libp2p-crypto v0.0.2

exclude github.com/ipfs/go-filestore v0.0.3
