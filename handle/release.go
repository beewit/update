package handle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/beewit/beekit/utils"
	"github.com/beewit/beekit/utils/convert"
	"github.com/beewit/update/global"
	"github.com/labstack/echo"

	"encoding/base64"
	"strings"

	"github.com/beewit/beekit/utils/enum"
)

const (
	tagFmt                 = "v%d.%d.%d"
	API_SPREAD_TEMP_URL    = "https://gitee.com/api/v5/repos/beewit/%s/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_URL         = "https://gitee.com/api/v5/repos/beewit/spread/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_APP_URL     = "https://gitee.com/api/v5/repos/beewit/app/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_DB_URL      = "https://gitee.com/api/v5/repos/beewit/spread-db/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_INSTALL_URL = "https://gitee.com/api/v5/repos/beewit/spread-install/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_PC_URL      = "https://gitee.com/api/v5/repos/beewit/spread-pc-app/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	API_SPREAD_EXE_URL     = "https://gitee.com/api/v5/repos/beewit/spread-pc-exe/releases/latest?access_token=kdw2HGxYpTzVrdKpbQbV"
	SPREAD                 = "spread"
	SPREAD_APP             = "spread-app"
	SPREAD_INSTALL         = "spread-install"
	SPREAD_PC              = "spread-pc"
	SPREAD_PC_EXE          = "spread-pc-exe"
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
	i := c.FormValue("i")
	if i != "" && utils.IsValidNumber(i) {
		go func() {
			//查询分享id是否存在用户
			rows, _ := global.DB.Query("SELECT id FROM account WHERE id=? AND status=?", i, enum.NORMAL)
			if len(rows) <= 0 {
				global.Log.Error("未找到此用户【%s】", i)
				return
			}
			//添加下载记录关系
			insertDownloadAccessLog(i, c.RealIP())
		}()
	}
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
		//添加下载记录
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

func insertDownloadAccessLog(accId, ip string) {
	m := queryDownloadAccessLog(accId)
	if m != nil {
		sql := "UPDATE download_access_log SET ip=?,ct_time=? WHERE id=?"
		_, err := global.DB.Update(sql, ip, utils.CurrentTime(), m["id"])
		if err != nil {
			global.Log.Error(err.Error())
		}
	} else {

		sql := "INSERT INTO download_access_log(id,ip,account_id,ct_time)VALUES(?,?,?,?)"
		_, err := global.DB.Insert(sql, utils.ID(), ip, accId, utils.CurrentTime())
		if err != nil {
			global.Log.Error(err.Error())
		}
	}
}

func queryDownloadAccessLog(accId string) map[string]interface{} {
	sql := "SELECT * FROM download_access_log WHERE account_id=? LIMIT 1"
	rows, err := global.DB.Query(sql, accId)
	if err != nil {
		global.Log.Error(err.Error())
		return nil
	}
	if len(rows) != 1 {
		return nil
	}
	return rows[0]
}

func GetDownloadQrCode(c echo.Context) error {
	i := c.FormValue("i")
	app := c.FormValue("app")
	if app == "" {
		return utils.ResultHtml(c, "下载类型参数错误")
	}
	base64Img, err := utils.CreateQrCode(fmt.Sprintf("http://update.9ee3.com/download?app=%s&i=%s&u=%s",
		app, i, convert.ToString(time.Now().UnixNano())))
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
	case SPREAD_PC:
		apiUrl = API_SPREAD_PC_URL
		break
	case SPREAD_PC_EXE:
		apiUrl = API_SPREAD_EXE_URL
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
