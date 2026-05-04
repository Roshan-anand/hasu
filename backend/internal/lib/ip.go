package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

type IPResponse struct {
	IP string `json:"ip"`
}

func GetPublicUrl() string {
	resp, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var data IPResponse

	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("https://%s:8080", data.IP)
}

func ValidatePublicUrl(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	if _, err := net.LookupHost(u.Host); err != nil {
		return false
	}

	return true
}
