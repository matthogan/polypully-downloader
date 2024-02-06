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
	appevents "github.com/matthogan/polypully-events"

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

			// handle shutdown signals
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigs
				slog.Info("shutting down...", "signal", sig)
				os.Exit(0)
			}()

			// init the event producer
			events, err := appevents.NewEvents(&appevents.EventsConfig{
				BootstrapServers: viper.GetString("kafka_bootstrap_servers"),
				ClientId:         viper.GetString("kafka_client_id"),
				Acks:             viper.GetString("kafka_acks"),
				Topic:            viper.GetString("kafka_topic"),
				ProducerId:       viper.GetString("kafka_producer_id"),
				Config:           viper.GetStringMapString("kafka_config")})
			if err != nil {
				slog.Error("failed to create the event producer", "error", err)
				os.Exit(-1)
			}
			events.Notify(appevents.NewServiceEvent("started"))

			// init the service
			slog.Info("server starting", "port", o.Port)
			defaultApiService := service.NewApiService(events)
			defaultApiController := openapi.NewDefaultApiController(defaultApiService)
			router := openapi.NewRouter(defaultApiController)

			// start the server
			if err := http.ListenAndServe(fmt.Sprintf(":%d", o.Port), router); err != nil {
				events.Notify(appevents.NewServiceEvent("start failed"))
				slog.Error("failed to start server", err)
				os.Exit(-1)
			}
			events.Notify(appevents.NewServiceEvent("stopped"))
		},
	}
	o.AddFlags(cmd, viper.GetViper())
	return cmd
}
