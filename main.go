package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
)

var bucket *gocb.Bucket

type Inventory struct {
	IP     string            `json:"ip"`
	Tag    []string          `json:"tag"`
	Apps   []string          `json:"apps"`
	Active bool              `json:"active"`
	Params map[string]string `json:"params"`
}

type Conf struct {
	Server struct {
		Http struct {
			Http_port string `yaml:"http_port"`
		} `yaml:"http"`
		Dns struct {
			Dns_port string `yaml:"dns_port"`
			Network  string `yaml:"network"`
			Ttl      uint32 `yaml:"ttl"`
		} `yaml:"dns"`
	} `yaml:"server"`

	Storage struct {
		Login    string   `yaml:"login"`
		Password string   `yaml:"password"`
		Bucket   string   `yaml:"bucket"`
		Hosts    []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./default.yaml", "set config file in the yaml format")

var config Conf

func configure() Conf {

	configFile, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		fmt.Println("Failed read configuration file: ", err)
	}

	var c Conf
	err = yaml.Unmarshal(configFile, &c)

	if err != nil {
		fmt.Println("Failed unmarshal ", *configFlag, err)
	}

	return c
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
		answer.A = net.ParseIP(host.IP)
		m.Answer = append(m.Answer, answer)
	}
	m.SetReply(r)
	fmt.Printf("%+v\n", m)
	w.WriteMsg(m)
}

func manager(w http.ResponseWriter, r *http.Request) {

	met := r.Method

	switch met {
	case "GET":
		/*
			doc := r.URL.Path[len("/manager/"):]
			var res json.RawMessage
			bucket.Get(doc, &res)
			val, err := json.Marshal(res)
			if err != nil {
				fmt.Fprintf(w, "can't marshal: ", val, err)
			}
			fmt.Fprintf(w, "%v\n", string(val))
		*/

		var document Inventory

		doc := r.URL.Path[len("/manager/"):]
		_, error := bucket.Get(doc, &document)
		if error != nil {
			fmt.Println(error.Error())
			return
		}
		jsonDocument, error := json.Marshal(&document)
		if error != nil {
			fmt.Println(error.Error())
		}
		fmt.Fprintf(w, "%v\n", string(jsonDocument))

	case "POST":

		var result Inventory

		doc := r.URL.Path[len("/manager/"):]
		body, error := ioutil.ReadAll(r.Body)
		if error != nil {
			fmt.Println(error.Error())
		}
		error = json.Unmarshal(body, &result)
		if error != nil {
			fmt.Println(w, "can't unmarshal: ", doc, error)
		} else {
			bucket.Upsert(doc, result, 0)
		}

	case "DELETE":

		doc := r.URL.Path[len("/manager/"):]
		bucket.Remove(doc, 0)

	//TODO: finish
	case "UPDATE":

		doc := r.URL.Path[len("/manager/"):]
		fragment, error := bucket.LookupIn(doc).Get("inventory").Execute()
		if error != nil {
			//fmt.Println(error.Error())
			//return
		}
		var inventory Inventory

		fragment.Content("inventory", &inventory)
		jsonInventory, error := json.Marshal(&inventory)
		if error != nil {
			//fmt.Println(error.Error())
			//return
		}
		fmt.Println(string(jsonInventory))

		_, error = bucket.MutateIn(doc, 0, 0).Upsert("inventory.ip", "1.2.3.4", true).Execute()
		if error != nil {
			//fmt.Println(error.Error())
			//return
		}

	default:

		fmt.Println("Error: ", "\"", met, "\"", " - unknown method. Using GET, POST, DELETE or UPDATE method.")
	}
}
