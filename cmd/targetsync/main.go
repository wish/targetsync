package main

import (
	"context"
	"os"
	"strings"

	targetsync "github.com/jacksontj/targetSync"
	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

var opts struct {
	ConfigFile string `long:"config" description:"path to the config file" required:"true"`
	LogLevel   string `long:"log-level" description:"Log level" default:"info"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		// If the error was from the parser, then we can simply return
		// as Parse() prints the error already
		if _, ok := err.(*flags.Error); ok {
			os.Exit(1)
		}
		logrus.Fatalf("Error parsing flags: %v", err)
	}

	// Use log level
	level := logrus.InfoLevel
	switch strings.ToLower(opts.LogLevel) {
	case "panic":
		level = logrus.PanicLevel
	case "fatal":
		level = logrus.FatalLevel
	case "error":
		level = logrus.ErrorLevel
	case "warn":
		level = logrus.WarnLevel
	case "info":
		level = logrus.InfoLevel
	case "debug":
		level = logrus.DebugLevel
	default:
		logrus.Fatalf("Unknown log level: %s", opts.LogLevel)
	}
	logrus.SetLevel(level)

	// Set the log format to have a reasonable timestamp
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logrus.SetFormatter(formatter)

	// Create base context for this daemon
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load config
	cfg, err := targetsync.ConfigFromFile(opts.ConfigFile)
	if err != nil {
		logrus.Fatalf("Unable to load config: %v", err)
	}

	// Create syncer
	src, err := targetsync.NewConsulSource(&cfg.ConsulConfig)
	if err != nil {
		logrus.Fatalf("Error creating consul source: %v", err)
	}

	dst, err := targetsync.NewAWSTargetGroup(&cfg.AWSConfig)
	if err != nil {
		logrus.Fatalf("Error creating aws dest: %v", err)
	}

	syncer := &targetsync.Syncer{
		Config: &cfg.SyncConfig,
		Locker: src,
		Src:    src,
		Dst:    dst,
	}

	// Run
	if err := syncer.Run(ctx); err != nil {
		logrus.Errorf("Error starting targetSync: %v", err)
	}
}
