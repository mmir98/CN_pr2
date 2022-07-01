package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	NOTIF_SERVER_GET_NOTIFS_URL    = "http://localhost:8080/notifs"
	NODE_DIRECTORRY_GET_RELAYS_URL = "http://localhost:9090/nodes"

	PAYLOAD_ENCRYPTED_TYPE = "ENC"
	PAYLOAD_REQUEST_TYPE   = "REQ"

	GET_METHOD  = "GET"
	POST_METHOD = "POST"

	NEW_vC_PATH      = "/new-vc"
	FORWARD_API_PATH = "/forward"
)

// {
// vc-id, payload-type,
// payload { method, url, body }
// }

type final_payload_struct struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   []byte `json:"body"`
}

type relay_payload_struct struct {
	VC_ID        string `json:"vc_id"`
	Payload_type string `json:"payload_type"`
	Payload      []byte `json:"payload"`
}

type relay_node_struct struct {
	Name           string `json:"name"`
	Port           int    `json:"port"`
	Directory_node string `json:"dir_node"`
}

type vc_nodes_struct struct {
	entry_node  relay_node_struct
	middle_node relay_node_struct
	exit_node   relay_node_struct
}

func main() {
	log.Println("Client started...")
	nodes := getRelayNodes()
	fmt.Println("Choose three relay nodes :")
	for i := 0; i < len(nodes); i++ {
		fmt.Printf("%d. %s \t %d\n", i+1, nodes[i].Name, nodes[i].Port)
	}
	selected_nodes := getSelectedRelays(nodes)
	new_vc := vc_nodes_struct{
		entry_node:  selected_nodes[0],
		middle_node: selected_nodes[1],
		exit_node:   selected_nodes[2],
	}
	vc_id := "123456"
	vc_w_entry := create_vc_with_entry_node(vc_id, new_vc.entry_node)
	if vc_w_entry == false {

	}
	vc_w_middle := extend_vc_with_middle_node(vc_id, new_vc.middle_node, new_vc.entry_node)
	if vc_w_middle == false {

	}
	log.Println("extending with middle node was successful")
	vc_w_exit := extend_vc_with_exit_node(vc_id, new_vc.exit_node, new_vc.middle_node, new_vc.entry_node)
	if vc_w_exit == false {
		
	}
	log.Println("extending vc with exit node was successful")
	
	// final_payload := final_payload_struct{
	// 	Method: GET_METHOD,
	// 	URL:    NOTIF_SERVER_GET_NOTIFS_URL,
	// }
	// json_payload, err := json.Marshal(final_payload)
	// if err != nil {

	// }
	// relay_3_payload := relay_payload_struct{
	// 	VC_ID:        "1000",
	// 	Payload_type: PAYLOAD_REQUEST_TYPE,
	// 	Payload:      json_payload,
	// }
	// json_relay_3, err := json.Marshal(relay_3_payload)
	// if err != nil {

	// }
	// resp, err := http.Post("http://localhost:10000/it-is-working", "application/json", bytes.NewBuffer(json_relay_3))
	// if err != nil {

	// }
	// defer resp.Body.Close()
	// log.Println(resp.StatusCode)

}

func getRelayNodes() []relay_node_struct {
	log.Println("Sending Get request to dir_node...")
	resp, err := http.Get(NODE_DIRECTORRY_GET_RELAYS_URL)
	if err != nil {

	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {

	}
	nodes_json_list, err := ioutil.ReadAll(resp.Body)
	if err != nil {

	}
	var nodes []relay_node_struct
	json_err := json.Unmarshal(nodes_json_list, &nodes)
	if json_err != nil {

	}

	return nodes
}

func getSelectedRelays(nodes []relay_node_struct) []relay_node_struct {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	var n []relay_node_struct
	for _, f := range strings.Fields(scanner.Text()) {
		i, err := strconv.Atoi(f)
		if err == nil {
			n = append(n, nodes[i-1])
		}
	}

	return n
}

func create_vc_with_entry_node(vc_id string, node relay_node_struct) bool {
	args := new_vc_struct{
		VC_id: vc_id,
	}
	json_body, err := json.Marshal(args)
	if err != nil {

	}
	resp, err := http.Post("http://localhost:"+strconv.Itoa(node.Port)+NEW_vC_PATH, "application/json", bytes.NewBuffer(json_body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return false
	}
	return true
}

func extend_vc_with_middle_node(vc_id string, middle_node relay_node_struct, entry_node relay_node_struct) bool {
	args := new_vc_struct{
		VC_id: vc_id,
		Incomming_port: entry_node.Port,
	}
	args_json, err := json.Marshal(args)
	if err != nil {
		
	}
	final_payload := final_payload_struct{
		Method: POST_METHOD,
		URL:    "http://localhost:" + strconv.Itoa(middle_node.Port) + NEW_vC_PATH,
		Body:   args_json,
	}
	json_payload, err := json.Marshal(final_payload)
	if err != nil {

	}
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      json_payload,
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(string(json_node_1))
	resp, err := http.Post("http://localhost:"+strconv.Itoa(entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	return true
}

type new_vc_struct struct {
	VC_id          string `json:"vc_id"`
	Incomming_port int    `json:"incomming_port"`
}

func extend_vc_with_exit_node(vc_id string, exit_node relay_node_struct, middle_node relay_node_struct, entry_node relay_node_struct) bool{
	args := new_vc_struct{
		VC_id: vc_id,
		Incomming_port: middle_node.Port,
	}
	args_json, err := json.Marshal(args)
	if err != nil {
		
	}
	final_payload := final_payload_struct{
		Method: POST_METHOD,
		URL:    "http://localhost:" + strconv.Itoa(exit_node.Port) + NEW_vC_PATH,
		Body:   args_json,
	}
	json_payload, err := json.Marshal(final_payload)
	if err != nil {

	}
	node_2_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      json_payload,
	}
	json_node_2, err := json.Marshal(node_2_payload)
	if err != nil {
		log.Println(err)
		return false
	}
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      json_node_2,
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(string(json_node_1))
	resp, err := http.Post("http://localhost:"+strconv.Itoa(entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	return true
}
