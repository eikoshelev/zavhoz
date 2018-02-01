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
	"github.com/couchbase/gocb/cbft"
	"github.com/go-yaml/yaml"
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

func manager(w http.ResponseWriter, r *http.Request) {

	type Cas gocb.Cas

	met := r.Method

	switch met {
	case "GET":

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

	case "UPDATE":

		doc := r.URL.Path[len("/manager/"):]

		var document Inventory

		cas, error := bucket.GetAndLock(doc, 000, &document) //TODO: set time lock
		if error != nil {
			fmt.Println(error.Error()) //TODO: обработка ошибки
		}
		body, error := ioutil.ReadAll(r.Body)
		if error != nil {
			fmt.Println(error.Error()) //TODO: обработка ошибки
		}
		error = json.Unmarshal(body, &document)
		if error != nil {
			fmt.Println(w, "can't unmarshal: ", error.Error()) //TODO: обработка ошибки
		}

		cas, error = bucket.Replace(doc, &document, cas, 0)
		if error != nil {
			fmt.Println("Failed Replace document")
		}
		bucket.Unlock(doc, cas)

	default:

		fmt.Println("Error: ", "\"", met, "\"", " - unknown method. Using GET, POST, DELETE, UPDATE method.")
	}
}

func search(w http.ResponseWriter, r *http.Request) {

	var search Inventory

	type FtsHit struct {
		ID string `json:"id,omitempty"`
	}

	//doc := r.URL.Path[len("/search/"):]

	body, error := ioutil.ReadAll(r.Body)
	if error != nil {
		fmt.Println(error.Error()) //TODO: обработка ошибки
	}

	error = json.Unmarshal(body, &search)
	if error != nil {
		fmt.Println(w, "can't unmarshal: ", error.Error()) //TODO: обработка ошибки
	}

	//docid := cbft.NewDocIdQuery(doc)

	//TODO: составной запрос (работает только в случае указания значений для всех полей)
	qp := cbft.NewConjunctionQuery(
		cbft.NewPhraseQuery(search.Ip).Field("ip"),
		cbft.NewPhraseQuery(search.Tag[0]).Field("tag"),
		cbft.NewPhraseQuery(search.Apps[0]).Field("apps"),
		cbft.NewBooleanFieldQuery(search.Active).Field("active"),
		//cbft.NewQueryStringQuery(search.Params).Field("params"),
	)

	/*
		if doc != "" && doc != "*" {
			qp.And(cbft.NewDisjunctionQuery(
				//...
			))
		}
	*/

	q := gocb.NewSearchQuery("search-index", qp)

	rows, err := bucket.ExecuteSearchQuery(q)
	if err != nil {
		fmt.Println(err.Error())
	}

	var result Inventory

	for _, hit := range rows.Hits() {
		res, _ := bucket.Get(hit.Id, &result)
		fmt.Printf(hit.Id, ":", res)
	}

	jsonDocument, error := json.Marshal(&result)
	if error != nil {
		fmt.Println(error.Error())
	}
	fmt.Fprintf(w, "%v\n", string(jsonDocument))

	/*
		doc := r.URL.Path[len("/search/"):]

		if doc == "" {
			doc = "*"
		}

		var docsearch Inventory
		var param []interface{}

		body, error := ioutil.ReadAll(r.Body)
		if error != nil {
			fmt.Println(error.Error()) //TODO: обработка ошибки
		}

		error = json.Unmarshal(body, &docsearch)
		if error != nil {
			fmt.Println(w, "can't unmarshal: ", error.Error()) //TODO: обработка ошибки
		}

		query := gocb.NewN1qlQuery("SELECT " + doc + " FROM `testbucket` " + "WHERE ip=$1 AND tag=$2 AND apps=$3 AND active=$4 AND params=$5")

		param = append(param, docsearch.Ip)
		param = append(param, docsearch.Tag)
		param = append(param, docsearch.Apps)
		param = append(param, docsearch.Active)
		param = append(param, docsearch.Params)

		rows, error := bucket.ExecuteN1qlQuery(query, param)
		if error != nil {
			fmt.Println(error.Error()) //TODO: обработка ошибки
		}

		var res interface{}

		for rows.Next(&res) {
			fmt.Printf("Row: %+v\n", res)
		}
		if error = rows.Close(); error != nil {
			fmt.Printf("Couldn't get all the rows: %s\n", error)
		}
	*/
}
