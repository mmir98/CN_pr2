package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	DIR_URL              = "http://localhost"
	ADD_NODE_API_PATH    = "/nodes/add"
	REMOVE_NODE_API_PATH = "/nodes/remove"
	NEW_VC_API_PATH      = "/new-vc"
	FORWARD_API_PATH     = "/forward"

	GET_METHOD  = "GET"
	POST_METHOD = "POST"

	PAYLOAD_ENCRYPTED_TYPE = "ENC"
	PAYLOAD_REQUEST_TYPE   = "REQ"

	VC_ID_FIELD          = "vc_id"
	PAYLOAD_TYPE_FIELD   = "payload_type"
	PAYLOAD_FIELD        = "payload"
	PAYLOAD_METHOD_FIELD = "method"
	PAYLOAD_URL_FIELD    = "url"
	PAYLOAD_BODY_FIELD   = "body"
)

type node struct {
	name           string
	port           int
	directory_name string
}

type vc struct {
	id        string
	key       string
	pre_node  node
	next_node node
}

var circuits = make([]vc, 0)

func main() {
	node_info := node{
		name:           "node 3",
		port:           13000,
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
	resp, err := http.PostForm(DIR_URL+":"+node_info.directory_name+ADD_NODE_API_PATH,
		url.Values{"name": {node_info.name}, "dir_node": {node_info.directory_name}, "port": {strconv.Itoa(node_info.port)}})
	if err != nil {
		log.Println("node " + node_info.name + " can't send post req to notify dir_node. err :" + err.Error())
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
	resp, err := http.PostForm(DIR_URL+":"+node_info.directory_name+REMOVE_NODE_API_PATH,
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
	log.Println(r.URL.Path)
	if r.URL.Path == NEW_VC_API_PATH && r.Method == POST_METHOD {
		create_new_vc(w, r)
		return
	}
	if r.URL.Path == FORWARD_API_PATH && r.Method == POST_METHOD {
		forward_msg(w, r)
		return
	}

	w.WriteHeader(http.StatusNotImplemented)

}

func create_new_vc(w http.ResponseWriter, r *http.Request) {
	// vc_id := r.FormValue(VC_ID_FIELD)
	json_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()
	log.Println(string(json_body))
	var request new_vc_struct
	if err := json.Unmarshal(json_body, &request); err != nil {

	}
	
	log.Println("creating new vc with vc_id : " + request.VC_id)

	// pre_node_port, err := strconv.Atoi(r.FormValue("incomming_port"))
	// if err != nil {

	// }
	newVC := vc{
		id:       request.VC_id,
		pre_node: node{port: request.incomming_port},
	}
	circuits = append(circuits, newVC)

	w.WriteHeader(http.StatusCreated)
}

func forward_msg(w http.ResponseWriter, r *http.Request) {
	// vc_id := r.FormValue(VC_ID_FIELD)
	// payload_type := r.FormValue(PAYLOAD_TYPE_FIELD)

	json_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()
	log.Println(string(json_body))
	var request payload_struct
	if err := json.Unmarshal(json_body, &request); err != nil {

	}
	log.Println("forwading msg with vc_id : " + request.VC_id + " and payload_type : " + request.Payload_type)
	if request.Payload_type == PAYLOAD_REQUEST_TYPE {
		var final_payload final_payload_struct
		err := json.Unmarshal([]byte(request.Payload), &final_payload)
		if err != nil {
			log.Println("error occured while trying to decode final_payload. err : " + err.Error())

			return
		}
		if strings.Contains(final_payload.URL, NEW_VC_API_PATH) {
			fields := strings.FieldsFunc(final_payload.URL, func(r rune) bool {
				if r == '/' || r == ':' {
					return true
				}
				return false
			})
			port, err := strconv.Atoi(fields[2])
			if err != nil {
				
			}
			for i := 0; i < len(circuits); i++ {
				if circuits[i].id == request.VC_id {
					circuits[i].next_node = node{port: port}
					break
				}
			}
		}
		if final_payload.Method == GET_METHOD {
			res, err := http.Get(final_payload.URL)
			if err != nil {

			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {

			}
			log.Println(res.StatusCode)
			w.WriteHeader(res.StatusCode)
			w.Write(body)
		}
		if final_payload.Method == POST_METHOD {
			res, err := http.Post(final_payload.URL, "application/json", bytes.NewBuffer(final_payload.Body))
			if err != nil {
				log.Println(err)
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			log.Println("response status code of post request : " + res.Status)
			w.WriteHeader(res.StatusCode)
			w.Write(body)
		}
		return
	}
	if request.Payload_type == PAYLOAD_ENCRYPTED_TYPE {
		var cir_index int
		for i := 0; i < len(circuits); i++ {
			if circuits[i].id == request.VC_id {
				cir_index = i
				break
			}
		}
		resp, err := http.Post("http://localhost:"+strconv.Itoa(circuits[cir_index].next_node.port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(request.Payload))
		if err != nil {
			
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	}

}

type final_payload_struct struct {
	Method string
	URL    string
	Body   []byte
}

type payload_struct struct {
	VC_id        string
	Payload_type string
	Payload      []byte
}

type new_vc_struct struct {
	VC_id string
	incomming_port int
}