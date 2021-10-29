/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
	"github.com/slashdevops/idp-scim-sync/internal/config"
	"github.com/slashdevops/idp-scim-sync/internal/version"
	"github.com/slashdevops/idp-scim-sync/pkg/aws"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var cfg config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "idpscim",
	Version: version.Version,
	Short:   "Sync your AWS Single Sing-On (SSO) with Google Workspace",
	Long: `
Sync your Google Workspace Groups and Users to AWS Single Sing-On using
AWS SSO SCIM API (https://docs.aws.amazon.com/singlesignon/latest/developerguide/what-is-scim.html).`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if cfg.IsLambda {
		lambda.Start(rootCmd.Execute)
	}
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cfg = config.New()
	cfg.IsLambda = len(os.Getenv("_LAMBDA_SERVER_PORT")) > 0

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&cfg.Debug, "debug", "d", config.DefaultDebug, "enable log debug level")
	rootCmd.PersistentFlags().StringVarP(&cfg.LogFormat, "log-format", "f", config.DefaultLogFormat, "set the log format")
	rootCmd.PersistentFlags().StringVarP(&cfg.LogLevel, "log-level", "l", config.DefaultLogLevel, "set the log level")

	rootCmd.PersistentFlags().StringVarP(&cfg.SCIMAccessToken, "aws-scim-access-token", "t", "", "AWS SSO SCIM API Access Token")
	rootCmd.MarkPersistentFlagRequired("aws-scim-access-token")

	rootCmd.PersistentFlags().StringVarP(&cfg.SCIMEndpoint, "aws-scim-endpoint", "e", "", "AWS SSO SCIM API Endpoint")
	rootCmd.MarkPersistentFlagRequired("aws-scim-endpoint")

	rootCmd.PersistentFlags().StringVarP(&cfg.GWSServiceAccountFile, "gws-service-account-file", "s", config.DefaultGWSServiceAccountFile, "path to Google Workspace service account file")
	rootCmd.MarkPersistentFlagRequired("gws-service-account-file")

	rootCmd.PersistentFlags().StringVarP(&cfg.GWSUserEmail, "gws-user-email", "u", "", "Google Workspace user email with allowed access to the Google Workspace Service Account")
	rootCmd.MarkPersistentFlagRequired("gws-user-email")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("idpscim") // allow to read in from environment
	viper.AutomaticEnv()          // read in environment variables that match

	envVars := []string{
		"gws_user_email",
		"gws_service_account_file",
		"scim_access_token",
		"scim_endpoint",
		"log_level",
		"log_format",
		"sync_method",
		"gws_service_account_file_secret_name",
		"gws_user_email_secret_name",
		"scim_endpoint_secret_name",
		"scim_access_token_secret_name",
	}

	for _, e := range envVars {
		if err := viper.BindEnv(e); err != nil {
			log.Fatalf(errors.Wrap(err, "cannot bind environment variable").Error())
		}
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf(errors.Wrap(err, "cannot unmarshal config").Error())
	}

	switch cfg.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	default:
		log.Warnf("unknown log format: %s, using text", cfg.LogFormat)
		log.SetFormatter(&log.TextFormatter{})
	}

	if cfg.Debug {
		cfg.LogLevel = "debug"
	}

	// set the configured log level
	if level, err := log.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(level)
	}

	if cfg.IsLambda {
		getSecrets()
	}
}

func getSecrets() {
	awsconf, err := awsconf.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot load aws config").Error())
	}

	svc := secretsmanager.NewFromConfig(awsconf)

	secrets, err := aws.NewSecretsManagerService(svc)
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot create aws secrets manager service").Error())
	}

	unwrap, err := secrets.GetSecretValue(context.Background(), cfg.GWSUserEmailSecretName)
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot get secretmanager value").Error())
	}
	cfg.GWSUserEmail = unwrap

	unwrap, err = secrets.GetSecretValue(context.Background(), cfg.GWSServiceAccountFileSecretName)
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot get secretmanager value").Error())
	}
	cfg.GWSServiceAccountFile = unwrap

	unwrap, err = secrets.GetSecretValue(context.Background(), cfg.SCIMAccessTokenSecretName)
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot get secretmanager value").Error())
	}
	cfg.SCIMAccessToken = unwrap

	unwrap, err = secrets.GetSecretValue(context.Background(), cfg.SCIMEndpointSecretName)
	if err != nil {
		log.Fatalf(errors.Wrap(err, "cannot get secretmanager value").Error())
	}
	cfg.SCIMEndpoint = unwrap
}
