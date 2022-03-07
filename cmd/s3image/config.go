package main

import (
	"github.com/BurntSushi/toml"
	"github.com/je4/zsearch/v2/configdata"
	"log"
	"os"
)

type LocalFS struct {
	Path string `toml:"path"`
}

type Config struct {
	ServiceName         string              `toml:"servicename"`
	Logfile             string              `toml:"logfile"`
	Loglevel            string              `toml:"loglevel"`
	Logformat           string              `toml:"logformat"`
	AccessLog           string              `toml:"accesslog"`
	Addr                string              `toml:"addr"`
	AddrExt             string              `toml:"addrext"`
	CertPEM             string              `toml:"certpem"`
	KeyPEM              string              `toml:"keypem"`
	Buckets             map[string]string   `toml:"buckets"`
	UserName            string              `toml:"username"`
	Password            string              `toml:"password"`
	S3                  configdata.CfgS3    `toml:"s3"`
	S3CacheExp          configdata.Duration `toml:"s3cacheexp"`
	CacheDir            string              `toml:"cachedir"`
	Templates           map[string]string   `toml:"template"`
	ClearCacheOnStartup bool                `toml:"clearcacheonstartup"`
	Filesystem          string              `toml:"filesystem"`
	Local               LocalFS             `toml:"local"`
}

func LoadConfig(filepath string) Config {
	var conf Config
	conf.Logformat = "%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}"
	conf.Filesystem = "s3"
	_, err := toml.DecodeFile(filepath, &conf)
	if err != nil {
		log.Fatalln("Error on loading config: ", err)
	}

	clearcache := os.Getenv("S3IMAGE_CLEARCACHE")
	switch clearcache {
	case "true":
		conf.ClearCacheOnStartup = true
	case "false":
		conf.ClearCacheOnStartup = false
	}

	return conf
}
