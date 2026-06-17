package utils

import (
	"encoding/json"
	"fmt"
	"slices"
)

type ServiceEnvArray struct {
	Env          []string
	BuildArgs    []string
	BuildSecrets []string
}

type ServiceEnvByte struct {
	Env          []byte
	BuildArgs    []byte
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
	var build_args []string
	var build_secrets []string

	if err := json.Unmarshal(e.Env, &env); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal env: %v", err)
	}

	if err := json.Unmarshal(e.BuildArgs, &build_args); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal build args: %v", err)
	}

	if err := json.Unmarshal(e.BuildSecrets, &build_secrets); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal build secrets: %v", err)
	}

	return &ServiceEnvArray{
		Env:          CleanArray(env),
		BuildArgs:    CleanArray(build_args),
		BuildSecrets: CleanArray(build_secrets),
	}, nil
}

// to marshal all the env into byte array
func MarshalServiceEnv(e *ServiceEnvArray) (*ServiceEnvByte, error) {
	envByte, err := json.Marshal(e.Env)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal env: %v", err)
	}

	buildArgsByte, err := json.Marshal(e.BuildArgs)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal build args: %v", err)
	}

	buildSecretsByte, err := json.Marshal(e.BuildSecrets)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal build secrets: %v", err)
	}

	return &ServiceEnvByte{
		Env:          envByte,
		BuildArgs:    buildArgsByte,
		BuildSecrets: buildSecretsByte,
	}, nil
}
