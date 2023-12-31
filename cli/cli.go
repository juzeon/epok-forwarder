package cli

import (
	"errors"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/joho/godotenv"
	"github.com/juzeon/epok-forwarder/data"
	"github.com/juzeon/epok-forwarder/util"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

var envFile string
var apiURL string
var apiSecret string
var client *req.Client

func InitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		util.ErrExit(err)
	}
	envFile = path.Join(home, ".config/epok-forwarder/.env")
	err = os.MkdirAll(path.Dir(envFile), 0755)
	if err != nil {
		util.ErrExit(err)
	}
	_ = godotenv.Load(envFile)
	apiURL = strings.TrimSuffix(os.Getenv("EPOK_API"), "/")
	apiSecret = os.Getenv("EPOK_SECRET")
	client = req.C().
		SetTimeout(5*time.Second).
		SetCommonHeader("Authorization", "Bearer "+apiSecret).
		SetBaseURL(apiURL)
}
func validateCLiState() {
	if apiURL == "" {
		util.ErrExit(errors.New("EPOK_API is not set"))
	}
}
func Reload() {
	validateCLiState()
	resp, err := client.R().Post("/api/reload")
	if err != nil {
		util.ErrExit(err)
	}
	res := resp.String()
	if resp.IsSuccessState() {
		slog.Info("Success hot reload: " + res)
		os.Exit(0)
	} else {
		slog.Error("Error hot reload: " + res)
		os.Exit(1)
	}
}
func Generate(baseConfig data.BaseConfig) {
	err := os.WriteFile(envFile, []byte(fmt.Sprintf(`EPOK_API=%s
EPOK_SECRET=%s
`, "http://"+baseConfig.API, baseConfig.Secret)), 0644)
	if err != nil {
		util.ErrExit(err)
	}
	slog.Info("Generated env file: " + envFile)
	os.Exit(0)
}
