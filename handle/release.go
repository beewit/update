package handle

import (
	"github.com/labstack/echo"
	"github.com/beewit/beekit/utils"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"net/url"
	"github.com/beewit/update/global"
	"time"
	"github.com/beewit/beekit/utils/convert"

	"strings"
	"encoding/base64"
)

const (
	tagFmt                 = "v%d.%d.%d"
	API_SPREAD_TEMP_URL    = "https://gitee.com/api/v5/repos/beewit/%s/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_URL         = "https://gitee.com/api/v5/repos/beewit/spread/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_APP_URL     = "https://gitee.com/api/v5/repos/beewit/app/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_DB_URL      = "https://gitee.com/api/v5/repos/beewit/spread-db/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_INSTALL_URL = "https://gitee.com/api/v5/repos/beewit/spread-install/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	SPREAD                 = "spread"
	SPREAD_APP             = "spread-app"
	SPREAD_INSTALL         = "spread-install"
	SPREAD_DB              = "spread-db"
)

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

type Release struct {
	Version
	Body    string  `json:"body"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Url string `json:"browser_download_url"`
}

func (gr Release) ToRelease() (rel Release) {
	var major, minor, patch int
	fmt.Sscanf(gr.TagName, tagFmt, &major, &minor, &patch)
	rel.TagName = gr.TagName
	rel.Body = gr.Body
	rel.Version = Version{major, minor, patch}
	for _, ga := range gr.Assets {
		url := GetUrl(ga.Url)
		if url != "" {
			rel.Assets = append(rel.Assets, Asset{url})
		}
	}
	return
}

func GetUrl(href string) (newUrl string) {
	u, err := url.Parse(href)
	if err != nil {
		return
	}
	v, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return
	}
	return v.Get("u")
}

func GetRelease(c echo.Context) error {
	app := c.FormValue("app")
	rel, err := getRelease(app)
	if err != nil {
		return utils.ErrorNull(c, fmt.Sprintf("获取更新版本失败，原因：%s", err.Error()))
	}
	return utils.SuccessNullMsg(c, rel)
}

func GetDownloadUrl(c echo.Context) error {
	if utils.IsWechatBrowser(c.Request().UserAgent()) {
		return c.File("app/page/index.html")
	}
	app := c.FormValue("app")
	rel, err := getRelease(app)
	if err != nil {
		return utils.ErrorNull(c, fmt.Sprintf("获取更新版本失败，原因：%s", err.Error()))
	}
	if len(rel.Assets) <= 0 {
		return utils.ErrorNull(c, "下载失败，应用的下载地址未找到！")
	}
	go func() {
		m := map[string]interface{}{}
		m["id"] = utils.ID()
		m["ct_time"] = utils.CurrentTime()
		m["type"] = app
		m["user_agent"] = c.Request().UserAgent()
		m["ct_ip"] = c.RealIP()
		_, err = global.DB.InsertMap("download_logs", m)
		if err != nil {
			global.Log.Error(err.Error())
		}
	}()
	return utils.Redirect(c, rel.Assets[0].Url)
}

func GetDownloadQrCode(c echo.Context) error {
	app := c.FormValue("app")
	if app == "" {
		return utils.ResultHtml(c, "下载类型参数错误")
	}
	base64Img, err := utils.CreateQrCode(fmt.Sprintf("http://update.9ee3.com/download?app=%s&u=%s", app, convert.ToString(time.Now().UnixNano())))
	if err != nil {
		return utils.ResultHtml(c, "生成二维码失败")
	}
	base64Img = strings.Replace(base64Img, "data:image/png;base64,", "", -1)
	dist, _ := base64.StdEncoding.DecodeString(base64Img)
	return c.Stream(http.StatusOK, "image/jpeg", strings.NewReader(string(dist)))
}

func getRelease(app string) (rel Release, err error) {
	var apiUrl string
	switch app {
	case SPREAD:
		apiUrl = API_SPREAD_URL
		break
	case SPREAD_APP:
		apiUrl = API_SPREAD_APP_URL
		break
	case SPREAD_DB:
		apiUrl = API_SPREAD_DB_URL
		break
	case SPREAD_INSTALL:
		apiUrl = API_SPREAD_INSTALL_URL
		break
	default:
		apiUrl = fmt.Sprintf(API_SPREAD_TEMP_URL, app)
		return
	}
	resp, err := http.Get(apiUrl)
	if err != nil {
		return
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var release Release
	json.Unmarshal(dat, &release)
	rel = release.ToRelease()
	return
}
