package main

import (
	"fmt"
	"regexp"
)

func main() {
	// 编译正则表达式
	re := regexp.MustCompile(`(https?:\/\/)?[-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9\/?=&\*]{0,80})+`)

	// 测试字符串
	testStrings := []string{
		"https://example-string1.example.com",
		"example-string1.example.com",
		"*.example-string1.example.com",
		"example-string1.example.com/api/sss",
		"example-string1.example.com/api/sss?query=123",
	}

	// 检查每个字符串是否匹配
	for _, testString := range testStrings {
		if re.FindString(testString) != "" {
			fmt.Println("Found in:", testString)
		} else {
			fmt.Println("Did not find in:", testString)
		}
	}
}
