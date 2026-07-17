package docker

import (
	"fmt"
	"strings"

	"github.com/Roshan-anand/godploy/internal/lib/security"
)

type GenerateNameRes struct {
	ServiceName string
	ImgName     string
}

// helper function to generate service and image name
func GenerateServiceAndImgName(name string, branch string) *GenerateNameRes {
	branch = strings.ReplaceAll(branch, "/", "-")

	base := fmt.Sprintf("%s-%s", name, branch)
	id := security.GenerateRandomID(3, true)
	serviceName := fmt.Sprintf("%s-%s", base, id)
	imgName := strings.ToLower(fmt.Sprintf("%s-dyp_%s", base, id))

	return &GenerateNameRes{
		ServiceName: serviceName,
		ImgName:     imgName,
	}
}
