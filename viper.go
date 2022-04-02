package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"

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
	EXCLUDE      = "exclude"
	INCLUDE      = "include"
	LOG_LEVEL    = "LOG_LEVEL"
)

func InitConfig() {
	viper.SetDefault(LOG_LEVEL, "info")
	viper.SetDefault(GITLAB_HOST, "https://gitlab.com")
	viper.SetDefault(GTILAB_TOKEN, "")

	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.AddConfigPath(home)
	viper.SetConfigType(configFileType)
	viper.SetConfigName(configFileName)

	viper.AutomaticEnv()
	setLogLevel()

	usedConfigFile := getConfigFilename(home)
	if usedConfigFile != "" {
		cleartext, err := decrypt.File(usedConfigFile, configFileType)

		if err != nil {
			logger.Warnf("Error decrypting. %s. Maybe you're not using an encrypted config?", err)
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
	} else {
		logger.Debug("No config file used!")
	}

	clinar.ExcludeFilter = viper.GetStringSlice("exclude")

	if viper.GetString(INCLUDE) != "" {
		rex, err := regexp.Compile(viper.GetString(INCLUDE))
		if err != nil {
			logger.Fatal(err)
		}
		clinar.IncludePattern = rex
	}
}

func getConfigFilename(homedir string) string {
	pathWithoutExt := path.Join(homedir, configFileName)
	logger.Debugf("Check if %s exists", pathWithoutExt)
	if _, err := os.Stat(pathWithoutExt); err == nil {
		return pathWithoutExt
	}

	pathWithExt := fmt.Sprintf("%s.%s", pathWithoutExt, configFileType)
	logger.Debugf("Check if %s exists", pathWithExt)
	if _, err := os.Stat(pathWithExt); err == nil {
		return pathWithExt
	}
	pathWithExt = fmt.Sprintf("%s.%s", pathWithoutExt, "yml")
	logger.Debugf("Check if %s exists", pathWithExt)
	if _, err := os.Stat(pathWithExt); err == nil {
		return pathWithExt
	}
	return ""
}

func setLogLevel() {
	lvl, err := logger.ParseLevel(viper.GetString(LOG_LEVEL))
	if err == nil {
		logger.SetLevel(lvl)
	} else {
		logger.Errorf("setLogLevel: %v", err)
	}
}
