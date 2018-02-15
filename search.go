package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/couchbase/gocb"
	"github.com/couchbase/gocb/cbft"
)

func search(w http.ResponseWriter, r *http.Request) {

	var answer []Inventory

	body, error := ioutil.ReadAll(r.Body)
	if error != nil {
		fmt.Println(error.Error()) //TODO: обработка ошибки !!!
	}

	search := make(map[string]interface{})

	err := json.Unmarshal(body, &search)
	if err != nil {
		log.Println(err)
		return
	}

	// слайс для хранения запроса
	res := []cbft.FtsQuery{}

	// получаем имя дока
	doc := r.URL.Path[len("/search/"):]

	// если имя дока было указано - добавляем в запрос
	if doc != "" {
		res = append(res, cbft.NewDocIdQuery(doc))
	}

	for key, val := range search {
		fmt.Println("Key:", key, "Val:", val)

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
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, hit := range rows.Hits() {
		var ans Inventory
		_, err := bucket.Get(hit.Id, &ans)
		if err != nil {
			fmt.Println(err.Error())
		}
		answer = append(answer, ans)

	}

	jsonDocument, err := json.Marshal(&answer)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Fprintf(w, "%v\n", string(jsonDocument))
}
