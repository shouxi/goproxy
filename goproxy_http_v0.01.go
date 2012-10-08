package main

import (
	"net/http"
	"io/ioutil"
	"log"
	"os"
	"io"
)

func handler(w http.ResponseWriter, r *http.Request) {

	log.Println("----------------------------------")
	log.Println("RequestURI", r.RequestURI )
	log.Println("RemoteAddr", r.RemoteAddr)
	//log.Println("URL", r.URL)
	log.Println("----------------------------------")

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	r.RequestURI = ""

	//log.Println("RequestURI", r.RequestURI)
	resp, err := http.DefaultClient.Do(r)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(resp.StatusCode)
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		panic(err)
	}
	w.Write(result)
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Start serving on port 8000")
	http.ListenAndServe(":8000", nil)
	os.Exit(0)
}
