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
	viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))

	cmd.Flags().StringVarP(&o.Ip, "ip", "i", "127.0.0.1", "IP address to listen on")
	viper.BindPFlag("server.ip", cmd.Flags().Lookup("ip"))

	cmd.Flags().StringVarP(&o.ServerCert, "cert", "c", "", "location of server certificate to encrypt communications")
	viper.BindPFlag("server.cert", cmd.Flags().Lookup("cert"))

	cmd.Flags().IntVarP(&o.MaxConcurrentDownloads, "max-conc", "m", 1, "max concurrent downloads")
	viper.BindPFlag("download.max_concurrent", cmd.Flags().Lookup("max-conc"))

	cmd.Flags().StringVarP(&o.DownloadDirectory, "dir", "d", "/tmp", "directory to store temporary downloads")
	viper.BindPFlag("download.directory", cmd.Flags().Lookup("dir"))

	Load(cmd, v)
}
