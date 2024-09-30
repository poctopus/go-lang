package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Stream struct {
	URL           string `json:"url"`
	Key1          string `json:"key1"`
	Key2          string `json:"key2"`
	Key3          string `json:"key3"`
	Key4          string `json:"key4,omitempty"` // 如果 key4 是空的，省略输出
	UserAgent     string `json:"useragent"`
	Authorization string `json:"authorization"`
	Proxy         string `json:"proxy"`
	ShakaPackager bool   `json:"shaka-packager"`
	Resolution    string `json:"resolution"`
}

// base64ToHex: 将 Base64 字符串解码为 HEX 格式
func base64ToHex(input string) (string, error) {
	missingPadding := len(input) % 4
	if missingPadding > 0 {
		input += strings.Repeat("=", 4-missingPadding)
	}
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decoded), nil
}

func parseM3UContent(content string) map[string]Stream {
	streams := make(map[string]Stream)

	// 正则表达式来匹配各个部分
	tvgIDRegex := regexp.MustCompile(`tvg-id\s*=\s*"([^"]+)"`)
	urlRegex := regexp.MustCompile(`(https?://[^\s]+\.mpd)`)
	licenseKeyRegex := regexp.MustCompile(`"k"\s*:\s*"([^"]+)"\s*,\s*"kid"\s*:\s*"([^"]+)"`)
	userAgentRegex := regexp.MustCompile(`#EXTVLCOPT:http-user-agent="([^"]+)"`)

	// 分割M3U文件中的每个部分
	sections := strings.Split(content, "#EXTINF")
	for _, section := range sections {
		if len(section) == 0 {
			continue
		}

		fmt.Println("正在处理部分:\n", section)

		// 提取tvg-id和该区块的唯一URL
		tvgIDMatch := tvgIDRegex.FindStringSubmatch(section)
		urlMatch := urlRegex.FindString(section) // 在每个 section 内查找唯一的 HTTP URL

		if len(tvgIDMatch) > 1 && len(urlMatch) > 0 {
			tvgID := tvgIDMatch[1]
			url := urlMatch

			fmt.Printf("找到 tvg-id: %s, url: %s\n", tvgID, url)

			// 提取多个kid和k并转换为hex
			licenseKeyMatches := licenseKeyRegex.FindAllStringSubmatch(section, -1)
			var key1, key2, key3, key4 string

			for i, match := range licenseKeyMatches {
				if i >= 4 {
					break
				}
				kBase64 := strings.TrimSpace(match[1])  // k 的 Base64 值
				kidBase64 := strings.TrimSpace(match[2]) // kid 的 Base64 值

				fmt.Printf("找到 kid (Base64): %s, k (Base64): %s\n", kidBase64, kBase64)

				// Base64 解码并转换为 HEX
				kidHex, err := base64ToHex(kidBase64)
				if err != nil {
					fmt.Printf("解析 kid 时发生错误: %v, kid 的 Base64 值: %s\n", err, kidBase64)
					continue
				}

				kHex, err := base64ToHex(kBase64)
				if err != nil {
					fmt.Printf("解析 k 时发生错误: %v, k 的 Base64 值: %s\n", err, kBase64)
					continue
				}

				// 组合key值
				key := fmt.Sprintf("%s:%s", kidHex, kHex)

				// 分别赋值给 key1, key2, key3, key4
				switch i {
				case 0:
					key1 = key
				case 1:
					key2 = key
				case 2:
					key3 = key
				case 3:
					key4 = key
				}

				fmt.Printf("转换后的 key%d: %s\n", i+1, key)
			}

			// 提取user-agent，若没有找到则使用默认值
			userAgentMatch := userAgentRegex.FindStringSubmatch(section)
			userAgent := "Mozilla/5.0 (Linux; Android 10; BRAVIA 4K VH2 Build/QTG3.200305.006.S292; wv)"
			if len(userAgentMatch) > 1 {
				userAgent = userAgentMatch[1]
			}

			// 构造Stream结构体，并判断key4是否为空，使用omitempty自动忽略空的key4
			streams[tvgID] = Stream{
				URL:           url,
				Key1:          key1,
				Key2:          key2,
				Key3:          key3,
				Key4:          key4, // 新增key4，但如果为空则会被省略输出
				UserAgent:     userAgent,
				Authorization: "",
				Proxy:         "",
				ShakaPackager: false,
				Resolution:    "1280",
			}
		} else {
			fmt.Println("未找到有效的 tvg-id 或 url。Section内容如下：")
			fmt.Println(section) // 打印整个Section内容以帮助调试
		}
	}

	return streams
}

func main() {
	// 手动输入文件路径
	fmt.Print("请输入 M3U 文件路径（如在当前目录下，直接输入M3U文件名就可以）: ")
	reader := bufio.NewReader(os.Stdin)
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	// 读取M3U文件
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	// 处理换行符，统一为 \n
	contentStr := strings.ReplaceAll(string(content), "\r\n", "\n")

	// 解析并转换M3U文件内容
	streams := parseM3UContent(contentStr)

	// 转换为JSON输出
	jsonData, err := json.MarshalIndent(streams, "", "  ")
	if err != nil {
		panic(err)
	}

	// 将结果输出到文件或控制台
	err = ioutil.WriteFile("output.json", jsonData, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("转换完成，结果保存在 output.json 中")
}
