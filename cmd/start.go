package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codejago/polypully/downloader/api/generated/openapi"
	"github.com/codejago/polypully/downloader/cmd/options"
	"github.com/codejago/polypully/downloader/internal/app/service"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"golang.org/x/exp/slog"
)

func StartCmd() *cobra.Command {
	o := &options.StartOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start downloader service",
		Run: func(cmd *cobra.Command, args []string) {

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigs
				slog.Info("Shutting down...", "signal", sig)
				os.Exit(0)
			}()

			slog.Info("server starting", "port", o.Port)
			defaultApiService := service.NewApiService()
			defaultApiController := openapi.NewDefaultApiController(defaultApiService)
			router := openapi.NewRouter(defaultApiController)

			if err := http.ListenAndServe(fmt.Sprintf(":%d", o.Port), router); err != nil {
				slog.Error("Failed to start server", err)
				os.Exit(-1)
			}
		},
	}
	o.AddFlags(cmd, viper.GetViper())
	return cmd
}
