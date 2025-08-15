// SPDX-License-Identifier: Apache-2.0
// SPDX-FileContributor: thedevop (J)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
	"xdas/internal/config"
	"xdas/internal/findx"
	"xdas/internal/keyspaces"
	"xdas/internal/logger"
	"xdas/internal/magicbyte"

	"github.com/thedevop1/jsoncr"
)

// Configuration holds all the config
type Configuration struct {
	// raw             []byte
	Verbose         bool
	NoMetrics       bool
	ValidateContent bool
	Web             *config.WebConfig
	HClient         *config.HClientConfig
	Redis           *config.RedisConfig
	Keyspaces       map[string]*KeyspaceConfig
	Multipart       struct {
		Keyspaces []string
	}
	DeviceMapping struct {
		TTL      string
		AccelTTL string
		ttl      time.Duration
		accelTTL time.Duration
	}
}

// KeyspaceConfig holds config for keyspace
type KeyspaceConfig struct {
	Input     KeyspaceFormat
	Store     KeyspaceFormat
	Output    KeyspaceFormat
	Kind      keyspaces.Kind
	FindX     *findx.FindX
	TTLString string `json:"ttl"`
	ttl       time.Duration
}

// KeyspaceFormat specifies the content-type and content-encoding for keyspace
type KeyspaceFormat struct {
	ContentType     string `json:"contentType"`
	ContentEncoding string `json:"contentEncoding"`
	magicByte       magicbyte.MagicByte
}

func getConfig(logger *logger.Logger) *Configuration {
	var (
		version    bool
		configFile = flag.String("config", os.Getenv("XX_CONFIG"), "The config filename, env: XX_CONFIG")
		verbose    = flag.Bool("verbose", false, "Turn on verbose logging")
	)

	flag.BoolVar(&version, "v", false, "Shows version and exit")
	flag.BoolVar(&version, "version", false, "Shows version and exit")
	flag.Parse()

	if version {
		fmt.Println(AppName, AppVersion, BuildTime)
		os.Exit(0)
	}

	config := &Configuration{
		Web:     config.NewWeb(),
		HClient: config.NewHClient(),
		Redis:   config.NewRedis(),
	}

	if *configFile != "" {
		file, err := os.Open(*configFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer file.Close()
		b, err := jsoncr.Remove(file)
		if err != nil {
			logger.Fatal(err)
		}
		if err = json.Unmarshal(b, &config); err != nil {
			logger.Fatal(err)
		}
		// config.raw = b
	}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "verbose" {
			config.Verbose = *verbose
		}
	})

	if err := config.Web.Validate(); err != nil {
		logger.Fatal("Web server config error", "err", err)
	}
	if err := config.HClient.Validate(); err != nil {
		logger.Fatal("HTTP client config error", "err", err)
	}
	if err := config.Redis.Validate(); err != nil {
		logger.Fatal("Redis config error", "err", err)
	}
	validateKeyspaceConfig(logger, config)
	validateDeviceMappingConfig(logger, config)

	logger.Info("Web server", "config", fmt.Sprint(config.Web.Server))
	logger.Info("HClient", "config", fmt.Sprint(config.HClient.Client.Transport))
	// logger.Println("Redis addr:", config.Redis.ClientConfig.Addrs)
	// logger.Println("Redis client config", config.Redis.ClientConfig)
	// logger.Println("Config:", config)
	// logger.Println("WebTLS Config:", config.WebTLS)

	return config
}

func validateDeviceMappingConfig(logger *logger.Logger, config *Configuration) {
	ttl, err := time.ParseDuration(config.DeviceMapping.TTL)
	if err != nil {
		logger.Info("Invalid DeviceMapping TTL", "ttl", config.DeviceMapping.TTL, "err", err)
		ttl = defaultDMTTL
	}
	config.DeviceMapping.ttl = ttl

	accelTTL, err := time.ParseDuration(config.DeviceMapping.AccelTTL)
	if err != nil {
		logger.Info("Invalid DeviceMapping AccelTTL", "ttl", config.DeviceMapping.AccelTTL, "err", err)
		accelTTL = defaultAccelDMTTL
	}
	config.DeviceMapping.accelTTL = accelTTL
}

func validateKeyspaceConfig(logger *logger.Logger, config *Configuration) {
	for key, value := range config.Keyspaces {
		value.Input.magicByte = magicbyte.New(value.Input.ContentEncoding, value.Input.ContentType, 0)

		if value.Store.ContentEncoding == "" {
			value.Store.ContentEncoding = value.Input.ContentEncoding
		}
		if value.Store.ContentType == "" {
			value.Store.ContentType = value.Input.ContentType
		}
		value.Store.magicByte = magicbyte.New(value.Store.ContentEncoding, value.Store.ContentType,
			config.Redis.Encryption) // ensure encryption at rest

		if value.Store.ContentType == "" { // can't have different output contentType when stored is unknown
			value.Output.ContentType = ""
		}
		if value.Output.ContentEncoding == "" {
			value.Output.ContentEncoding = value.Store.ContentEncoding
		}
		if value.Output.ContentType == "" {
			value.Output.ContentType = value.Store.ContentType
		}
		value.Output.magicByte = magicbyte.New(value.Output.ContentEncoding, value.Output.ContentType, 0)

		if value.FindX == nil {
			value.FindX = new(findx.FindX)
		}
		value.FindX.UserAgent = AppName

		ttl, err := time.ParseDuration(value.TTLString)
		if err != nil {
			logger.Fatal("KeyspaceConfig error, must have valid TTL", key, value, err)
		}
		value.ttl = ttl
	}
}
