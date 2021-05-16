package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func loadEnvFrom(path string) map[string]string {
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

	return ret
}

func loadEnv() map[string]string {
	path := ".env"
	_path := os.Getenv("GO_ENV_FILE_PATH")

	if len(_path) > 0 {
		path = _path
	}

	return loadEnvFrom(path)
}
