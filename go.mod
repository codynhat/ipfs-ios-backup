module github.com/codynhat/ipfs-ios-backup

go 1.14

require (
	github.com/go-co-op/gocron v0.1.2-0.20200429025551-8c7e3da6cc03
	github.com/golang/protobuf v1.4.2
	github.com/hsanjuan/ipfs-lite v1.1.13 // indirect
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-filestore v1.0.0 // indirect
	github.com/ipfs/go-ipfs v0.8.0
	github.com/ipfs/go-ipfs-config v0.12.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/libp2p/go-libp2p-core v0.8.0
	github.com/libp2p/go-sockaddr v0.1.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/spf13/cobra v0.0.7
	github.com/spf13/viper v1.4.0
	github.com/textileio/go-threads v0.1.18
	google.golang.org/grpc v1.31.1
	google.golang.org/protobuf v1.25.0
)

exclude github.com/libp2p/go-libp2p-crypto v0.0.2

exclude github.com/ipfs/go-filestore v0.0.3
