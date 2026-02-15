package internal

import (
	_ "errors"
	"github.com/gokul1063/songer/configs"
	"log"
	"os"
)

const LogFormat string = configs.LogFormat
const logFile string = configs.LogLocation + "go-commands.log"

func createError(err error) error {
	return err
}

func WriteLog(funtionName string, receivedError error) error {

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		file, err := os.Create(logFile)

		if err != nil {
			return createError(err)
		}
		file.Close()

	}

	file, err := os.OpenFile(
		logFile,
		os.O_APPEND|os.O_WRONLY,
		0666,
	)

	if err != nil {
		return createError(err)
	}

	defer file.Close()

	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf(LogFormat, funtionName, receivedError)

	return nil

}
