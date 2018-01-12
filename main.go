package main

import (

	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"io/ioutil"
	"os/signal"
	"syscall"

	"github.com/go-yaml/yaml"
	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
)

type inventory struct {
	IP     string `json:"ip"`
	Apps []string `json:"apps"`
	Active bool   `json:"active"`
}

type conf struct {
	
	Server struct {
		Port    string    `yaml:"port"`
		Network string `yaml:"network"`
		Ttl     uint32    `yaml:"ttl"`
	} `yaml:"server"`

	Storage struct {
		Login    string `yaml:"login"`
		Password string `yaml:"password"`
		Bucket   string `yaml:"bucket"`
		Hosts []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./config.yaml", "set config file in the yaml format")
var config conf

func configure() conf {

	configFile, err := ioutil.ReadFile(*configFlag)
	if err != nil {
    	fmt.Println(err)
	}

    var c  conf
	err = yaml.Unmarshal(configFile, &c)

	if err != nil {
		fmt.Println("can't unmarshal", *configFlag, err)
	}

	return c
}

var bucket *gocb.Bucket

func main() {

	flag.Parse()
	config = configure()
	fmt.Printf("%+v",config)
	
	conn, err := gocb.Connect(config.Storage.Hosts[0])
	if err != nil {
    	fmt.Println(err)
	}
	_ = conn.Authenticate(gocb.PasswordAuthenticator{config.Storage.Login, config.Storage.Password})
	bucket, err = conn.OpenBucket(config.Storage.Bucket, "")
	if err != nil {
    	// ...
	}

	server := &dns.Server{Addr: ":" + config.Server.Port, Net: config.Server.Network}

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
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: config.Server.Ttl}
		answer.A = net.ParseIP(host.IP)
		m.Answer = append(m.Answer, answer)
	}
	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}