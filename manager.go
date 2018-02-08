package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/couchbase/gocb"
)

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

		cas, error := bucket.GetAndLock(doc, 10, &document) //TODO: set time lock
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
