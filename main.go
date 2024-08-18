package main

import (
	"fmt"
	"log"
	"net"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/beevik/etree"
	discover "github.com/use-go/onvif/ws-discovery"
)

func main() {
	a := app.New()
	w := a.NewWindow("ONVIF 设备搜索")

	// 设置窗口分辨率为1280x720
	w.Resize(fyne.NewSize(1280, 720))

	// 获取网络接口列表
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println("获取网络接口失败:", err)
		return
	}

	// 创建UI组件
	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	ifaceSelect := widget.NewSelect(ifaceNames, func(selected string) {
		// 选择网络接口时的回调函数
	})
	ifaceSelect.SetSelected(ifaces[0].Name)

	deviceList := widget.NewTable(
		func() (int, int) { return 9, 4 }, // 预留8行空白信息
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText("")
		},
	)

	// 设置表头和预留的空白行
	deviceList.SetColumnWidth(0, 50)
	deviceList.SetColumnWidth(1, 150)
	deviceList.SetColumnWidth(2, 150)
	deviceList.SetColumnWidth(3, 100)
	deviceList.Length = func() (int, int) { return 9, 4 }
	deviceList.CreateCell = func() fyne.CanvasObject {
		return widget.NewLabel("template")
	}
	deviceList.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		if id.Row == 0 {
			// 表头
			headers := []string{"ID", "设备名称", "IP", "端口"}
			cell.(*widget.Label).SetText(headers[id.Col])
		} else {
			// 预留的空白行
			switch id.Col {
			case 0:
				cell.(*widget.Label).SetText(fmt.Sprintf("%d", id.Row))
			case 1:
				cell.(*widget.Label).SetText("")
			case 2:
				cell.(*widget.Label).SetText("")
			case 3:
				cell.(*widget.Label).SetText("")
			}
		}
	}

	searchBtn := widget.NewButton("搜索设备", func() {
		ifaceName := ifaceSelect.Selected
		log.Println("开始搜索设备...")
		devices, err := runDiscovery(ifaceName)
		if err != nil {
			log.Println("搜索设备失败:", err)
			return
		}
		log.Println("搜索设备成功，找到", len(devices), "个设备")
		deviceList.Length = func() (int, int) { return len(devices) + 1, 4 }
		deviceList.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
			if id.Row == 0 {
				// 表头
				headers := []string{"ID", "设备名称", "IP", "端口"}
				cell.(*widget.Label).SetText(headers[id.Col])
			} else {
				device := devices[id.Row-1]
				switch id.Col {
				case 0:
					cell.(*widget.Label).SetText(fmt.Sprintf("%d", id.Row))
				case 1:
					cell.(*widget.Label).SetText(device.Name)
				case 2:
					cell.(*widget.Label).SetText(device.IP)
				case 3:
					cell.(*widget.Label).SetText(fmt.Sprintf("%d", device.Port))
				}
			}
		}
		deviceList.Refresh()
	})

	// 布局UI
	w.SetContent(container.NewVBox(
		widget.NewLabel("选择网络接口:"),
		ifaceSelect,
		searchBtn,
		widget.NewLabel("设备列表:"),
		deviceList,
	))

	w.ShowAndRun()
}

// Host host
type Host struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func runDiscovery(interfaceName string) ([]Host, error) {
	start := time.Now()
	log.Println("开始搜索设备...")
	var hosts []*Host
	devices, err := discover.SendProbe(interfaceName, nil, []string{"dn:NetworkVideoTransmitter"}, map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	if err != nil {
		log.Printf("搜索设备失败: %s", err)
		return nil, err
	}
	for _, j := range devices {
		doc := etree.NewDocument()
		if err := doc.ReadFromString(j); err != nil {
			log.Printf("解析设备信息失败: %s", err)
		} else {
			endpoints := doc.Root().FindElements("./Body/ProbeMatches/ProbeMatch/XAddrs")
			scopes := doc.Root().FindElements("./Body/ProbeMatches/ProbeMatch/Scopes")

			host := &Host{}

			for _, xaddr := range endpoints {
				xaddrText := xaddr.Text()
				xaddrParts := strings.Split(strings.Split(xaddrText, " ")[0], "/")
				host.URL = xaddrParts[2]
				host.IP = strings.Split(host.URL, ":")[0]
				port, err := strconv.Atoi(strings.Split(host.URL, ":")[1])
				if err == nil {
					host.Port = port
				}
			}

			for _, scope := range scopes {
				re := regexp.MustCompile(`onvif:\/\/www\.onvif\.org\/name\/[A-Za-z0-9-]+`)
				match := re.FindStringSubmatch(scope.Text())
				host.Name = path.Base(match[0])
			}

			hosts = append(hosts, host)
		}
	}

	var result []Host
	for _, host := range hosts {
		result = append(result, *host)
	}
	log.Printf("搜索设备完成，耗时: %s", time.Since(start))
	return result, nil
}