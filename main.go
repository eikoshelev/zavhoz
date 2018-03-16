package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
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

	// TODO: dns client

	// для выхода из приложения по "Ctrl+C"
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	Logger.Infof("Signal (%v) received, stopping\n", s)
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {

	Logger, err := initLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m := new(dns.Msg)
	fmt.Println("handleRequest:inbound message:")
	fmt.Printf("%+v", r)

	for _, q := range r.Question {
		name := q.Name

		var host inventory

		_, err := bucket.Get(name[:len(name)-1], &host)

		if err != nil {
			Logger.Fatalf("Failed get: %s", name[:len(name)-1])
			fmt.Println(name, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}
		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: Config.Server.DNS.TTL}
		answer.A = net.ParseIP(host.IP)
		m.Answer = append(m.Answer, answer)
	}
	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}
