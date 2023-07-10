package main

import (
	"strings"

	log "github.com/cihub/seelog"
)

func setInitLogging(logLevel string) {

	logLevel = strings.ToLower(logLevel)

	testConfig := `
	<seelog  type="sync" minlevel="`
	testConfig = testConfig + logLevel
	testConfig = testConfig + `">
		<outputs formatid="main">
			<filter levels="error">
				<file path="./esm.log"/>
			</filter>
			<console formatid="main" />
		</outputs>
		<formats>
			<format id="main" format="[%Date(01-02) %Time] [%LEV] [%File:%Line,%FuncShort] %Msg%n"/>
		</formats>
	</seelog>`

	logger, err := log.LoggerFromConfigAsString(testConfig)
	if err != nil {
		log.Error("init config error,", err)
	}
	err = log.ReplaceLogger(logger)
	if err != nil {
		log.Error("init config error,", err)
	}
}
