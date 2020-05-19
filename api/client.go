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

// Init initializes the repo
func (c *Client) Init(ctx context.Context, repoPath string) (*pb.InitReply, error) {
	return c.c.Init(ctx, &pb.InitRequest{
		RepoPath: repoPath,
	})
}
