package geo

import (
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/juzeon/epok-forwarder/util"
	"github.com/oschwald/geoip2-golang"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"
)

func Setup() {
	dbFile := "Country.mmdb"
	executable, err := os.Executable()
	if err == nil {
		slog.Info("Get executable path", "path", executable)
		dbFile = filepath.Join(filepath.Dir(executable), dbFile)
	}
	defer openFile(dbFile)
	if _, err := os.Stat(dbFile); err != nil {
		slog.Info("Start downloading geo file...", "path", dbFile)
	} else {
		return
	}
	client := req.C().SetTimeout(120 * time.Second)
	success := false
	dbOriginalURL := "https://raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb"
	for _, url := range []string{
		"https://mirror.ghproxy.com/raw.githubusercontent.com/Loyalsoldier/geoip/release/Country.mmdb",
		dbOriginalURL,
		"https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country.mmdb",
	} {
		resp, err := client.R().SetDownloadCallback(func(info req.DownloadInfo) {
			if info.Response.Response != nil {
				slog.Info(fmt.Sprintf("Downloaded: %.2f%%\n",
					float64(info.DownloadedSize)/float64(info.Response.ContentLength)*100.0))
			}
		}).SetOutputFile(dbFile).Get(url)
		if err != nil {
			slog.Warn("Get geo file from "+url+" failed", "err", err)
			continue
		}
		if resp.IsErrorState() {
			slog.Warn("Get geo file from "+url+" failed", "code", resp.GetStatusCode())
			continue
		}
		success = true
		break
	}
	if !success {
		slog.Error("Could not download geo file. Please download it manually",
			"path", dbFile, "url", dbOriginalURL)
		_ = os.Remove(dbFile)
		os.Exit(1)
	}
	slog.Info("Downloaded geo file")
}

var reader *geoip2.Reader

func openFile(dbFile string) {
	r, err := geoip2.Open(dbFile)
	if err != nil {
		util.ErrExit(err)
	}
	reader = r
	slog.Info("Opened geo file")
}

func GetCountryCode(ip string) string {
	c, err := reader.Country(net.ParseIP(ip))
	if err != nil {
		slog.Error("Could not get country", "ip", ip)
		return ""
	}
	return c.Country.IsoCode
}
