package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"unsafe"
)

type Dingtalker interface {
	SendText(content string, atmobiles []string, atuserid []string, isatall bool)
	SendMarkdown(title string, text string, atmobiles []string, atuserids []string, isatall bool)
	SendLink(title string, text string, messageurl string, picurl string)
	SendFeedcard(title string, text string, titlechild []string, actionurl []string, btnorientation string)
	SendActioncard(title string, text string, titlechild []string, actionurl []string, btnorientation string)
	SendWholeActioncard(title string, text string, singtitle string, singleurl string, btnorientation string)
}

type Robot struct {
	AppKey    string `json:"token"`
	AppSecret string `json:"secret"`
}

func (receiver Robot) Signature() string {
	/*
		加签
	*/

	webhookurl := "https://oapi.dingtalk.com/robot/send?access_token=" + string(receiver.AppKey)
	// 获取当前秒级时间戳
	timestamp := time.Now()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp.UnixNano()/1e6, receiver.AppSecret)
	mac := hmac.New(sha256.New, []byte(receiver.AppSecret))
	mac.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	hookurl := fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookurl, timestamp.UnixNano()/1e6, sign)

	return hookurl
}

type MessageContent struct {
	Urls    []string
	Targets []string
}

type BountyContent struct {
	Hackerone MessageContent
	Bugcrowd  MessageContent
	Intigriti MessageContent
}

func (receiver Robot) SendText(content string, atmobiles []string, atuserid []string, isatall bool) []byte {
	/*
		发送文本信息
		content: 文本内容
		atmobiles: 需要@的手机号列表
		atuserid: 需要@的用户id列表
		isatall: 是否需要@全体成员
	*/

	type params struct {
		At struct {
			AtMobiles []string `json:"atMobiles"`
			AtUserIds []string `json:"atUserIds"`
			IsAtAll   bool     `json:"isAtAll"`
		} `json:"at"`
		Text struct {
			Content string `json:"content"`
		} `json:"text"`
		Msgtype string `json:"msgtype"`
	}
	var p params
	p.At.AtUserIds = atuserid
	p.At.AtMobiles = atmobiles
	p.At.IsAtAll = isatall
	p.Text.Content = content
	p.Msgtype = "text"
	resA := &p
	result, _ := json.Marshal(resA)
	return result
}

func (receiver Robot) SendLink(title string, text string, messageurl string, picurl string) []byte {
	/*
		发送链接信息
		title: 标题
		text: 文本内容
		messageurl: 链接URL
		picurl: 图片地址
	*/

	type params struct {
		Msgtype string `json:"msgtype"`
		Link    struct {
			Text       string `json:"text"`
			Title      string `json:"title"`
			Picurl     string `json:"picurl"`
			Messageurl string `json:"messageurl"`
		} `json:"link"`
	}
	var p params
	p.Msgtype = "link"
	p.Link.Messageurl = messageurl
	p.Link.Picurl = picurl
	p.Link.Title = title
	p.Link.Text = text
	resA := &p
	result, _ := json.Marshal(resA)
	return result
}

