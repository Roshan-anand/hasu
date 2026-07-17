package utils

import (
	"encoding/json"
	"fmt"
	"slices"
)

type ServiceEnvArray struct {
	Env          []string
	BuildSecrets []string
}

type ServiceEnvByte struct {
	Env          []byte
	BuildSecrets []byte
}

// remove all the empty string from the array and return pointer to new array
func CleanArray(arr []string) []string {
	return slices.DeleteFunc(arr, func(s string) bool {
		return s == ""
	})
}

// to unmarshal all the evn into array of string
func UnmarshalServiceEnv(e *ServiceEnvByte) (*ServiceEnvArray, error) {
	var env []string
	var buildSecrets []string

	if e.Env != nil {
		if err := json.Unmarshal(e.Env, &env); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal env: %v", err)
		}
	}

	if e.BuildSecrets != nil {
		if err := json.Unmarshal(e.BuildSecrets, &buildSecrets); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal build secrets: %v", err)
		}
	}

	return &ServiceEnvArray{
		Env:          CleanArray(env),
		BuildSecrets: CleanArray(buildSecrets),
	}, nil
}

// to marshal all the env into byte array
func MarshalServiceEnv(e *ServiceEnvArray) (*ServiceEnvByte, error) {
	envByte, err := json.Marshal(e.Env)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal env: %v", err)
	}

	buildSecretsByte, err := json.Marshal(e.BuildSecrets)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal build secrets: %v", err)
	}

	return &ServiceEnvByte{
		Env:          envByte,
		BuildSecrets: buildSecretsByte,
	}, nil
}
