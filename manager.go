package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/couchbase/gocb"
)

func manager(w http.ResponseWriter, r *http.Request) {

	Logger, err := initLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	type Cas gocb.Cas

	switch r.Method {
	case "GET":

		var document Inventory

		doc := r.URL.Path[len("/manager/"):]
		_, err := bucket.Get(doc, &document)
		if err != nil {
			Logger.Errorf("GET: Failed getting: %s", err)
			return
		}
		jsonDocument, err := json.Marshal(&document)
		if err != nil {
			Logger.Errorf("GET: Can`t marshal: %v", err)
		}
		fmt.Fprintf(w, "%v\n", string(jsonDocument))

	case "POST":

		var result Inventory

		doc := r.URL.Path[len("/manager/"):]
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Logger.Fatalf("Incorrect body request: %v", err)
		}
		err = json.Unmarshal(body, &result)
		if err != nil {
			Logger.Errorf("POST: Can't unmarshal: %v", err)
		} else {
			bucket.Upsert(doc, result, 0)
		}

	case "DELETE":

		doc := r.URL.Path[len("/manager/"):]
		_, err := bucket.Remove(doc, 0)
		if err != nil {
			Logger.Errorf("DELETE: Deleting failed: %v", err)
		}

	case "UPDATE":

		doc := r.URL.Path[len("/manager/"):]

		var document Inventory

		cas, err := bucket.GetAndLock(doc, 10, &document) //TODO: set time lock
		if err != nil {
			Logger.Errorf("UPDATE: Failed getting and locking: %v", err) //TODO: обработка ошибки
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Logger.Errorf("UPDATE: Incorrect body request: %v", err) //TODO: обработка ошибки
		}
		err = json.Unmarshal(body, &document)
		if err != nil {
			Logger.Errorf("UPDATE: Can't unmarshal: %v", err) //TODO: обработка ошибки
		}

		cas, err = bucket.Replace(doc, &document, cas, 0)
		if err != nil {
			Logger.Errorf("UPDATE: Failed replace document: %v", err)
		}
		bucket.Unlock(doc, cas)

	default:

		Logger.Fatalf("DEFAULT MANAGER: Incorrect method!")
		fmt.Println("Error: ", "\"", r.Method, "\"", " - unknown method. Using GET, POST, DELETE, UPDATE method.")
	}
}
