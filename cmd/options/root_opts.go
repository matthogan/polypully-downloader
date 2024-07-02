package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootOptions struct {
	Config string
}

var _ Interface = (*RootOptions)(nil)

// AddFlags implements Interface
func (o *RootOptions) AddFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.PersistentFlags().StringVar(&o.Config, "config", "", "config file (default is config/application.yaml)")
	Load(cmd, v)
}
