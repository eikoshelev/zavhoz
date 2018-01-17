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
	"net/http"

	"github.com/go-yaml/yaml"
	"github.com/couchbase/gocb"
	"github.com/miekg/dns"
)

// the note structure in the key/value storage
type inventory struct {
	IP     string            `json:"ip"`
	Tag    []string          `json:"tag"`
	Apps   []string          `json:"apps"`
	Active bool              `json:"active"`
	Params map[string]string `json:"params"`
}

// config file structure
type conf struct {

	Server struct {
		Port    string `yaml:"port"`
		Network string `yaml:"network"`
		Ttl     uint32 `yaml:"ttl"`
	} `yaml:"server"`

	Storage struct {
		Login    string   `yaml:"login"`
		Password string   `yaml:"password"`
		Bucket   string   `yaml:"bucket"`
		Hosts    []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./default.yaml", "set config file in the yaml format")

var config conf

func configure() conf {

	configFile, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		fmt.Println("Failed read configuration file: ", err)
	}

    var c conf
	err = yaml.Unmarshal(configFile, &c)

	if err != nil {
		fmt.Println("Failed unmarshal ", *configFlag, err)
	}
	
	return c
}

var bucket *gocb.Bucket

func main() {

	flag.Parse()
	config = configure()
	fmt.Printf("Configuration:%+v\n", config)
	
	conn, err := gocb.Connect(config.Storage.Hosts[0])
	if err != nil {
    	fmt.Println("Failed connect to ", *conn, err)
	}
	_ = conn.Authenticate(gocb.PasswordAuthenticator{config.Storage.Login, config.Storage.Password})
	bucket, err = conn.OpenBucket(config.Storage.Bucket, "")
	if err != nil {
    	fmt.Println("Failed open bucket: ", *bucket, err)
	}
	
	http.HandleFunc("/manager/", manager)
	
	errr := http.ListenAndServe(":8059", nil)
    if errr != nil {
        log.Fatal("ListenAndServe: ", err)
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

func manager(w http.ResponseWriter, r *http.Request) {

	res := r.Method

	switch res {
	case "GET":

		var result inventory
		req := r.URL.Path[len("/manager/"):]
		bucket.Get(req, &result)
		fmt.Println(req, ":", result)

	//TODO: finish
	case "POST":

		fmt.Println("this is POST method")

	case "DELETE":

		req := r.URL.Path[len("/manager/"):]
		bucket.Remove(req, 0)
	
	//TODO: finish
	case "UPDATE":

		fmt.Println("this is UPDATE method")

	default:

		fmt.Println("Error: ", "\"", res, "\"", " - unknown method. Using GET, POST, DELETE or UPDATE method.")
	}
}
