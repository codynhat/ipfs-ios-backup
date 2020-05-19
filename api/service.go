package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	pb "github.com/codynhat/ipfs-ios-backup/api/pb"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

// Service is a gRPC service
type Service struct{}

// Init initializes the repo
func (s *Service) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitReply, error) {
	err := handleInitReq(req.RepoPath)
	if err != nil {
		return nil, err
	}

	return &pb.InitReply{}, nil
}

func handleInitReq(repoPath string) error {
	_, err := os.Stat(repoPath)
	if err == nil {
		return fmt.Errorf("repo already exists at %s", repoPath)
	}

	// Create repo
	if err := checkWritable(repoPath); err != nil {
		return err
	}

	// Create backups dir
	backupDir := filepath.Join(repoPath, "backups")
	if err := checkWritable(backupDir); err != nil {
		return err
	}

	if err := initIpfsRepo(repoPath); err != nil {
		return err
	}

	fmt.Printf("Repo created at %s\n", repoPath)
	return nil
}

func initIpfsRepo(repoRoot string) error {
	ipfsRepoRoot := filepath.Join(repoRoot, ".ipfs")

	if err := checkWritable(ipfsRepoRoot); err != nil {
		return err
	}

	if fsrepo.IsInitialized(ipfsRepoRoot) {
		return fmt.Errorf("IPFS Repo already exists at %s", ipfsRepoRoot)
	}

	// Setup plugins
	if err := setupPlugins(ipfsRepoRoot); err != nil {
		return fmt.Errorf("Failed to setup plugins: %s", err)
	}

	// Create config
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return fmt.Errorf("Could not create config: %s", err)
	}

	customizeConfig(cfg)

	// Create the repo with the config
	err = fsrepo.Init(ipfsRepoRoot, cfg)
	if err != nil {
		return fmt.Errorf("Could not initialize repo: %s", err)
	}

	createSwarmKey(ipfsRepoRoot)

	return nil
}

func customizeConfig(cfg *config.Config) {
	// Remove Bootstrap list
	cfg.Bootstrap = make([]string, 0)

	// Enable Filestore
	cfg.Experimental.FilestoreEnabled = true
}

// Taken from go-ipfs example
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// Taken from go-ipfs init
func checkWritable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// dir exists, make sure we can write to it
		testfile := filepath.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// dir doesn't exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("cannot write to %s, incorrect permissions", err)
	}

	return err
}

// See https://github.com/Kubuxu/go-ipfs-swarm-key-gen
func createSwarmKey(ipfsRepoRoot string) error {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return fmt.Errorf("While trying to create swarm key: %s", err)
	}

	encodedKey := hex.EncodeToString(key)
	swarmKeyPath := filepath.Join(ipfsRepoRoot, "swarm.key")
	err = ioutil.WriteFile(swarmKeyPath, []byte(fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base16/\n%s\n", encodedKey)), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write swarm key: %s", err)
	}

	return nil
}
