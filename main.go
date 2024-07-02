package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/codejago/polypully/downloader/cmd"
	"github.com/codejago/polypully/downloader/cmd/options"
)

var (
	co = &cmd.Configuration{}
)

func init() {
	defer finally()
}

func main() {

	defer finally()

	rootCmd := &cobra.Command{
		Use:   "downloader",
		Short: "polypully downloader service",
	}

	o := &options.RootOptions{} // args
	o.AddFlags(rootCmd, viper.GetViper())

	rootCmd.AddCommand(cmd.StartCmd())
	rootCmd.AddCommand(cmd.Config())

	co.Load()

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

// catch panics so stack traces are not output
func finally() {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
