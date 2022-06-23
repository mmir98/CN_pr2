package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type node struct {
	Name string 	`json:"name"`
	Port int		`json:"port"`
	Directory_node string	`json:"dir_node"`
}

var online_nodes_list []node = make([]node, 0)
const (
	NODE_ONLINE = "/nodes/add"
	NODE_OFFLINE = "/nodes/remove"
	NODE_LIST = "/nodes"
	POST_METHOD = "POST"
	GET_METHOD = "GET"
)

func main() {
	http.Handle("/", logger(handler))

	log.Println("Node-dir running on port 9090 ...")
	log.Println(http.ListenAndServe(":9090", nil))
}

func logger (f http.HandlerFunc) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request){
		log.Println(r.Method, r.URL)
		f(w, r)
	}
}

func handler(w http.ResponseWriter, r *http.Request){
	// url not found
	if r.URL.Path != NODE_OFFLINE && r.URL.Path != NODE_ONLINE && r.URL.Path != NODE_LIST {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == NODE_ONLINE && r.Method == POST_METHOD {
		node_came_online(w, r)
		return
	}
	if r.URL.Path == NODE_OFFLINE && r.Method == POST_METHOD {
		node_went_offline(w, r)	
		return
	}
	if r.URL.Path == NODE_LIST && r.Method == GET_METHOD {
		retrieve_nodes(w, r)
		return
	}

	// service not implemented
	w.WriteHeader(http.StatusNotImplemented)
}

func node_came_online (w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	dir_node := r.FormValue("dir_node")
	port, err := strconv.Atoi(r.FormValue("port"))
	if err != nil {
		log.Println("Invalid port number. err : ", err)
		res, err := json.Marshal(map[string] string{"error": "Invalid port number"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(res)
		return
	}
	newNode := node{
		Name: name,
		Directory_node: dir_node,
		Port: port,
	}
	online_nodes_list = append(online_nodes_list, newNode)
	log.Println("new node added to online list : ", newNode)

	w.WriteHeader(http.StatusCreated)
}

func node_went_offline (w http.ResponseWriter, r *http.Request){
	name := r.FormValue("name")
	list_len := len(online_nodes_list)
	for i := 0; i < list_len; i++ {
		if online_nodes_list[i].Name == name {
			log.Println("node removed from online list : ", online_nodes_list[i])
			online_nodes_list[i] = online_nodes_list[list_len - 1]
			online_nodes_list = online_nodes_list[:list_len - 1]
			break
		}
	}
	
	w.WriteHeader(http.StatusOK)
}

func retrieve_nodes (w http.ResponseWriter, r *http.Request) {
	log.Println("retreiving online nodes ...")
	res, err := json.Marshal(online_nodes_list)
	if err != nil {
		log.Println("retreiving online nodes failed. err : ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("online nodes retreived")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

