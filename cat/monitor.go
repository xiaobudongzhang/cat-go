package cat

import (
	"bytes"
	"encoding/xml"
	"github.com/shirou/gopsutil/cpu"
	"github.com/yeabow/cat-go/message"
	"runtime"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/host"
)

type catMonitor struct {
	scheduleMixin
	collectors []Collector
}

func (m *catMonitor) GetName() string {
	return "Monitor"
}

func sleep2NextMinute() *time.Timer {
	var delta = 60 - time.Now().Second()
	return time.NewTimer(time.Duration(delta) * time.Second)
}

func (m *catMonitor) afterStart() {
	LogEvent(typeSystem, nameReboot)
	m.collectAndSend()
}

func (m *catMonitor) process() {
	timer := sleep2NextMinute()
	defer timer.Stop()

	select {
	case sig := <-m.signals:
		m.handle(sig)
	case <-timer.C:
		m.collectAndSend()
	}
}

type OSInfo struct {
	Name                string `xml:"name,attr"`
	Arch                string `xml:"arch,attr"`
	Version             string `xml:"version,attr"`
	AvailableProcessors string `xml:"available-processors,attr"`
	SystemLoadAverage   string `xml:"system-load-average,attr"`
	//ProcessTime         string `xml:"process-time"`
	TotalPhysicalMemory    string `xml:"total-physical-memory,attr"`
	FreePhysicalMemory     string `xml:"free-physical-memory,attr"`
	CommittedVirtualMemory string `xml:"committed-virtual-memory,attr"`
	TotalSwapSpace         string `xml:"total-swap-space,attr"`
	FreeSwapSpace          string `xml:"free-swap-space,attr"`
}

func (m *catMonitor) buildXml() *bytes.Buffer {
	type ExtensionDetail struct {
		Id    string `xml:"id,attr"`
		Value string `xml:"value,attr"`
	}

	type Extension struct {
		Id      string            `xml:"id,attr"`
		Desc    string            `xml:"description"`
		Details []ExtensionDetail `xml:"extensionDetail"`
	}

	type CustomInfo struct {
		Key   string `xml:"key,attr"`
		Value string `xml:"value,attr"`
	}

	type Status struct {
		XMLName     xml.Name     `xml:"status"`
		Timestamp   string       `xml:"timestamp,attr"`
		OS          OSInfo       `xml:"os"`
		Extensions  []Extension  `xml:"extension"`
		CustomInfos []CustomInfo `xml:"customInfo"`
	}

	status := Status{
		Timestamp:   time.Now().Format("2006-01-02 15:04:05.999"),
		Extensions:  make([]Extension, 0, len(m.collectors)),
		CustomInfos: make([]CustomInfo, 0, 3),
		OS:          OSInfo{},
	}

	if platform, _, platformVersion, err := host.PlatformInformation(); err == nil {
		status.OS.Name = platform
		status.OS.Arch = runtime.GOARCH
		status.OS.Version = platformVersion
	}

	if count, err := cpu.Counts(true); err == nil {
		status.OS.AvailableProcessors = strconv.Itoa(count)
	} else {
		status.OS.AvailableProcessors = strconv.Itoa(runtime.GOMAXPROCS(0))
	}

	for _, collector := range m.collectors {
		extension := Extension{
			Id:      collector.GetId(),
			Desc:    collector.GetDesc(),
			Details: make([]ExtensionDetail, 0),
		}

		for k, v := range collector.GetProperties() {
			detail := ExtensionDetail{
				Id:    k,
				Value: v,
			}
			extension.Details = append(extension.Details, detail)
		}
		status.Extensions = append(status.Extensions, extension)

		collector.Fetch(&status.OS)
	}

	// add custom information.
	status.CustomInfos = append(status.CustomInfos, CustomInfo{"gocat-version", GoCatVersion})
	status.CustomInfos = append(status.CustomInfos, CustomInfo{"go-version", runtime.Version()})

	buf := bytes.NewBuffer([]byte{})
	encoder := xml.NewEncoder(buf)
	//encoder.Indent("", "\t")

	if err := encoder.Encode(status); err != nil {
		buf.Reset()
		buf.WriteString(err.Error())
		return buf
	}
	return buf
}

func (m *catMonitor) collectAndSend() {
	var trans = message.NewTransaction(typeSystem, "Status", manager.flush)
	defer trans.Complete()

	//trans.LogEvent("Cat_golang_Client_Version", GoCatVersion)

	// NOTE type & name is useless while sending a heartbeat
	heartbeat := message.NewHeartbeat("Heartbeat", config.ip, nil)
	heartbeat.SetData(m.buildXml().String())
	heartbeat.Complete()

	trans.AddChild(heartbeat)
}

var monitor = catMonitor{
	scheduleMixin: makeScheduleMixedIn(signalMonitorExit),
	collectors: []Collector{
		/*&memStatsCollector{},
		&cpuInfoCollector{
			lastTime:    &cpu.TimesStat{},
			lastCPUTime: 0,
		},*/
		&systemCollector{},
	},
}

func AddMonitorCollector(collector Collector) {
	monitor.collectors = append(monitor.collectors, collector)
}
