package main

import (
	"fmt"
	"github.com/couchbase/gocb"
)

type Data struct {

	IP	   string	         `json:"ip"`
	Tag    []string          `json:"tag"`
	Apps   []string          `json:"apps"`
	Active bool	             `json:"active"`
	Params map[string]string `json:"params"`

}

func main () {
	// connecting to cluster
	cluster, _ := gocb.Connect("127.0.0.1:8091")
	cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: "Admin",
		Password: "aadmin",
	})

	bucket, err := cluster.OpenBucket("testbucket", "")
	if err != nil{
		fmt.Println("Error: ", err)
	}

	// adding new data
	bucket.Upsert("test_data",
		Data {
			IP: "0.0.0.0",
			Tag: []string{"something tag"},
			Apps: []string{"foo", "bar"},
			Active:	true,
			Params: map[string]string{"key1":"val1", "key2":"val2"},
		}, 0)

	// getting data
	var inData Data
	bucket.Get("test_data", &inData)
	fmt.Println("Data: ", inData)

	//deleting data
	//bucket.Remove("test_data", 0)
}
