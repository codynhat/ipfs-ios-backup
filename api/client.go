package api

import (
	"context"

	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c    pb.APIClient
	conn *grpc.ClientConn
}

// NewClient starts the client.
func NewClient(target string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		c:    pb.NewAPIClient(conn),
		conn: conn,
	}, nil
}

// Close closes the client's grpc connection and cancels any active requests.
func (c *Client) Close() error {
	return c.conn.Close()
}

// GetKeyForDevice fetches the IPNS key for a device
func (c *Client) GetKeyForDevice(ctx context.Context, deviceID string) (*pb.GetKeyForDeviceReply, error) {
	return c.c.GetKeyForDevice(ctx, &pb.GetKeyForDeviceRequest{
		DeviceID: deviceID,
	})
}

// CreateKeyForDevice creates a new IPNS key for a device
func (c *Client) CreateKeyForDevice(ctx context.Context, deviceID string) (*pb.CreateKeyForDeviceReply, error) {
	return c.c.CreateKeyForDevice(ctx, &pb.CreateKeyForDeviceRequest{
		DeviceID: deviceID,
	})
}

// AddBackup adds a new backup to IPFS
func (c *Client) AddBackup(ctx context.Context, backupDir string) (*pb.AddBackupReply, error) {
	return c.c.AddBackup(ctx, &pb.AddBackupRequest{
		BackupDir: backupDir,
	})
}

// UpdateLatestBackup saves a reference to the latest backup
func (c *Client) UpdateLatestBackup(ctx context.Context, deviceID string, backupPath string) (*pb.UpdateLatestBackupReply, error) {
	return c.c.UpdateLatestBackup(ctx, &pb.UpdateLatestBackupRequest{
		DeviceID:   deviceID,
		BackupPath: backupPath,
	})
}

// ListBackups lists all known backups
func (c *Client) ListBackups(ctx context.Context) (*pb.ListBackupsReply, error) {
	return c.c.ListBackups(ctx, &pb.ListBackupsRequest{})
}
