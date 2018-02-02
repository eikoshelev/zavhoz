package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/couchbase/gocb"
	"github.com/couchbase/gocb/cbft"
	"github.com/go-yaml/yaml"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

var log = logrus.New()

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
		Log struct {
			Network_type string `yaml:"network_type"`
			Log_host     string `yaml:"log_host"`
			Log_port     string `yaml:"log_port"`
			File_path    string `yaml:"file_path"`
			File_name    string `yaml:"file_name"`
		} `yaml:"log"`
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

	var answer []Inventory

	body, error := ioutil.ReadAll(r.Body)
	if error != nil {
		fmt.Println(error.Error()) //TODO: обработка ошибки !!!
	}

	foo := make(map[string]interface{})
	err := json.Unmarshal(body, &foo)

	for _, j := range [][]byte{body} { //TODO: убрать хлам

		search := make(map[string]interface{})

		err := json.Unmarshal(j, &search)
		if err != nil {
			log.Println(err)
			return
		}

		// слайс для хранения запроса
		res := []cbft.FtsQuery{}

		for key, val := range search {

			switch valt := val.(type) {

			case string: // IP
				res = append(res, cbft.NewPhraseQuery(valt).Field(key))

			case []interface{}: // Tag and/or Apps
				for _, item := range valt {
					if s, ok := item.(string); ok {
						res = append(res, cbft.NewPhraseQuery(s).Field(key))
					}
				}

			case bool: // Active (!)
				res = append(res, cbft.NewBooleanFieldQuery(valt).Field(key))

			case map[string]interface{}: // Params
				for _, item := range valt {
					if s, ok := item.(string); ok {
						res = append(res, cbft.NewPhraseQuery(s).Field(key))
					}
				}
			}
		}

		// распаковываем слайс
		query := cbft.NewConjunctionQuery(res...)

		req := gocb.NewSearchQuery("search-index", query)

		// отправляем запрос
		rows, err := bucket.ExecuteSearchQuery(req)

		for _, hit := range rows.Hits() {
			var ans Inventory
			_, err := bucket.Get(hit.Id, &ans)
			if err != nil {
				fmt.Println(err.Error())
			}
			answer = append(answer, ans)

		}
	}
	jsonDocument, err := json.Marshal(&answer)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Fprintf(w, "%v\n", string(jsonDocument))
}

//TODO: finish logger func
func logger() {

	log.Formatter = new(logrus.TextFormatter)
	hook, err := logrus_syslog.NewSyslogHook(config.Server.Log.Network_type, config.Server.Log.Log_host+":"+config.Server.Log.Log_port, syslog.LOG_INFO, "")
	if err != nil {
		log.Errorln(err)
		hook, err = logrus_syslog.NewSyslogHook(config.Server.Log.Network_type, config.Server.Log.Log_host+":"+config.Server.Log.Log_port, syslog.LOG_INFO, "")
	} else {
		log.Hooks.Add(hook)
	}

	log.Out = ioutil.Discard
}
