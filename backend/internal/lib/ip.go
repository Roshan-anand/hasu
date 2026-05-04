package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
