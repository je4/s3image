package main

import (
	"context"
	"flag"
	"github.com/je4/s3image/v2/pkg/filesystem"
	"github.com/je4/s3image/v2/pkg/server"
	lm "github.com/je4/utils/v2/pkg/logger"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfgFile := flag.String("cfg", "/etc/s3image.toml", "locations of config file")
	flag.Parse()
	config := LoadConfig(*cfgFile)

	// create logger instance
	logger, lf := lm.CreateLogger("S3Image", config.Logfile, nil, config.Loglevel, config.Logformat)
	defer lf.Close()

	fs, err := filesystem.NewS3Fs(config.S3.Endpoint, config.S3.AccessKeyId, config.S3.SecretAccessKey, config.S3.UseSSL)
	if err != nil {
		log.Fatalf("cannot conntct to s3 instance: %v", err)
	}
	var accessLog io.Writer
	var f *os.File
	if config.AccessLog == "" {
		accessLog = os.Stdout
	} else {
		f, err = os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			logger.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accessLog = f
	}

	srv, err := server.NewServer(config.ServiceName, config.Addr, config.UserName, config.Password, logger, accessLog, fs)
	if err != nil {
		logger.Panicf("cannot start server: %v", err)
	}

	go func() {
		logger.Infof("server starting at %s - %s", config.Addr, config.AddrExt)
		if err := srv.ListenAndServe(config.CertPEM, config.KeyPEM); err != nil {
			logger.Fatalf("server died: %v", err)
		}
	}()

	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		logger.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	logger.Info("server stopped")

}
