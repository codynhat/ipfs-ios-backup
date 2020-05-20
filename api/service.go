package api

import (
	"context"
	"fmt"
	"os"

	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	"github.com/codynhat/ipfs-ios-backup/idevice"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

// Service is a gRPC service
type Service struct {
	ipfs icore.CoreAPI
}

func NewService(ipfs icore.CoreAPI) *Service {
	return &Service{
		ipfs: ipfs,
	}
}

// GetKeyForDevice fetches the IPNS key for a device
func (s *Service) GetKeyForDevice(ctx context.Context, req *pb.GetKeyForDeviceRequest) (*pb.GetKeyForDeviceReply, error) {
	deviceID := idevice.DeviceID(req.DeviceID)

	key, err := s.getIpnsKeyForDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return &pb.GetKeyForDeviceReply{
			Key: nil,
		}, nil
	}

	return &pb.GetKeyForDeviceReply{
		Key: &pb.Key{
			Name: key.Name(),
			Path: key.Path().String(),
		},
	}, nil
}

// CreateKeyForDevice creates a new IPNS key for a device
func (s *Service) CreateKeyForDevice(ctx context.Context, req *pb.CreateKeyForDeviceRequest) (*pb.CreateKeyForDeviceReply, error) {
	deviceID := idevice.DeviceID(req.DeviceID)

	key, err := s.createBackupIpnsKey(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return &pb.CreateKeyForDeviceReply{
		Key: &pb.Key{
			Name: key.Name(),
			Path: key.Path().String(),
		},
	}, nil
}

// AddBackup adds a new backup to IPFS
func (s *Service) AddBackup(ctx context.Context, req *pb.AddBackupRequest) (*pb.AddBackupReply, error) {
	backupIpfsPath, err := s.addBackupToIpfs(ctx, req.BackupDir)
	if err != nil {
		return nil, err
	}

	return &pb.AddBackupReply{
		BackupPath: backupIpfsPath.String(),
	}, nil
}

// UpdateLatestBackup saves a reference to the latest backup
func (s *Service) UpdateLatestBackup(ctx context.Context, req *pb.UpdateLatestBackupRequest) (*pb.UpdateLatestBackupReply, error) {
	deviceID := idevice.DeviceID(req.DeviceID)

	key, err := s.getIpnsKeyForDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, fmt.Errorf("IPNS key does not exist for device. Have backups for this device been enabled?")
	}

	path := path.New(req.BackupPath)

	backupIpnsEntry, err := s.updateLatestBackupIpns(ctx, path, key)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateLatestBackupReply{
		Entry: &pb.UpdateLatestBackupReply_Entry{
			Name:  backupIpnsEntry.Name(),
			Value: backupIpnsEntry.Value().String(),
		},
	}, nil
}

// ListBackups lists all known backups
func (s *Service) ListBackups(ctx context.Context, req *pb.ListBackupsRequest) (*pb.ListBackupsReply, error) {
	// Get all IPNS keys
	keys, err := s.ipfs.Key().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get IPNS keys: %s", err)
	}

	var results []*pb.ListBackupsReply_Backup
	for _, key := range keys {
		if key.Name() != "self" {
			path, err := s.ipfs.Name().Resolve(ctx, key.Path().String())
			ipfsPath := "no backup found"
			if err != nil {
				ipfsPath = fmt.Sprintf("ERROR: %s", err)
			}
			if path != nil && path.String() != "" {
				ipfsPath = path.String()
			}

			results = append(results, &pb.ListBackupsReply_Backup{
				Key: &pb.Key{
					Name: key.Name(),
					Path: key.Path().String(),
				},
				IpfsPath: ipfsPath,
			})
		}
	}

	return &pb.ListBackupsReply{
		Backups: results,
	}, nil
}

func (s *Service) getIpnsKeyForDevice(ctx context.Context, deviceID idevice.DeviceID) (icore.Key, error) {
	keys, err := s.ipfs.Key().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get IPNS keys: %s", err)
	}

	deviceIDRaw := string(deviceID)
	for _, v := range keys {
		if v.Name() == deviceIDRaw {
			return v, nil
		}
	}

	return nil, nil
}

func (s *Service) createBackupIpnsKey(ctx context.Context, deviceID idevice.DeviceID) (icore.Key, error) {
	opts := []options.KeyGenerateOption{
		options.Key.Size(2048),
		options.Key.Type(options.RSAKey),
	}

	key, err := s.ipfs.Key().Generate(ctx, string(deviceID), opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate IPNS key: %s", err)
	}

	return key, nil
}

func (s *Service) addBackupToIpfs(ctx context.Context, backupDir string) (path.Path, error) {
	backupDirNode, err := getUnixfsNode(backupDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to find backup: %s", err)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Nocopy(true),
	}
	cidDirectory, err := s.ipfs.Unixfs().Add(ctx, backupDirNode, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to add backup to IPFS: %s", err)
	}

	return cidDirectory, nil
}

func (s *Service) updateLatestBackupIpns(ctx context.Context, backupIpfsPath path.Path, key icore.Key) (icore.IpnsEntry, error) {
	opts := []options.NamePublishOption{
		options.Name.AllowOffline(true),
		options.Name.Key(key.Name()),
	}

	ipnsEntry, err := s.ipfs.Name().Publish(ctx, backupIpfsPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to publish to IPNS: %s", err)
	}

	return ipnsEntry, nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}
