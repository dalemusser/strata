package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

// Config structure
type Config struct {
	Host                string `json:"host" yaml:"host" toml:"host"`
	Port                string `json:"port" yaml:"port" toml:"port"`
	SSLEnv              string `json:"ssl_env" yaml:"ssl_env" toml:"ssl_env"`
	CertFile            string `json:"cert_file" yaml:"cert_file" toml:"cert_file"`
	KeyFile             string `json:"key_file" yaml:"key_file" toml:"key_file"`
	LogFile             string `json:"log_file" yaml:"log_file" toml:"log_file"`
	StaticDir           string `json:"static_dir" yaml:"static_dir" toml:"static_dir"`
	ShutdownTimeout     int    `json:"shutdown_timeout" yaml:"shutdown_timeout" toml:"shutdown_timeout"`
	CloudFrontURL       string `json:"cloudfront_url" yaml:"cloudfront_url" toml:"cloudfront_url"`
	CloudWatchLogGroup  string `json:"cloudwatch_log_group" yaml:"cloudwatch_log_group" toml:"cloudwatch_log_group"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream" yaml:"cloudwatch_log_stream" toml:"cloudwatch_log_stream"`
	AWSRegion           string `json:"aws_region" yaml:"aws_region" toml:"aws_region"`
	UseTLS              bool   `json:"use_tls" yaml:"use_tls" toml:"use_tls"`
	UseLetsEncrypt      bool   `json:"use_lets_encrypt" yaml:"use_lets_encrypt" toml:"use_lets_encrypt"`
}

var defaultConfig = Config{
	Host:                "localhost",
	Port:                "8080",
	SSLEnv:              "prod",
	StaticDir:           "./static",
	LogFile:             "server.log",
	ShutdownTimeout:     30,
	CloudFrontURL:       "",
	CloudWatchLogGroup:  "default-log-group",
	CloudWatchLogStream: "default-log-stream",
	AWSRegion:           "us-west-2",
	UseTLS:              true,
	UseLetsEncrypt:      false,
}

// LoadConfig loads configuration from file, environment variables, and command-line arguments.
func LoadConfig() *Config {
	hostFlag := flag.String("host", "", "The host on which the server will run")
	portFlag := flag.String("port", "", "The port on which the server will run")
	configFileFlag := flag.String("config", "", "Path to the config file")
	flag.Parse()

	config := &defaultConfig
	if *configFileFlag != "" {
		loadConfigFromFile(config, *configFileFlag)
	}

	overrideWithEnvVars(config)

	if *hostFlag != "" {
		config.Host = *hostFlag
	}
	if *portFlag != "" {
		config.Port = *portFlag
	}
	return config
}

// loadConfigFromFile loads config from YAML, JSON, or TOML file.
func loadConfigFromFile(config *Config, path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Println("Unable to load config file:", err)
		return
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err == nil {
		return
	}
	_, err = toml.DecodeFile(path, config)
	if err == nil {
		return
	}
	err = yaml.NewDecoder(file).Decode(config)
	if err != nil {
		log.Println("Unable to parse config file:", err)
	}
}

// overrideWithEnvVars overrides config with environment variables.
func overrideWithEnvVars(config *Config) {
	if val := os.Getenv("STRATA_HOST"); val != "" {
		config.Host = val
	}
	if val := os.Getenv("STRATA_PORT"); val != "" {
		config.Port = val
	}
	if val := os.Getenv("STRATA_SSL_ENV"); val != "" {
		config.SSLEnv = val
	}
	if val := os.Getenv("STRATA_CERT_FILE"); val != "" {
		config.CertFile = val
	}
	if val := os.Getenv("STRATA_KEY_FILE"); val != "" {
		config.KeyFile = val
	}
	if val := os.Getenv("STRATA_LOG_FILE"); val != "" {
		config.LogFile = val
	}
	if val := os.Getenv("STRATA_STATIC_DIR"); val != "" {
		config.StaticDir = val
	}
	if val := os.Getenv("STRATA_CLOUDFRONT_URL"); val != "" {
		config.CloudFrontURL = val
	}
	if val := os.Getenv("STRATA_CLOUDWATCH_LOG_GROUP"); val != "" {
		config.CloudWatchLogGroup = val
	}
	if val := os.Getenv("STRATA_CLOUDWATCH_LOG_STREAM"); val != "" {
		config.CloudWatchLogStream = val
	}
	if val := os.Getenv("STRATA_AWS_REGION"); val != "" {
		config.AWSRegion = val
	}
	if val := os.Getenv("STRATA_USE_TLS"); val == "false" {
		config.UseTLS = false
	}
	if val := os.Getenv("STRATA_USE_LETS_ENCRYPT"); val == "true" {
		config.UseLetsEncrypt = true
	}
}
