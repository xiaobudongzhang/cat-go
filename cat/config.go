package cat

import (
	"encoding/xml"
	"io/ioutil"
	"net"
	"os"
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

	if len(c.BaseLogDir) > 0 {
		config.baseLogDir = c.BaseLogDir
		logger.changeLogFile()
	}

	for _, x := range c.Servers.Servers {
		config.serverAddress = append(config.serverAddress, serverAddress{
			host:     x.Host,
			port:     x.Port,
			httpPort: x.HttpPort,
		})
	}

	logger.Info("Server addresses: %s", config.serverAddress)
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

	// TODO load env.

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
