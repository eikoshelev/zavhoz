package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/couchbase/gocb"
)

func manager(w http.ResponseWriter, r *http.Request) {

	Logger, _ := initLogger()

	type Cas gocb.Cas

	switch r.Method {
	case "GET":

		var document inventory

		doc := r.URL.Path[len("/manager/"):]
		_, err := bucket.Get(doc, &document)
		if err != nil {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
			Logger.Errorf("GET: Failed getting: %s", err)
			return
		} else {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
		}
		jsonDocument, err := json.Marshal(&document)
		if err != nil {
			Logger.Errorf("GET: Can`t marshal: %s", err)
		}
		fmt.Fprintf(w, "%s\n", string(jsonDocument))

	case "POST":

		var result inventory

		doc := r.URL.Path[len("/manager/"):]
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusBadRequest)).Inc()
			Logger.Errorf("Incorrect body request: %s", err)
		}
		err = json.Unmarshal(body, &result)
		if err != nil {
			Logger.Errorf("POST: Can't unmarshal: %s", err)
		} else {
			_, err = bucket.Upsert(doc, result, 0)
			if err != nil {
				totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
				Logger.Debugf("POST: Can't upsert: %s", err)
			} else {
				totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusCreated)).Inc()
			}
		}

	case "DELETE":

		doc := r.URL.Path[len("/manager/"):]
		_, err := bucket.Remove(doc, 0)
		if err != nil {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
			Logger.Errorf("DELETE: Deleting failed: %s", err)
		} else {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
		}

	case "UPDATE":

		doc := r.URL.Path[len("/manager/"):]

		var document inventory

		cas, err := bucket.GetAndLock(doc, 10, &document) //TODO: set time lock
		if err != nil {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
			Logger.Errorf("UPDATE: Failed getting and locking: %s", err)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			Logger.Errorf("UPDATE: Incorrect body request: %s", err)
		}
		err = json.Unmarshal(body, &document)
		if err != nil {
			Logger.Errorf("UPDATE: Can't unmarshal: %s", err)
		}
		cas, err = bucket.Replace(doc, &document, cas, 0)
		if err != nil {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusNotFound)).Inc()
			Logger.Errorf("UPDATE: Failed replace document: %s", err)
		} else {
			totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusOK)).Inc()
		}
		bucket.Unlock(doc, cas)

	default:

		totalRequestHttp.WithLabelValues(strconv.Itoa(http.StatusMethodNotAllowed)).Inc()
		Logger.Errorf("DEFAULT MANAGER: Incorrect method!")
		fmt.Println("Error: ", "\"", r.Method, "\"", " - unknown method. Using GET, POST, DELETE, UPDATE method.")
	}
}
