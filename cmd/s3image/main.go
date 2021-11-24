package main

import (
	"context"
	"flag"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/je4/s3image/v2/pkg/filesystem"
	"github.com/je4/s3image/v2/pkg/server"
	lm "github.com/je4/utils/v2/pkg/logger"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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

	fs, err := filesystem.NewS3Fs(config.S3.Endpoint, config.S3.AccessKeyId, config.S3.SecretAccessKey, config.S3CacheExp.Duration, config.S3.UseSSL)
	if err != nil {
		logger.Fatalf("cannot conntct to s3 instance: %v", err)
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

	stat, err := os.Stat(config.CacheDir)
	if err != nil {
		logger.Panicf("cannot stat cache \"%s\"", config.CacheDir)
		return
	}
	if !stat.IsDir() {
		logger.Panicf("%s not a director", config.CacheDir)
		return
	}
	if config.ClearCacheOnStartup {
		logger.Infof("deleting cache files in %s", config.CacheDir)
		if len(config.CacheDir) < 4 {
			logger.Panicf("%s too short. will not clear cache", config.CacheDir)
			return
		}
		d, err := os.Open(config.CacheDir)
		if err != nil {
			logger.Panicf("cannot open directory %s", config.CacheDir)
			return
		}
		names, err := d.Readdirnames(-1)
		if err != nil {
			d.Close()
			logger.Panicf("cannot read %s", config.CacheDir)
			return
		}
		d.Close()
		for _, name := range names {
			fullpath := filepath.Join(config.CacheDir, name)
			logger.Infof("delete %s", fullpath)
			if err := os.Remove(fullpath); err != nil {
				logger.Panicf("cannot delete %s", fullpath)
				return
			}
		}
	}
	/*
		if err := os.RemoveAll(config.CacheDir); err != nil {
			log.Errorf("cannot remove %s: %v", config.CacheDir, err)
		}
	*/
	bconfig := badger.DefaultOptions(config.CacheDir)
	if runtime.GOOS == "windows" {
		// bconfig.Truncate = true
	}
	bconfig.Logger = logger
	db, err := badger.Open(bconfig)
	if err != nil {
		log.Panicf("cannot open badger database: %v", err)
		return
	}
	defer db.Close()

	srv, err := server.NewServer(config.ServiceName, config.Addr, config.AddrExt, config.UserName, config.Password, logger, accessLog, fs, db, config.Buckets)
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
