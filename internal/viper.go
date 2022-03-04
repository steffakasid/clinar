package internal

import (
	"bytes"
	"os"
	"path"

	logger "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mozilla.org/sops/decrypt"
)

const (
	configFileType = "yaml"
	configFileName = ".clinar"
)

const (
	GITLAB_HOST  = "GITLAB_HOST"
	GTILAB_TOKEN = "GITLAB_TOKEN"
	APPROVE      = "approve"
	INCLUDE      = "include"
	LOG_LEVEL    = "LOG_LEVEL"
)

func init() {
	viper.SetDefault(LOG_LEVEL, "info")
	viper.SetDefault(GITLAB_HOST, "https://gitlab.com")
	viper.SetDefault(GTILAB_TOKEN, "")
}

func InitConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.AddConfigPath(home)
	viper.SetConfigType(configFileType)
	viper.SetConfigName(configFileName)

	viper.AutomaticEnv()

	cleartext, err := decrypt.File(path.Join(home, configFileName), configFileType)

	if err != nil {
		logger.Warnf("Error encrypting. %s. Maybe you're not using an encrypted config?", err)
		if err := viper.ReadInConfig(); err != nil {
			logger.Warnf("Error reading config. %s. Are you using a config?", err)
		} else {
			setLogLevel()
			logger.Debug("Using config file:", viper.ConfigFileUsed())
		}
	} else {
		if err := viper.ReadConfig(bytes.NewBuffer(cleartext)); err != nil {
			logger.Fatal(err)
		} else {
			setLogLevel()
			logger.Debug("Using sops encrypted config file:", viper.ConfigFileUsed())
		}
	}
}

func setLogLevel() {
	lvl, err := logger.ParseLevel(viper.GetString(LOG_LEVEL))
	if err == nil {
		logger.SetLevel(lvl)
	} else {
		logger.Error(err)
	}
}
