package main

import (
	"fmt"
	"net/http"
)

type Notification struct {
	id int
	title string
	description string
}

var notification_list []Notification = make([]Notification, 0)

const notif_create_url = "/notifs/add"
const notif_get_url = "/notifs"

func main() {
	fmt.Println("Hello world!")
	http.HandleFunc("/", handler)

	fmt.Println(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request){

	// Not found 404
	if r.URL.Path != notif_create_url && r.URL.Path != notif_get_url {
		http.NotFound(w, r)
		return
	}

	if r.URL.Path == notif_create_url && r.Method == "POST" {
		create_new_notif(w, r)
		return
	}

	if r.URL.Path == notif_get_url && r.Method == "GET" {
		retrieve_notifs(w, r)
		return
	}

	// Not implemented 503
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
}

func create_new_notif(w *http.ResponseWriter, r http.Request){

}

func retrieve_notifs(w *http.ResponseWriter, r http.Request){

}