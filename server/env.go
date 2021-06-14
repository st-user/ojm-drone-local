package main

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment struct {
	data map[string]string
}

func (env *Environment) Get(key string) string {
	return env.data[key]
}

func (env *Environment) GetInt(key string) int {
	ret, err := strconv.Atoi(env.Get(key))
	if err != nil {
		return 0
	}
	return ret
}

func (env *Environment) GetDuration(key string) time.Duration {
	ret, err := time.ParseDuration(env.Get(key))
	if err != nil {
		return 0
	}
	return ret
}

func loadEnvFrom(path string) Environment {
	body, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(body), "\n")
	ret := make(map[string]string)

	for _, _line := range lines {
		line := strings.ReplaceAll(_line, "\r", "")

		startChar := strings.Index(line, "#")

		if startChar == 0 {
			continue
		}

		lineComponents := strings.Split(line, "=")
		key := lineComponents[0]
		value := strings.Join(lineComponents[1:], "=")
		ret[key] = value
	}

	return Environment{
		data: ret,
	}
}

func loadEnv() Environment {
	path := ".env"
	_path := os.Getenv("GO_ENV_FILE_PATH")

	if len(_path) > 0 {
		path = _path
	}

	return loadEnvFrom(path)
}
