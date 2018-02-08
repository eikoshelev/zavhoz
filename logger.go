package main

import (
	"io/ioutil"
	"log/syslog"

	"github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

//TODO: finish logger func
func logger() {

	log.Formatter = new(logrus.TextFormatter)
	hook, err := logrus_syslog.NewSyslogHook(config.Server.Log.Network_type, config.Server.Log.Log_host+":"+config.Server.Log.Log_port, syslog.LOG_INFO, "")
	if err != nil {
		log.Errorln(err)
		hook, err = logrus_syslog.NewSyslogHook(config.Server.Log.Network_type, config.Server.Log.Log_host+":"+config.Server.Log.Log_port, syslog.LOG_INFO, "")
	} else {
		log.Hooks.Add(hook)
	}

	log.Out = ioutil.Discard
}
