package api

import (
	"context"
	"fmt"

	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	"github.com/codynhat/ipfs-ios-backup/idevice"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
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
