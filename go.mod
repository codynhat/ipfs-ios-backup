module github.com/codynhat/ipfs-ios-backup

go 1.14

require (
	github.com/go-co-op/gocron v0.1.2-0.20200429025551-8c7e3da6cc03
	github.com/golang/protobuf v1.5.2
	github.com/hsanjuan/ipfs-lite v1.1.13 // indirect
	github.com/ipfs/go-cid v0.2.0
	github.com/ipfs/go-ipfs v0.13.1
	github.com/ipfs/go-ipfs-config v0.5.3
	github.com/ipfs/go-ipfs-files v0.1.1
	github.com/ipfs/go-log v1.0.5
	github.com/ipfs/interface-go-ipfs-core v0.7.0
	github.com/libp2p/go-libp2p-core v0.15.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.4.0
	github.com/textileio/go-threads v0.1.18
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
)

exclude github.com/libp2p/go-libp2p-crypto v0.0.2

exclude github.com/ipfs/go-filestore v0.0.3
