package main

import (
	"io/ioutil"
	"log/syslog"
	"os"

	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

//глобальный логер
var logger *log.Logger

func GetLogger() *log.Logger {
	logger = initLogger()
	return logger
}

//логер с параметрами из конфига
func initLogger() *log.Logger {
	logger := log.New()
	switch config.Log.Type {
	case "syslog":
		logger = initSyslogger()
	case "stderr":
		logger.Out = os.Stderr
		return logger
	case "stdout":
		logger.Out = os.Stdout
		return logger
	default:
		//...
	}
	if config.Log.Debug_mode {
		logger.Out = os.Stdout
	}
	logger.Formatter = &log.TextFormatter{}
	logger.Level = logLevel[config.Log.Severity]
	return logger
}

var logLevel = map[string]log.Level{
	"LOG_EMERG":   log.PanicLevel,
	"LOG_ALERT":   log.PanicLevel,
	"LOG_CRIT":    log.FatalLevel,
	"LOG_ERR":     log.ErrorLevel,
	"LOG_WARNING": log.WarnLevel,
	"LOG_NOTICE":  log.InfoLevel,
	"LOG_INFO":    log.InfoLevel,
	"LOG_DEBUG":   log.DebugLevel,
}

//SysLog
func initSyslogger() *log.Logger {
	var LogSeverity = map[string]syslog.Priority{
		"LOG_EMERG":   syslog.LOG_EMERG,
		"LOG_ALERT":   syslog.LOG_ALERT,
		"LOG_CRIT":    syslog.LOG_CRIT,
		"LOG_ERR":     syslog.LOG_ERR,
		"LOG_WARNING": syslog.LOG_WARNING,
		"LOG_NOTICE":  syslog.LOG_NOTICE,
		"LOG_INFO":    syslog.LOG_INFO,
		"LOG_DEBUG":   syslog.LOG_DEBUG,
	}
	var LogFacility = map[string]syslog.Priority{
		"LOG_KERN":     syslog.LOG_KERN,
		"LOG_USER":     syslog.LOG_USER,
		"LOG_MAIL":     syslog.LOG_MAIL,
		"LOG_DAEMON":   syslog.LOG_DAEMON,
		"LOG_AUTH":     syslog.LOG_AUTH,
		"LOG_SYSLOG":   syslog.LOG_SYSLOG,
		"LOG_LPR":      syslog.LOG_LPR,
		"LOG_NEWS":     syslog.LOG_NEWS,
		"LOG_UUCP":     syslog.LOG_UUCP,
		"LOG_CRON":     syslog.LOG_CRON,
		"LOG_AUTHPRIV": syslog.LOG_AUTHPRIV,
		"LOG_FTP":      syslog.LOG_FTP,

		"LOG_LOCAL0": syslog.LOG_LOCAL0,
		"LOG_LOCAL1": syslog.LOG_LOCAL1,
		"LOG_LOCAL2": syslog.LOG_LOCAL2,
		"LOG_LOCAL3": syslog.LOG_LOCAL3,
		"LOG_LOCAL4": syslog.LOG_LOCAL4,
		"LOG_LOCAL5": syslog.LOG_LOCAL5,
		"LOG_LOCAL6": syslog.LOG_LOCAL6,
		"LOG_LOCAL7": syslog.LOG_LOCAL7,
	}
	logger := log.New()
	hook, err := logrus_syslog.NewSyslogHook(
		config.Log.Type,
		config.Log.Host+":"+config.Log.Port,
		LogSeverity[config.Log.Severity]|LogFacility[config.Log.Facility],
		config.Title)
	if err != nil {
		log.Errorln(err)
		hook, _ = logrus_syslog.NewSyslogHook(
			"", "", LogSeverity[config.Log.Severity]|LogFacility[config.Log.Facility],
			config.Title)
	}
	logger.Hooks.Add(hook)

	logger.Out = ioutil.Discard
	return logger
}
