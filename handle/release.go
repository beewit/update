package handle

import (
	"github.com/labstack/echo"
	"github.com/beewit/beekit/utils"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"github.com/pkg/errors"
	"net/url"
)

const (
	tagFmt             = "v%d.%d.%d"
	API_SPREAD_URL     = "https://gitee.com/api/v5/repos/beewit/spread/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_APP_URL = "https://gitee.com/api/v5/repos/beewit/app/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_DB_URL  = "https://gitee.com/api/v5/repos/beewit/spread-db/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	SPREAD             = "spread"
	SPREAD_APP         = "spread-app"
	SPREAD_DB          = "spread-db"
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

func GetDownloadUrl(c echo.Context)error  {app := c.FormValue("app")
	rel, err := getRelease(app)
	if err != nil {
		return utils.ErrorNull(c, fmt.Sprintf("获取更新版本失败，原因：%s", err.Error()))
	}
	return utils.Redirect(c, rel.Assets[0].Url)
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
	default:
		err = errors.New("app不存在")
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
