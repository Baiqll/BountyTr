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
	Proxies []string
}

func NewPool(proxies []string) *Pool {

	return &Pool{
		Proxies: proxies,
	}
}
func (p Pool) RandProxy() string {

	index := rand.Intn(len(p.Proxies))
	return p.Proxies[index]
}

type ProxyPool struct {
	Source string
	Pool   Pool
}

func NewProxyPool() *ProxyPool {

	return &ProxyPool{
		Source: "https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-$",
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

				proxies = append(proxies, fmt.Sprintf("http://%s:%s", td_content[1], td_content[2]))
			}

		})
	})

	return

}

func (h ProxyPool) InitPool() (pool Pool) {

	var proxies []string

	for i := 1; i <= 6; i++ {

		proxies = append(proxies, h.GetProxyList(i)...)

	}

	h.Pool = *NewPool(proxies)

	pool = h.Pool
	return
}
