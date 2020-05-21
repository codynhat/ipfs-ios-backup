/*
Copyright © 2020 Cody Hatfield <cody.hatfield@me.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	logging "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/codynhat/ipfs-ios-backup/api"
	"google.golang.org/grpc"
)

var (
	cfgFile string
	client  *api.Client
	addr    ma.Multiaddr
	log     = logging.Logger("ipfs-ios-backup")
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ipfs-ios-backup",
	Short: "Backup iOS devices to IPFS",
	Long:  "Backup iOS devices to IPFS",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		var opts []grpc.DialOption
		addrAPI := viper.GetString("addrAPI")
		opts = append(opts, grpc.WithInsecure())
		var err error

		addr, err = ma.NewMultiaddr(addrAPI)
		if err != nil {
			log.Fatal(err)
		}

		ptarget, err := TcpAddrFromMultiAddr(addr)
		if err != nil {
			log.Fatal(err)
		}

		client, err = api.NewClient(ptarget, opts...)
		if err != nil {
			log.Fatal(err)
		}

		if viper.GetBool("debug") {
			if err := logging.SetLogLevel("ipfs-ios-backup", "debug"); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := logging.SetLogLevel("ipfs-ios-backup", "info"); err != nil {
				log.Fatal(err)
			}
		}
	},
	PersistentPostRun: func(c *cobra.Command, args []string) {
		if client != nil {
			if err := client.Close(); err != nil {
				log.Fatal(err)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	defaultRepoPath, err := homedir.Expand("~/.ipfs-ios-backup")
	if err != nil {
		log.Fatal(err)
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ipfs-ios-backup.json)")
	rootCmd.PersistentFlags().String("repoPath", defaultRepoPath, "Path to IPFS iOS Backup repo")
	viper.BindPFlag("repoPath", rootCmd.PersistentFlags().Lookup("repoPath"))

	rootCmd.PersistentFlags().String("addrAPI", "/ip4/127.0.0.1/tcp/3006", "API endpoint")
	viper.BindPFlag("addrAPI", rootCmd.PersistentFlags().Lookup("addrAPI"))

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".ipfs-ios-backup")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %v", viper.ConfigFileUsed())
	}
}
