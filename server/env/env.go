package env

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/st-user/ojm-drone-local/appos"
)

type Environment struct {
	data map[string]string
}

var once sync.Once
var singleton *Environment
var loadChan = make(chan struct{})

func Get(key string) string {
	loadEnv()
	return singleton.data[key]
}

func GetInt(key string) int {
	ret, err := strconv.Atoi(Get(key))
	if err != nil {
		return 0
	}
	return ret
}

func GetBool(key string) bool {
	ret, err := strconv.ParseBool(Get(key))
	if err != nil {
		return false
	}
	return ret
}

func GetDuration(key string) time.Duration {
	ret, err := time.ParseDuration(Get(key))
	if err != nil {
		return 0
	}
	return ret
}

func loadEnv() {

	once.Do(func() {
		path := filepath.Join(appos.BaseDir(), ".env")
		_path := os.Getenv("GO_ENV_FILE_PATH")

		if len(_path) > 0 {
			path = _path
		}

		singleton = loadEnvFrom(path)

		close(loadChan)
	})
	<-loadChan
}

func loadEnvFrom(path string) *Environment {
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

	return &Environment{
		data: ret,
	}
}
