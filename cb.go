package main

import (
	"fmt"
	"github.com/couchbase/gocb"
)
// create structur
type Data struct {

	IP  string  `json:"ip"`
	App  []string  `json:"apps"`
	Active  bool  `json:"active"`

}

func main() {
	// connect to cluster
	cluster, _ := gocb.Connect("127.0.0.1:8091")
	cluster.Authenticate(gocb.PasswordAuthenticator {
		Username: "admin",
		Password: "admin",
	})

	bucket, _ := cluster.OpenBucket("testbucket", "")

	// working with bucket
	// bucket.Manager(...)

	// added new data
	bucket.Upsert("test_data",
		Data {

			IP: "0.0.0.0",
			App: []string{"foo", "bar"},
			Active: true,

			}, 0)

	// get data
	var inData Data
	bucket.Get("test_data", &inData)
	fmt.Println("Data: ", inData)
}
