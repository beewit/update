package main

import (
	"github.com/beewit/beekit/utils"
	"github.com/beewit/beekit/utils/convert"
	"github.com/beewit/update/global"
	"github.com/beewit/update/handle"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())
	e.Static("/app", "app")
	e.File("MP_verify_3Z6AKFClzM8nQt3q.txt", "app/page/MP_verify_3Z6AKFClzM8nQt3q.txt")
	e.GET("/api/release", handle.GetRelease)
	e.GET("/download", handle.GetDownloadUrl)
	e.GET("/download/qrcode", handle.GetDownloadQrCode)
	utils.Open(global.Host)
	port := ":" + convert.ToString(global.Port)
	e.Logger.Fatal(e.Start(port))
}
