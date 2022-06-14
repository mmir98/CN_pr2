package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Notification struct {
	Author string	`json:"author"`
	Text string		`json:"text"`
}

var notification_list []Notification = make([]Notification, 0)

const notif_create_url = "/notifs/add"
const notif_get_url = "/notifs"

func main() {
	http.HandleFunc("/", logger(handler))

	log.Println("Notifs Server running on port 8080...")
	log.Println(http.ListenAndServe(":8080", nil))
}

func logger (f http.HandlerFunc) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request)  {
		log.Println(r.Method, r.URL)
		f(w, r)
	}
}

func handler(w http.ResponseWriter, r *http.Request){
	// Not found 404
	if r.URL.Path != notif_create_url && r.URL.Path != notif_get_url {
		log.Println(http.StatusText(http.StatusNotFound))
		http.NotFound(w, r)
		return
	}

	// Creating new notif
	if r.URL.Path == notif_create_url && r.Method == "POST" {
		create_new_notif(w, r)
		return
	}

	// Retrieving all notifs
	if r.URL.Path == notif_get_url && r.Method == "GET" {
		retrieve_notifs(w, r)
		return
	}

	// Not implemented 503
	log.Println(http.StatusText(http.StatusNotImplemented))
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
}

// * Create new Notif handler
func create_new_notif(w http.ResponseWriter, r *http.Request){
	var author string = r.FormValue("author")
	var text string = r.FormValue("text")

	newNotif := Notification{
		Author: author,
		Text: text,
	}
	notification_list = append(notification_list, newNotif)
	log.Println("New notif created : ",  newNotif)

	w.WriteHeader(http.StatusCreated)
	// w.Write([]byte(http.StatusText(http.StatusCreated)))
}

// * Retrive notif handler
func retrieve_notifs(w http.ResponseWriter, r *http.Request){
	log.Println("retrieving notif list...")
	res, err := json.Marshal(notification_list)
	if err != nil {
		log.Println("retrieving notif list failed : ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write(res)
	log.Println("Notif list retrieved")
}