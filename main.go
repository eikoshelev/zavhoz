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

type Inventory struct {
	Ip     string            `json:"ip,omitempty"`
	Tag    []string          `json:"tag,omitempty"`
	Apps   []string          `json:"apps,omitempty"`
	Active bool              `json:"active,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

func main() {

	Logger := GetLogger()

	config = configure()

	Logger.Infof("Starting...\n", config)

	flag.Parse()

	conn, err := gocb.Connect(config.Storage.Hosts[0])
	if err != nil {
		Logger.Error("Failed connect to host", config.Storage.Hosts[0], err)
	}
	_ = conn.Authenticate(gocb.PasswordAuthenticator{config.Storage.Login, config.Storage.Password})

	bucket, err = conn.OpenBucket(config.Storage.Bucket, "")
	if err != nil {
		Logger.Error("Failed open bucket: ", config.Storage.Bucket, err)
	}

	http.HandleFunc("/manager/", manager)
	http.HandleFunc("/search/", search)

	errr := http.ListenAndServe(":"+config.Server.Http.Port, nil)
	if errr != nil {
		Logger.Fatal("ListenAndServe: ", err)
	}

	server := &dns.Server{Addr: ":" + config.Server.Dns.Port, Net: config.Server.Dns.Network}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			Logger.Fatalf("Failed to set udp listener %s\n", err)
		}
	}()

	dns.HandleFunc(".", handleRequest)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	Logger.Fatalf("Signal (%v) received, stopping\n", s)
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {

	defer w.Close()

	m := new(dns.Msg)
	fmt.Println("handleRequest:inbound message:")
	fmt.Printf("%+v", r)
	for _, q := range r.Question {
		name := q.Name
		fmt.Println(name)

		var host Inventory

		_, err := bucket.Get(name[:len(name)-1], &host)

		if err != nil {
			fmt.Println(name, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}

		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: config.Server.Dns.Ttl}
		answer.A = net.ParseIP(host.Ip)
		m.Answer = append(m.Answer, answer)
	}
	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}
