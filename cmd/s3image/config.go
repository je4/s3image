package s3image

import (
	"github.com/BurntSushi/toml"
	"github.com/je4/zsearch/v2/configdata"
	"log"
)

type Config struct {
	ServiceName string           `toml:"servicename"`
	Logfile     string           `toml:"logfile"`
	Loglevel    string           `toml:"loglevel"`
	Logformat   string           `toml:"logformat"`
	AccessLog   string           `toml:"accesslog"`
	Addr        string           `toml:"addr"`
	AddrExt     string           `toml:"addrext"`
	CertPEM     string           `toml:"certpem"`
	KeyPEM      string           `toml:"keypem"`
	UserName    string           `toml:"username"`
	Password    string           `toml:"password"`
	S3          configdata.CfgS3 `toml:"s3"`
}

func LoadConfig(filepath string) Config {
	var conf Config
	_, err := toml.DecodeFile(filepath, &conf)
	if err != nil {
		log.Fatalln("Error on loading config: ", err)
	}
	return conf
}