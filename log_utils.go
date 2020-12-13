package hlsdl

import (
	"io/ioutil"
	"log"
	"os"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func EnableDebugMessages() {
	logger.SetOutput(os.Stderr)
}

func DisableDebugMessages() {
	logger.SetOutput(ioutil.Discard)
}
