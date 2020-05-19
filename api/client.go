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