func (receiver Robot) SendMarkdown(title string, text string, atmobiles []string, atuserids []string, isatall bool) []byte {
	/*
		发送Markdown文本
		title: 标题
		text: 文本内容
		atmobiles: 需要@的用户手机号码列表
		atuserid: 需要@的用户id列表
		isatall: 是否需要@全体成员
	*/
	type params struct {
		Msgtype  string `json:"msgtype"`
		Markdown struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"markdown"`
		At struct {
			AtMobiles []string `json:"atMobiles"`
			AtUserIds []string `json:"atUserIds"`
			IsAtAll   bool     `json:"isAtAll"`
		} `json:"at"`
	}
	var p params
	p.Msgtype = "markdown"
	p.Markdown.Title = title
	p.Markdown.Text = text
	p.At.IsAtAll = isatall
	p.At.AtUserIds = atuserids
	p.At.AtMobiles = atmobiles
	resA := &p
	result, _ := json.Marshal(resA)
	return result

}

func (receiver Robot) SendFeedcard(title []string, messageurl []string, picurl []string) []byte {
	/*
		发送feedcard类型消息
		title:单条信息文本 ["时代的火车向前开1",时代的火车向前开2]
		messageURL:点击单条信息到跳转链接 ["https://www.dingtalk.com/","https://www.dingtalk.com/"]
		picURL:单条信息后面图片的URL ["https://img.alicdn.com/tfs/TB1NwmBEL9TBuNjy1zbXXXpepXa-2400-1218.png","https://img.alicdn.com/tfs/TB1NwmBEL9TBuNjy1zbXXXpepXa-2400-1218.png"]

	*/
	type params struct {
		Msgtype  string `json:"msgtype"`
		FeedCard struct {
			Links []struct {
				Title      string `json:"title"`
				MessageURL string `json:"messageURL"`
				PicURL     string `json:"picURL"`
			} `json:"links"`
		} `json:"feedCard"`
	}
	var p params
	p.Msgtype = "feedCard"

	if len(title) == len(messageurl) && len(messageurl) == len(picurl) {
		length := len(title)
		for i := 0; i < length; i++ {
			type jsonparams struct {
				Title      string `json:"title"`
				MessageURL string `json:"messageURL"`
				PicURL     string `json:"picURL"`
			}
			var jp jsonparams
			jp.Title = title[i]
			jp.MessageURL = messageurl[i]
			jp.PicURL = messageurl[i]
			p.FeedCard.Links = append(p.FeedCard.Links, jp)
		}
	} else {
		panic("标题与文章链接不一致")
	}
	resA := &p
	result, _ := json.Marshal(resA)
	return result
}

func (receiver Robot) SendActioncard(title string, text string, titlechild []string, actionurl []string, btnorientation string) []byte {
	/*
		发送整体卡片
		title: 首屏会话透出的展示内容。
		text:markdown格式的消息。
		btns:按钮
		title_child:按钮标题
		actionURL:点击按钮触发的URL
		btnOrientation:0：按钮竖直排列  1：按钮横向排列 默认为0
	*/
	if btnorientation == "" {
		btnorientation = "0"
	}
	type params struct {
		Msgtype    string `json:"msgtype"`
		ActionCard struct {
			Title          string `json:"title"`
			Text           string `json:"text"`
			BtnOrientation string `json:"btnOrientation"`
			Btns           []struct {
				Title     string `json:"title"`
				ActionURL string `json:"actionURL"`
			} `json:"btns"`
		} `json:"actionCard"`
	}
	var p params
	p.Msgtype = "actionCard"
	p.ActionCard.Title = text
	p.ActionCard.Title = title
	p.ActionCard.BtnOrientation = btnorientation
	if len(titlechild) == len(actionurl) {
		length := len(titlechild)
		for i := 0; i < length; i++ {
			type jsonparams struct {
				Title     string `json:"title"`
				ActionURL string `json:"actionURL"`
			}
			var jp jsonparams
			jp.Title = titlechild[i]
			jp.ActionURL = actionurl[i]
			p.ActionCard.Btns = append(p.ActionCard.Btns, jp)
		}
	} else {
		panic("标题与文章链接数量不一致")
	}
	resA := &p
	result, _ := json.Marshal(resA)
	return result
}

func (receiver Robot) SendWholeActioncard(title string, text string, singtitle string, singleurl string, btnorientation string) []byte {
	/*
		发送独立卡片
		title:首屏会话透出的展示内容。
		text:markdown格式的消息。
		singleTitle:单个按钮的标题。注意 设置此项和singleURL后，btns无效。
		singleURL:点击singleTitle按钮触发的URL。
		btnOrientation:0：按钮竖直排列  1：按钮横向排列 默认为0
	*/
	if btnorientation == "" {
		btnorientation = "0"
	}
	type params struct {
		ActionCard struct {
			Title          string `json:"title"`
			Text           string `json:"text"`
			BtnOrientation string `json:"btnOrientation"`
			SingleTitle    string `json:"singleTitle"`
			SingleURL      string `json:"singleURL"`
		} `json:"actionCard"`
		Msgtype string `json:"msgtype"`
	}
	var p params
	p.Msgtype = "actionCard"
	p.ActionCard.SingleTitle = singtitle
	p.ActionCard.Title = title
	p.ActionCard.Text = text
	p.ActionCard.SingleURL = singleurl
	p.ActionCard.BtnOrientation = btnorientation
	resA := &p
	result, _ := json.Marshal(resA)
	return result
}

func SendRequest(webhook string, params []byte) {

	fmt.Println(webhook)

	reader := bytes.NewReader(params)
	request, err := http.NewRequest("POST", webhook, reader)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// 设置请求头及代理
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	// 发送请求
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//byte数组直接转成string，优化内存
	str := (*string)(unsafe.Pointer(&respBytes))
	fmt.Println(*str)
}

func TargetMarkdown(target_type string, content MessageContent) (msg_content string) {

	if len(content.Urls) == 0 {
		return
	}

	var url_content_list []string

	var target_content = strings.Join(content.Targets, "\n\n")

	for _, url := range content.Urls {

		url_content_list = append(url_content_list, fmt.Sprintf("[%s](%s)", url, url))

	}

	var url_content = strings.Join(url_content_list, "\n\n")

	msg_content = fmt.Sprintf("#### %s:\n%s\n\n%s\n", target_type, target_content, url_content)

	return

}
