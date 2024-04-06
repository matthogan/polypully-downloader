package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type StartOptions struct {
	Port                   int
	Ip                     string
	ServerCert             string
	MaxConcurrentDownloads int
	DownloadDirectory      string
	MetadataDirectory      string
	LogDirectory           string
	LogConfigFile          string
}

var _ Interface = (*StartOptions)(nil)

// AddFlags implements Interface
func (o *StartOptions) AddFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().IntVarP(&o.Port, "port", "p", 8080, "port to listen on")
	cmd.Flags().StringVarP(&o.Ip, "nic", "n", "127.0.0.1", "IP address to listen on")
	cmd.Flags().StringVarP(&o.ServerCert, "server-cert", "s", "", "location of server certificate to encrypt communications")
	cmd.Flags().IntVarP(&o.MaxConcurrentDownloads, "max-concurrent-downloads", "m", 1, "max concurrent downloads")
	cmd.Flags().StringVarP(&o.DownloadDirectory, "download-directory", "d", "/tmp", "directory to store temporary downloads")
	Load(cmd, v)
}
