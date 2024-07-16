package proxypool

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Pool struct {
	Enable bool
	Proxies []string
}

func NewPool(proxies []string, enable bool) *Pool {

	return &Pool{
		Proxies: proxies,
		Enable: enable,
	}
}
func (p Pool) RandProxy() (fixedURL string) {
	/*
		Pool.Enable 看是否使用代理
	*/

	if(p.Enable){
		index := rand.Intn(len(p.Proxies))
		fixedURL = p.Proxies[index]
	}
	return 
}

type ProxyPool struct {
	Enable bool
	Source string
	Pool   Pool
}

func NewProxyPool(enable bool) *ProxyPool {

	return &ProxyPool{
		Enable:  enable,
		Source: "https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-$",
		// Source: "https://list.proxylistplus.com/Socks-List-$",
	}
}

func (h ProxyPool) GetProxyList(index int) (proxies []string) {
	res, err := http.Get(strings.Replace(h.Source, "$", fmt.Sprintf("%d", index), 1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取网页失败: %v\n", err)
		return
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析HTML失败: %v\n", err)
		return
	}

	doc.Find(".bg").Each(func(i int, table *goquery.Selection) {
		table.Find(".cells").Each(func(i int, tr *goquery.Selection) {

			var td_content []string

			tr.Find("td").Each(func(i int, s *goquery.Selection) {
				td_content = append(td_content, s.Text())
			})

			if len(td_content) > 0 {

				proxies = append(proxies, fmt.Sprintf("%s://%s:%s", td_content[3], td_content[1], td_content[2]))
			}

		})
	})

	return

}

func (h ProxyPool) InitPool() (pool Pool) {

	var proxies = []string{
		"http://115.205.239.179:7890",
		"http://77.46.138.38:8080",
		"https://111.224.10.230:8089",
		"http://202.110.67.141:9091",
		"http://217.52.247.89:1981",
		"http://111.59.10.36:9002",
		"http://8.219.5.240:8081",
		"http://114.233.216.149:7788",
		"http://201.182.251.142:999",
		"http://47.109.53.253:45554",
		"http://117.1.107.89:4014",
		"http://167.71.5.83:8080",
		"https://180.121.129.43:8899",
		"http://102.218.160.132:80",
		"http://41.77.188.131:80",
		"http://124.70.55.29:87",
		"http://117.70.48.139:8089",
		"http://119.13.111.169:20201",
		"https://115.74.163.47:4007",
		"https://43.251.117.67:45787",
		"https://117.57.92.235:8089",
		"http://80.74.77.48:80",
		"http://134.35.221.24:8080",
		"http://93.1.195.28:8080",
		"https://43.251.119.55:45787",
		"https://111.224.11.176:8089",
		"http://171.249.92.16:4011",
		"http://221.6.139.190:9002",
		"http://156.192.170.244:8080",
		"http://124.70.205.56:50001",
		"http://8.208.89.32:8081",
		"http://120.82.174.128:9091",
		"http://64.225.8.132:9981",
		"http://183.164.242.55:8089",
		"http://111.224.213.219:8089",
		"http://47.254.47.61:443",
		"http://171.233.132.239:4014",
		"http://47.92.239.69:9992",
		"http://111.225.152.161:8089",
		"http://47.113.219.226:9091",
		"http://60.12.168.114:9002",
	}

	h.Pool = *NewPool(proxies, h.Enable)

	pool = h.Pool

	return
}
