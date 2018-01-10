package main

import (

	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/package"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
)

type inventory struct {
	IP     string   `json:"ip"`
	Apps   []string `json:"apps"`
	Active bool     `json:"active"`
}

type Config struct {
    Dnscb []Dnscb
}

type Dnscb struct {
    Host    string
    Dnsport string
    Cluster Cluster
}

type Cluster struct {
    Login      string
    Pass       string
    Bucketname string
}

var configFlag, _ = flag.String("config", "", "a string")
var file, _ = os.Open(flag.Parse(configFlag))
var decoder = NewDecoder(file)
var config = new(Config)

var conn, _ = gocb.Connect(config.Dnscb[0].host)
var cluster = conn.Authenticate(gocb.PasswordAuthenticator{config.Dnscb[0].cluster.login, config.Dnscb[0].cluster.pass})
var bucket, _ = conn.OpenBucket(config.Dnscb[0].cluster.bucketname, "")


func main() {

	server := &dns.Server{config.Dnscb[0].dnsport, Net: "udp"}

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

		_, err := bucket.Get(name[:len(name)-1], &host)

		if err != nil {
			fmt.Println(name, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}

		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600}
		answer.A = net.ParseIP(host.IP)
		m.Answer = append(m.Answer, answer)
	}

	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}
