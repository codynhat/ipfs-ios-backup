package api

import (
	"context"
	"fmt"
	"os"
	"time"

	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	"github.com/codynhat/ipfs-ios-backup/idevice"
	"github.com/golang/protobuf/ptypes"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type Backup struct {
	ID              core.InstanceID `json:"_id"` // DeviceID
	LatestBackupCid string
	UpdatedAt       time.Time
}

// Service is a gRPC service
type Service struct {
	ipfs             icore.CoreAPI
	d                *db.DB
	backupCollection *db.Collection
}

func NewService(ipfs icore.CoreAPI, d *db.DB) (*Service, error) {
	collection := d.GetCollection("Backup")
	return &Service{
		ipfs:             ipfs,
		d:                d,
		backupCollection: collection,
	}, nil
}

// AddBackup adds a new backup to IPFS
func (s *Service) AddBackup(ctx context.Context, req *pb.AddBackupRequest) (*pb.AddBackupReply, error) {
	backupCid, err := s.addBackupToIpfs(ctx, req.BackupDir)
	if err != nil {
		return nil, err
	}

	return &pb.AddBackupReply{
		BackupCid: backupCid.String(),
	}, nil
}

// UpdateLatestBackup saves a reference to the latest backup
func (s *Service) UpdateLatestBackup(ctx context.Context, req *pb.UpdateLatestBackupRequest) (*pb.UpdateLatestBackupReply, error) {
	deviceID := idevice.DeviceID(req.DeviceID)

	backup := &Backup{
		ID:              core.InstanceID(deviceID),
		LatestBackupCid: req.BackupCid,
		UpdatedAt:       time.Now(),
	}

	backupExists, err := s.backupExistsForDevice(deviceID)
	if err != nil {
		return nil, err
	}

	if backupExists {
		err := s.backupCollection.Save(util.JSONFromInstance(backup))
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.backupCollection.Create(util.JSONFromInstance(backup))
		if err != nil {
			return nil, err
		}
	}

	t, err := ptypes.TimestampProto(backup.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateLatestBackupReply{
		Backup: &pb.Backup{
			DeviceID:  backup.ID.String(),
			BackupCid: backup.LatestBackupCid,
			UpdatedAt: t,
		},
	}, nil
}

// ListBackups lists all known backups
func (s *Service) ListBackups(ctx context.Context, req *pb.ListBackupsRequest) (*pb.ListBackupsReply, error) {
	// Get all backups
	backups, err := s.backupCollection.Find(&db.Query{})
	if err != nil {
		return nil, err
	}

	var results []*pb.Backup
	for _, b := range backups {
		backup := &Backup{}
		util.InstanceFromJSON(b, backup)

		t, err := ptypes.TimestampProto(backup.UpdatedAt)
		if err != nil {
			return nil, err
		}

		results = append(results, &pb.Backup{
			DeviceID:  backup.ID.String(),
			BackupCid: backup.LatestBackupCid,
			UpdatedAt: t,
		})
	}

	return &pb.ListBackupsReply{
		Backups: results,
	}, nil
}

// Export returns the information needed to share backups with another device
func (s *Service) Export(ctx context.Context, req *pb.ExportRequest) (*pb.ExportReply, error) {
	addrs, key, err := s.d.GetDBInfo()
	if err != nil {
		return nil, err
	}

	var rawAddrs []string
	for _, addr := range addrs {
		rawAddrs = append(rawAddrs, addr.String())
	}

	return &pb.ExportReply{
		Addrs:     rawAddrs,
		ThreadKey: key.String(),
	}, nil
}

func (s *Service) addBackupToIpfs(ctx context.Context, backupDir string) (cid.Cid, error) {
	backupDirNode, err := getUnixfsNode(backupDir)
	if err != nil {
		return cid.Undef, fmt.Errorf("Failed to find backup: %s", err)
	}

	opts := []options.UnixfsAddOption{
		options.Unixfs.Pin(true),
		options.Unixfs.Nocopy(true),
	}
	cidDirectory, err := s.ipfs.Unixfs().Add(ctx, backupDirNode, opts...)
	if err != nil {
		return cid.Undef, fmt.Errorf("Failed to add backup to IPFS: %s", err)
	}

	return cidDirectory.Cid(), nil
}

func (s *Service) backupExistsForDevice(deviceID idevice.DeviceID) (bool, error) {
	backups, err := s.backupCollection.FindByID(core.InstanceID(deviceID))
	if err != nil && err != db.ErrNotFound {
		return false, err
	}

	if err == db.ErrNotFound {
		return false, nil
	}

	return len(backups) > 0, nil
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
