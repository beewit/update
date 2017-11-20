package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/beewit/beekit/utils/convert"
	"github.com/beewit/beekit/utils"
	"fmt"
	"github.com/beewit/beekit/conf"
	"github.com/beewit/update/handle"
)

var (
	CFG  = conf.New("config.json")
	IP   = CFG.Get("server.ip")
	Port = CFG.Get("server.port")
	Host = fmt.Sprintf("http://%v:%v", IP, Port)
)

func main() {
	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Recover())
	e.GET("/api/release", handle.GetRelease)
	utils.Open(Host)
	port := ":" + convert.ToString(Port)
	e.Logger.Fatal(e.Start(port))
}
