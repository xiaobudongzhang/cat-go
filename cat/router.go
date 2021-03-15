package cat

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type routerConfigXMLProperty struct {
	XMLName xml.Name `xml:"property"`
	Id      string   `xml:"id,attr"`
	Value   string   `xml:"value,attr"`
}

type routerConfigXML struct {
	XMLName    xml.Name                  `xml:"property-config"`
	Properties []routerConfigXMLProperty `xml:"property"`
}

type routerConfigJson struct {
	Kvs map[string]string `json:"kvs"`
}

type catRouterConfig struct {
	scheduleMixin
	sample  float64
	routers []serverAddress
	current *serverAddress
	ticker  *time.Ticker
}

var router = catRouterConfig{
	scheduleMixin: makeScheduleMixedIn(signalRouterExit),
	sample:        1.0,
	routers:       make([]serverAddress, 0),
	ticker:        nil,
}

func (c *catRouterConfig) GetName() string {
	return "Router"
}

func (c *catRouterConfig) updateRouterConfig() {
	var query = url.Values{}
	query.Add("env", config.env)
	query.Add("domain", config.domain)
	query.Add("ip", config.ip)
	query.Add("hostname", config.hostname)
	query.Add("op", "json")

	u := url.URL{
		Scheme:   "http",
		Path:     "/cat/s/router",
		RawQuery: query.Encode(),
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	for _, server := range config.serverAddress {
		u.Host = fmt.Sprintf("%s:%d", server.Host, server.HttpPort)
		logger.Info("Getting router config from %s", u.String())

		resp, err := client.Get(u.String())
		if err != nil {
			logger.Warning("Error occurred while getting router config from url %s", u.String())
			continue
		}

		err = c.parse(resp.Body)
		if err == nil {
			return
		} else {
			continue
		}
	}

	logger.Error("Can't get router config from remote server.")
	return
}

func (c *catRouterConfig) handle(signal int) {
	switch signal {
	case signalResetConnection:
		logger.Warning("Connection has been reset, reconnecting.")
		c.current = nil
		c.updateRouterConfig()
	default:
		c.scheduleMixin.handle(signal)
	}
}

func (c *catRouterConfig) afterStart() {
	c.ticker = time.NewTicker(time.Minute * 3)
	c.updateRouterConfig()
}

func (c *catRouterConfig) beforeStop() {
	c.ticker.Stop()
}

func (c *catRouterConfig) process() {
	select {
	case sig := <-c.signals:
		c.handle(sig)
	case <-c.ticker.C:
		c.updateRouterConfig()
	}
}

func (c *catRouterConfig) updateSample(v string) error {
	sample, err := strconv.ParseFloat(v, 32)
	if err != nil {
		logger.Warning("Sample should be a valid float, %s given", v)
		return err
	} else if math.Abs(sample-c.sample) > 1e-9 {
		c.sample = sample
		logger.Info("Sample rate has been set to %f%%", c.sample*100)
	}
	return nil
}

func (c *catRouterConfig) updateBlock(v string) {
	if v == "false" {
		enable()
	} else {
		disable()
	}
}

func (c *catRouterConfig) parse(reader io.ReadCloser) error {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	t := new(routerConfigJson)
	if err := json.Unmarshal(bytes, &t); err != nil {
		logger.Warning("Error occurred while parsing router config json content.\n%s", string(bytes))
	}

	for k, v := range t.Kvs {
		switch k {
		case propertySample:
			err = c.updateSample(v)
			if err != nil {
				return err
			}
		case propertyRouters:
			err = c.updateRouters(v)
			if err != nil {
				return err
			}
		case propertyBlock:
			c.updateBlock(v)
		}
	}
	return nil
}

func (c *catRouterConfig) updateRouters(router string) error {
	newRouters := resolveServerAddresses(router)

	oldLen, newLen := len(c.routers), len(newRouters)

	if newLen == 0 {
		return errors.New("Routers not found")
	} else if oldLen == 0 {
		logger.Info("Routers has been initialized to: %s", newRouters)
		c.routers = newRouters
	} else if oldLen != newLen {
		logger.Info("Routers has been changed to: %s", newRouters)
		c.routers = newRouters
	} else {
		for i := 0; i < oldLen; i++ {
			if !compareServerAddress(&c.routers[i], &newRouters[i]) {
				logger.Info("Routers has been changed to: %s", newRouters)
				c.routers = newRouters
				break
			}
		}
	}

	for _, server := range newRouters {
		if compareServerAddress(c.current, &server) {
			return nil
		}

		addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
		if conn, err := net.DialTimeout("tcp", addr, time.Second); err != nil {
			logger.Info("Failed to connect to %s, retrying...", addr)
			return errors.New("Failed to connect to " + addr)
		} else {
			c.current = &server
			logger.Info("Connected to %s.", addr)
			sender.chConn <- conn
			return nil
		}
	}

	logger.Info("Cannot established a connection to cat server.")
	return nil
}
