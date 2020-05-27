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

// AddBackup adds a new backup to IPFS
func (c *Client) AddBackup(ctx context.Context, backupDir string) (*pb.AddBackupReply, error) {
	return c.c.AddBackup(ctx, &pb.AddBackupRequest{
		BackupDir: backupDir,
	})
}

// UpdateLatestBackup saves a reference to the latest backup
func (c *Client) UpdateLatestBackup(ctx context.Context, deviceID string, backupCid string) (*pb.UpdateLatestBackupReply, error) {
	return c.c.UpdateLatestBackup(ctx, &pb.UpdateLatestBackupRequest{
		DeviceID:  deviceID,
		BackupCid: backupCid,
	})
}

// ListBackups lists all known backups
func (c *Client) ListBackups(ctx context.Context) (*pb.ListBackupsReply, error) {
	return c.c.ListBackups(ctx, &pb.ListBackupsRequest{})
}
