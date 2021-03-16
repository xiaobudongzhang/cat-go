package cat

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	domain        string
	hostname      string
	env           string
	ip            string
	ipHex         string
	baseLogDir    string
	serverAddress []serverAddress
}

type XMLConfig struct {
	Name       xml.Name         `xml:"config"`
	BaseLogDir string           `xml:"base-log-dir"`
	Servers    XMLConfigServers `xml:"servers"`
}

type XMLConfigServers struct {
	Servers []XMLConfigServer `xml:"server"`
}

type XMLConfigServer struct {
	Host     string `xml:"ip,attr"`
	Port     int    `xml:"port,attr"`
	HttpPort int    `xml:"http-port,attr"`
}

var config = Config{
	domain:        defaultAppKey,
	hostname:      defaultHostname,
	env:           defaultEnv,
	ip:            defaultIp,
	ipHex:         defaultIpHex,
	baseLogDir:    defaultLogDir,
	serverAddress: []serverAddress{},
}

func loadConfigFromLocalFile(filename string) (data []byte, err error) {
	file, err := os.Open(filename)
	if err != nil {
		logger.Warning("Unable to open file `%s`.", filename)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warning("Cannot close local client.xml file.")
		}
	}()

	data, err = ioutil.ReadAll(file)
	if err != nil {
		logger.Warning("Unable to read content from file `%s`", filename)
	}
	return
}

func loadConfig(location string) (data []byte, err error) {
	if len(location) == 0 {
		location = defaultXmlFile
	}
	if data, err = loadConfigFromLocalFile(location); err != nil {
		logger.Error("Failed to load local config file.")
		return
	}
	return
}

func parseXMLConfig(data []byte) (err error) {
	c := XMLConfig{}
	err = xml.Unmarshal(data, &c)
	if err != nil {
		logger.Warning("Failed to parse xml content")
	}

	if err = loadXmlConfig(c); err != nil {
		return
	}

	return
}

func loadXmlConfig(c XMLConfig) (err error) {
	if len(c.BaseLogDir) > 0 {
		config.baseLogDir = c.BaseLogDir
	} else {
		config.baseLogDir = "/data/applogs/cat"
	}
	_, err = os.Stat(config.baseLogDir)
	if err != nil {
		if os.IsNotExist(err) {
			//创建失败
			if err = os.Mkdir(config.baseLogDir, os.ModePerm); err != nil {
				//置为空
				var dir string
				dir, err = filepath.Abs(filepath.Dir(os.Args[0]))
				if err == nil {
					config.baseLogDir = strings.Replace(dir, "\\", "/", -1)
				}

			}
		}
	}

	logger.changeLogFile()

	for _, x := range c.Servers.Servers {
		config.serverAddress = append(config.serverAddress, serverAddress{
			Host:     x.Host,
			Port:     x.Port,
			HttpPort: x.HttpPort,
		})
	}

	json, _ := json.Marshal(config.serverAddress)

	logger.Info("Server addresses: %s", string(json))

	return err
}

func (config *Config) InitWithConfig(domain string, cfg XMLConfig) (err error) {

	config.domain = domain

	defer func() {
		if err == nil {
			logger.Info("Cat has been initialized successfully with appkey: %s", config.domain)
		} else {
			logger.Error("Failed to initialize cat.")
		}
	}()

	var ip net.IP
	if ip, err = getLocalhostIp(); err != nil {
		config.ip = defaultIp
		config.ipHex = defaultIpHex
		logger.Warning("Error while getting local ip, using default ip: %s", defaultIp)
	} else {
		config.ip = ip2String(ip)
		config.ipHex = ip2HexString(ip)
		logger.Info("Local ip has been configured to %s", config.ip)
	}

	if config.hostname, err = os.Hostname(); err != nil {
		config.hostname = defaultHostname
		logger.Warning("Error while getting hostname, using default hostname: %s", defaultHostname)
	} else {
		logger.Info("Hostname has been configured to %s", config.hostname)
	}

	// Print config content to log file.
	data, err := xml.Marshal(cfg)
	if err != nil {
		return
	}

	logger.Info("\n%s", string(data))

	if err = loadXmlConfig(cfg); err != nil {
		return
	}

	return
}

func (config *Config) Init(domain, location string) (err error) {

	config.domain = domain

	defer func() {
		if err == nil {
			logger.Info("Cat has been initialized successfully with appkey: %s", config.domain)
		} else {
			logger.Error("Failed to initialize cat.")
		}
	}()

	//默认空，取ENV，其次使用/data/appdatas/cat目录作为默认目录
	if location == "" {
		location = os.Getenv("CAT_HOME")
		if location == "" {
			location = "/data/appdatas/cat"
		}
		location += "/client.xml"
	}

	var ip net.IP
	if ip, err = getLocalhostIp(); err != nil {
		config.ip = defaultIp
		config.ipHex = defaultIpHex
		logger.Warning("Error while getting local ip, using default ip: %s", defaultIp)
	} else {
		config.ip = ip2String(ip)
		config.ipHex = ip2HexString(ip)
		logger.Info("Local ip has been configured to %s", config.ip)
	}

	if config.hostname, err = os.Hostname(); err != nil {
		config.hostname = defaultHostname
		logger.Warning("Error while getting hostname, using default hostname: %s", defaultHostname)
	} else {
		logger.Info("Hostname has been configured to %s", config.hostname)
	}

	var data []byte
	if data, err = loadConfig(location); err != nil {
		return
	}

	// Print config content to log file.
	logger.Info("\n%s", data)

	if err = parseXMLConfig(data); err != nil {
		return
	}

	return
}
