package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type node struct {
	name           string
	port           int
	directory_name string
}

const (
	DIR_URL = "http://localhost"
	ADD_NODE_API_PATH = "/nodes/add"
	REMOVE_NODE_API_PATH = "/nodes/remove"
)

func main() {
	node_info := node{
		name:           "node 1",
		port:           10000,
		directory_name: "9090",
	}

	l, err := net.Listen("tcp", ":"+strconv.Itoa(node_info.port))
	if err != nil {
		log.Println("node " + node_info.name + " is unable to listen on port " + strconv.Itoa(node_info.port) + " err : " + err.Error())
		return
	}

	log.Println("node started on port : " + strconv.Itoa(node_info.port))
	if notify_dir_im_alive(node_info) == false {
		log.Println("node " + node_info.name + " can't notify node_directory")
		return // TODO maybe put it in a loop
	}

	if err := http.Serve(l, http.HandlerFunc(handler)); err != nil {
		notify_dir_im_dead(node_info)
	}

}

func notify_dir_im_alive(node_info node) bool {
	log.Println("Notifying directory_node I'm alive...")
	resp, err := http.PostForm(DIR_URL + ":" + node_info.directory_name + ADD_NODE_API_PATH,
		url.Values{"name": {node_info.name}, "dir_node": {node_info.directory_name}, "port": {strconv.Itoa(node_info.port)}})
	if err != nil {
		log.Println("node " + node_info.name + " can't send post req to notify dir_node. err :" + err.Error() )
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		log.Println("directory_node successfully notified")
		return true
	}
	// if resp.StatusCode == http.StatusInternalServerError {
	// 	return false		
	// }
	// if resp.StatusCode == http.StatusNotAcceptable {
	// 	return false
	// }
	return false
}

func notify_dir_im_dead(node_info node) {
	log.Println("Notifying dir_node I'm dead...")
	resp, err := http.PostForm(DIR_URL + ":" + node_info.directory_name + REMOVE_NODE_API_PATH,
		url.Values{"name": {node_info.name}})
	if err != nil {
		log.Println("node " + node_info.name + "can't send remove req to dir_node. err : " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		log.Println("remove response came with status OK")
	}
}

func handler(w http.ResponseWriter, r *http.Request) {

}
