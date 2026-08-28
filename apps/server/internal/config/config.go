package config

import (
	"os"

	"github.com/Roshan-anand/hasu/internal/lib/types"
)

type Config struct {
	Port             string
	SessionDataName  string
	SessionTokenName string
	EchoCtxUserKey   string
	JwtSecret        string
	WebUrl           string
	SqliteDir        string
	BadgerDir        string
	AppEnv           types.AppEnv
	ServerUrl        string
	CodeStoreDir     string
}

func LoadConfig() (*Config, error) {
	appEnv := os.Getenv("SERVER_ENV")
	jwtSecrect := os.Getenv("JWT_SECRET")
	webUrl := os.Getenv("WEB_URL")
	srvUrl := os.Getenv("SERVER_PUBLIC_URL")

	// TODO : load from env variable
	return &Config{
		Port:             "8080",
		SessionDataName:  "hasu_session_data",
		SessionTokenName: "hasu_session_token",
		EchoCtxUserKey:   "user_email",
		JwtSecret:        jwtSecrect,
		WebUrl:           webUrl,
		SqliteDir:        "hasu-data/sqlite",
		BadgerDir:        "hasu-data/badger",
		AppEnv:           types.AppEnv(appEnv),
		ServerUrl:        srvUrl,
		CodeStoreDir:     "/etc/hasu/code",
	}, nil
}
