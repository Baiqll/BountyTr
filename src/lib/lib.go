package lib

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/edsrzf/mmap-go"
	"github.com/projectdiscovery/dnsx/libs/dnsx"
	"golang.org/x/sys/unix"
)

var Blacklist = []string{
	".gov",
	".edu",
	".json",
	".[0-9.]+$",
}

func In(target string, str_array []string) bool {
	// 判断字符串是否 存在于字符串数组内
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func DomainMatch(url string) []string {
	// 提取域名

	// 黑名单正则
	var black_pattern []string
	for _, black := range Blacklist {

		black_pattern = append(black_pattern, fmt.Sprintf(".*%s", black))
	}

	// 特殊过滤
	// black_pattern = append(black_pattern, filterlist...)
	pattern := fmt.Sprintf(`(?!%s)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`, strings.Join(black_pattern, "|"))

	domain_rege := regexp2.MustCompile(pattern, 0)
	// domain_rege := regexp.MustCompile(`^(?!.*gov|.*edu)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`)

	// return dedupe_from_list(domain_rege.FindAllString(url, -1))
	return DedupeFromList(Regexp2FindAllString(domain_rege, url))
}

func ReadFileToMap(filename string) map[string]bool {
	// 读取文件到 map

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	size := info.Size()

	if size == 0 {
		if _, err := file.WriteString("\n"); err != nil {
			log.Fatal(err)
		}
	}

	hash := make(map[string]bool)
	reader := bufio.NewReader(file)
	mm, err := mmap.Map(file, unix.PROT_READ, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer mm.Unmap()

	for i := 0; int64(i) < size; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		hash[strings.Replace(line, "\n", "", -1)] = true
	}
	return hash
}

func SaveTargetsToFile(filename string, targets []string) {

	// 保存目标到文件内

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, target := range targets {
		writer.WriteString(target + "\n")
	}
	writer.Flush()
	file.Sync()

}
func HomeDir() string {
	// 获取 $home 路径
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get user home directory:", err)
	}
	return usr.HomeDir
}

func DedupeFromList(source []string) []string {
	// 列表去重
	var new_list []string

	dedupe_set := make(map[string]bool)
	for _, v := range source {
		dedupe_set[v] = true
	}

	for k := range dedupe_set {

		new_list = append(new_list, k)
	}

	return new_list
}

func Regexp2FindAllString(re *regexp2.Regexp, s string) []string {
	// 正则匹配提取
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func DomainValid(domain string) bool {
	// 检查domain是否符合url格式
	_, err := url.Parse("http://" + domain)
	if err != nil {
		return false
	}

	// DNS 查询
	dnsClient, _ := dnsx.New(dnsx.DefaultOptions)

	// DNS 查询 A 记录
	result, _ := dnsClient.Lookup(domain)

	if len(result) == 0 {
		return false
	}

	return true
}
