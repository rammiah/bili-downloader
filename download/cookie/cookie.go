package cookie

import (
	_ "embed"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"unicode"

	"golang.org/x/net/publicsuffix"
)

var (
	jar        *cookiejar.Jar
	biliUrl, _ = url.Parse("https://bilibili.com")
)

func ParseCookies(ckTxt string) []*http.Cookie {
	kvs := strings.Split(ckTxt, ";")
	cks := make([]*http.Cookie, 0, len(kvs))
	for _, kv := range kvs {
		if idx := strings.Index(kv, "="); idx == -1 {
			continue
		} else {
			k, v := kv[:idx], kv[idx+1:]
			k = strings.TrimSpace(k)
			v = strings.TrimFunc(v, func(r rune) bool {
				return r == '"' || unicode.IsSpace(r)
			})
			if v == "" || k == "" {
				continue
			}
			cks = append(cks, &http.Cookie{
				Name:   k,
				Value:  v,
				Secure: true,
				Domain: ".bilibili.com",
			})
		}
	}
	return cks
}

func init() {
	jar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	configPath := os.Getenv("HOME") + "/.config/bili-downloader/"
	os.MkdirAll(configPath, 0755)
	confFile := configPath + "cookie.txt"
	_, err := os.Stat(configPath + "cookie.txt")
	var ckTxt string
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	} else {
		buf, err := os.ReadFile(confFile)
		if err != nil {
			panic(err)
		}
		ckTxt = string(buf)
	}

	jar.SetCookies(biliUrl, ParseCookies(string(ckTxt)))
}

// SaveCookies save cookies in jar to file
func SaveCookies() {
	configPath := os.Getenv("HOME") + "/.config/bili-downloader/"
	os.MkdirAll(configPath, 0755)
	confFile := configPath + "cookie.txt"
	of, err := os.OpenFile(confFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer func() {
		of.Sync()
		of.Close()
	}()
	// jar support concurrent access
	for _, k := range jar.Cookies(biliUrl) {
		of.WriteString(k.String() + ";")
	}
}

func GetCookieJar() *cookiejar.Jar {
	return jar
}
