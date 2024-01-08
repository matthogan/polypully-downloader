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
	cmd.Flags().IntVarP(&o.Port, "port", "p", 1944, "port to listen on")
	cmd.Flags().StringVar(&o.Ip, "ip", "127.0.0.1", "IP address to listen on")
	cmd.Flags().StringVar(&o.ServerCert, "server-cert", "", "location of server certificate to encrypt communications")
	cmd.Flags().IntVar(&o.MaxConcurrentDownloads, "max-concurrent-downloads", 1, "max concurrent downloads")
	cmd.Flags().StringVar(&o.DownloadDirectory, "download-directory", "/tmp", "directory to store temporary downloads")
	cmd.Flags().StringVar(&o.MetadataDirectory, "metadata-directory", "/tmp", "directory to store metadata")
	cmd.Flags().StringVar(&o.LogDirectory, "log-directory", "/tmp", "log directory")
	cmd.Flags().StringVar(&o.LogConfigFile, "log-config-file", "", "location of logging configuration")
	Load(cmd, v)
}
