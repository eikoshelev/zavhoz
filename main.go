package main

import (

	//"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
)

type inventory struct {
	IP     string   `json:"ip"`
	Apps   []string `json:"apps"`
	Active bool     `json:"active"`
}

var conn, _ = gocb.Connect("127.0.0.1:8091")
var cluster = conn.Authenticate(gocb.PasswordAuthenticator{Username: "Admin", Password: "aadmin"})
var bucket, _ = conn.OpenBucket("testbucket", "")

func main() {

	server := &dns.Server{Addr: ":51", Net: "udp"}

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

    	var host inventory

		_, err := bucket.Get(name, &host)

		if err != nil {
			fmt.Println(name, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}

		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600}
		answer.A = net.ParseIP(host.IP) // Preference = 10
		m.Answer = append(m.Answer, answer)
	}

	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}
