package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var bucket *gocb.Bucket

type inventory struct {
	IP     string            `json:"ip,omitempty"`
	Tag    []string          `json:"tag,omitempty"`
	Apps   []string          `json:"apps,omitempty"`
	Active bool              `json:"active,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

func main() {

	// считываем флаг
	flag.Parse()

	// читаем переданный конфиг
	Config = configure()

	// запускаем логгер
	Logger, err := initLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	Logger.Infof("Started!")

	// подключаемся к couchbase
	conn, err := gocb.Connect(Config.Storage.Hosts[0])
	if err != nil {
		Logger.Errorf("Failed connect to host: %s", err)
	}
	// авторизуемся
	err = conn.Authenticate(gocb.PasswordAuthenticator{Config.Storage.Login, Config.Storage.Password})
	if err != nil {
		Logger.Errorf("Failed authenticate: %s", err)
	}
	// открываем бакет для работы
	bucket, err = conn.OpenBucket(Config.Storage.Bucket, "")
	if err != nil {
		Logger.Errorf("Failed open bucket: %s", err)
	}

	// запускаем локальный DNS сервер
	server := &dns.Server{Addr: ":" + Config.Server.DNS.Port, Net: Config.Server.DNS.Network}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			Logger.Fatalf("Failed to set udp listener %s", err)
			os.Exit(1)
		}
	}()
	// вызываем функцию-обработчик для наших dns запросов
	dns.HandleFunc(".", handleRequest)

	// метрики Prometheus
	http.Handle("/metrics", promhttp.Handler())
	//TODO: вынести порт в конфиг
	go func() {
		if err := http.ListenAndServe(":"+Config.Metrics.Port, nil); err != nil {
			Logger.Fatalf("Failed to set http listener: %s", err)
			os.Exit(1)
		}
	}()

	// отдельные инструменты для работы с документами бакета
	http.HandleFunc("/manager/", manager)
	http.HandleFunc("/search/", search)
	// запускаем HTTP сервер
	go func() {
		if err := http.ListenAndServe(":"+Config.Server.HTTP.Port, nil); err != nil {
			Logger.Fatalf("Failed to set http listener: %s", err)
			os.Exit(1)
		}
	}()

	// для выхода из приложения по "Ctrl+C"
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	Logger.Infof("Signal (%v) received, stopping\n", s)
}
