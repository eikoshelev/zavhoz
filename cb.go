package main

import (
	"fmt"
	"github.com/couchbase/gocb"
)

// creating structur
type Data struct {

	IP	string	`json:"ip"`
	App	[]string`json:"apps"`
	Active	bool	`json:"active"`

}

func main () {
	// connecting to cluster
	cluster, _ := gocb.Connect("127.0.0.1:8091")
	cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: "admin",
		Password: "admin",
	})

	bucket, _ := cluster.OpenBucket("testbucket", "")

	// something work with bucket
	// bucket.Manager(...)

	// adding new data
	bucket.Upsert("test_data",
		Data {
			IP:	"0.0.0.0",
			App:	[]string{"foo", "bar"},
			Active:	true,
		}, 0)

	// getting data
	var inData Data
	bucket.Get("test_data", &inData)
	fmt.Println("Data: ", inData)
}
