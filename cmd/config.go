package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	log "golang.org/x/exp/slog"
)

// Load the application properties following
// the most used spring boot options. Does
// not merge.

var (
	envVar envy
)

func init() {
	envVar = envyVars{}
}

type envy interface {
	Getenv(key string) string
	GetenvSlice(key string, delim string) []string
}

type envyVars struct{}

func (e envyVars) Getenv(key string) string {
	return os.Getenv(key)
}

func (e envyVars) GetenvSlice(key string, delim string) []string {
	values := e.Getenv(key)
	if values != "" {
		return strings.Split(values, delim)
	}
	return []string{}
}

func Config() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "dump the application config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if viper.ConfigFileUsed() != "" {
				log.Debug("Detected a config file", "ConfigFileUsed", viper.ConfigFileUsed())
			}
			log.Info("All Properties:")
			for key, value := range viper.AllSettings() {
				log.Info("property", key, value)
			}
			return nil
		},
	}
	return cmd
}

type Configuration struct {
}

func (o *Configuration) Load() {
	log.Debug("Load config")
	profiles := envVar.GetenvSlice("ACTIVE_PROFILES", ",")
	location := envVar.Getenv("CONFIG_LOCATION")
	filetype := envVar.Getenv("CONFIG_FILETYPE")
	if err := o.Init(profiles, location, filetype); err != nil {
		log.Error("error while reading configuration %v", err)
		panic(err)
	}
}

func (o *Configuration) Init(profiles []string, location string, fileType string) error {
	log.Debug("Init config")
	var name = "application"
	fileType, err := readFileType(fileType)
	if err != nil {
		return err
	}
	viper.SetConfigType(fileType)
	var locations = [4]string{location, "$HOME/", "./config/", "./"}
	for _, l := range locations {
		viper.AddConfigPath(l) // handles nil
	}
	log.Debug(fmt.Sprintf("Config file search parameters are name: %s, fileType: %s, location: %s",
		name, fileType, location))
	log.Debug(fmt.Sprintf("Searching in the following locations: %s", locations))
	if err = readInConfig(name, ""); err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Searching for profiles: %s", profiles))
	for _, profile := range profiles {
		err := readInConfig(name, profile)
		if err != nil {
			return err
		}
	}
	return nil
}

func readFileType(fileType string) (string, error) {
	if fileType == "" {
		return "yaml", nil
	}
	if fileType != "yaml" && fileType != "json" {
		return "", fmt.Errorf("invalid file type %s must be yaml or json", fileType)
	}
	return fileType, nil
}

func readInConfig(name string, profile string) error {
	if profile != "" {
		name += "-" + profile
	}
	viper.SetConfigName(name)
	if err := viper.MergeInConfig(); err != nil { // merge
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug(fmt.Sprintf("Config file %s not found. Continuing...", name))
			return nil
		} else {
			return err
		}
	}
	log.Debug(fmt.Sprintf("Config file %s.yaml found", name))
	return nil
}
