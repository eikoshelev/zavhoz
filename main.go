package main

import (
	"flag"
	"fmt"
	"log"
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

	flag.Parse()
	config = configure()
	fmt.Printf("Configuration:%+v\n", config)

	conn, err := gocb.Connect(config.Storage.Hosts[0])
	if err != nil {
		fmt.Println("Failed connect to host", err)
	}
	_ = conn.Authenticate(gocb.PasswordAuthenticator{config.Storage.Login, config.Storage.Password})

	bucket, err = conn.OpenBucket(config.Storage.Bucket, "")
	if err != nil {
		fmt.Println("Failed open bucket: ", *bucket, err)
	}

	http.HandleFunc("/manager/", manager)
	http.HandleFunc("/search/", search)

	errr := http.ListenAndServe(":"+config.Server.Http.Http_port, nil)
	if errr != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	server := &dns.Server{Addr: ":" + config.Server.Dns.Dns_port, Net: config.Server.Dns.Network}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	dns.HandleFunc(".", handleRequest)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {

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
