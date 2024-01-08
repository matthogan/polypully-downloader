package options

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Interface interface {
	// AddFlags adds this options' flags to the cobra command.
	AddFlags(cmd *cobra.Command, v *viper.Viper)
}

// iterates over the cmd flags and sets the viper values
func Load(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		viper.Set(flag.Name, flag.Value)
	})
}
