package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type node struct {
	Name           string `json:"name"`
	Port           int    `json:"port"`
	Directory_node string `json:"dir_node"`
}

type vc_id_counter struct {
	sync.Mutex
	vc_id int
}

var vc_c_lock vc_id_counter
var online_nodes_list []node = make([]node, 0)

const (
	NODE_ONLINE  = "/nodes/add"
	NODE_OFFLINE = "/nodes/remove"
	NODE_LIST    = "/nodes"
	NODE_SELECT  = "/nodes/select"
	POST_METHOD  = "POST"
	GET_METHOD   = "GET"
)

func main() {
	log.Println("Node_directory started...")
	fmt.Println("Enter port number for node directory server :")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	port_number := scanner.Text()

	vc_c_lock = vc_id_counter{vc_id: 0}
	http.Handle("/", logger(handler))

	log.Println("Node-dir running on port " + port_number + "...")
	log.Println(http.ListenAndServe(":" + port_number, nil))
}

func logger(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL)
		f(w, r)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// url not found
	if r.URL.Path != NODE_OFFLINE && r.URL.Path != NODE_ONLINE && r.URL.Path != NODE_LIST && r.URL.Path != NODE_SELECT {
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
	if r.URL.Path == NODE_SELECT && r.Method == POST_METHOD {
		vc_c_lock.select_node(w, r)
		return
	}

	// service not implemented
	w.WriteHeader(http.StatusNotImplemented)
}

func node_came_online(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	dir_node := r.FormValue("dir_node")
	port, err := strconv.Atoi(r.FormValue("port"))
	if err != nil {
		log.Println("Invalid port number. err : ", err)
		res, err := json.Marshal(map[string]string{"error": "Invalid port number"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write(res)
		return
	}
	newNode := node{
		Name:           name,
		Directory_node: dir_node,
		Port:           port,
	}
	online_nodes_list = append(online_nodes_list, newNode)
	log.Println("new node added to online list : ", newNode)

	w.WriteHeader(http.StatusCreated)
}

func node_went_offline(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	list_len := len(online_nodes_list)
	for i := 0; i < list_len; i++ {
		if online_nodes_list[i].Name == name {
			log.Println("node removed from online list : ", online_nodes_list[i])
			online_nodes_list[i] = online_nodes_list[list_len-1]
			online_nodes_list = online_nodes_list[:list_len-1]
			break
		}
	}

	w.WriteHeader(http.StatusOK)
}

func retrieve_nodes(w http.ResponseWriter, r *http.Request) {
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

func (vc_c *vc_id_counter) select_node(w http.ResponseWriter, r *http.Request) {
	vc_c.Lock()
	defer vc_c.Unlock()
	log.Println("Sending VC_id : " + strconv.Itoa(vc_c.vc_id))
	var sending_vc int = vc_c.vc_id
	vc_c.vc_id = (vc_c.vc_id + 1) % 10000

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(sending_vc)))
}
