package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
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
	VC_id          string
	incomming_port int
	P              *big.Int
	G              *big.Int
	G_a_mod_p      *big.Int
}
type vc struct {
	id        string
	key       *big.Int
	pre_node  node
	next_node node
}

var circuits = make([]vc, 0)

func main() {
	node_info := node{
		name:           "node 1",
		port:           11000,
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
		return 
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
	json_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()

	var request new_vc_struct
	if err := json.Unmarshal(json_body, &request); err != nil {
		log.Panicln(err.Error())
	}
	log.Println("creating new vc with vc_id : " + request.VC_id)
	b, err := rand.Int(rand.Reader, request.P)
	if err != nil {
		log.Panicln(err.Error())
	}
	g_b_mod_p := new(big.Int)
	g_b_mod_p.Exp(request.G, b, request.P)

	key := new(big.Int)
	key.Exp(request.G_a_mod_p, b, request.P)
	log.Println(key)
	newVC := vc{
		id:       request.VC_id,
		pre_node: node{port: request.incomming_port},
		key:      key,
	}
	circuits = append(circuits, newVC)

	w.WriteHeader(http.StatusCreated)
	w.Write(g_b_mod_p.Bytes())
}

func forward_msg(w http.ResponseWriter, r *http.Request) {
	json_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()

	var request payload_struct
	if err := json.Unmarshal(json_body, &request); err != nil {
		log.Panicln(err.Error())
	}
	log.Println("forwading msg with vc_id : " + request.VC_id + " and payload_type : " + request.Payload_type)
	var circuit_index int
	for i := 0; i < len(circuits); i++ {
		if circuits[i].id == request.VC_id {
			circuit_index = i
			break
		}
	}
	log.Println(request.Payload)
	decrypted_payload := AES_decryptor(circuits[circuit_index].key.Bytes(), string(request.Payload))
	if err != nil {
		log.Println(err)
	}
	if request.Payload_type == PAYLOAD_REQUEST_TYPE {
		var final_payload final_payload_struct
		err := json.Unmarshal([]byte(decrypted_payload), &final_payload)
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
				log.Panicln(err.Error())
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
				log.Panicln(err.Error())
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Panicln(err.Error())
			}
			log.Println(res.StatusCode)
			encrypted_body := AES_encryptor(circuits[circuit_index].key.Bytes(), string(body))
			w.WriteHeader(res.StatusCode)
			w.Write([]byte(encrypted_body))
		}
		if final_payload.Method == POST_METHOD {
			res, err := http.Post(final_payload.URL, "application/json", bytes.NewBuffer(final_payload.Body))
			if err != nil {
				log.Println(err)
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			log.Println("response status code of post request : " + res.Status)
			encrypted_body := AES_encryptor(circuits[circuit_index].key.Bytes(), string(body))
			w.WriteHeader(res.StatusCode)
			w.Write([]byte(encrypted_body))
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
		resp, err := http.Post("http://localhost:"+strconv.Itoa(circuits[cir_index].next_node.port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer([]byte(decrypted_payload)))
		if err != nil {
			log.Panicln(err.Error())
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panicln(err.Error())
		}
		encrypted_body := AES_encryptor(circuits[cir_index].key.Bytes(), string(body))
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte(encrypted_body))
	}

}

func AES_encryptor(key []byte, stringToEncrypt string) (encryptedString string) {
	plaintext := []byte(stringToEncrypt)

	c, err := aes.NewCipher(key)
	if err != nil {
		log.Panicln(err.Error())
	}
	aesGCM, err := cipher.NewGCM(c)
	if err != nil {
		log.Panicln(err.Error())
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		log.Panicln(err.Error())
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext)
}

func AES_decryptor(key []byte, encryptedString string) (decryptedString string) {
	enc, _ := hex.DecodeString(encryptedString)

	c, err := aes.NewCipher(key)
	if err != nil {
		log.Panicln(err.Error())
	}
	aesGCM, err := cipher.NewGCM(c)
	if err != nil {
		log.Panicln(err.Error())
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Panicln(err.Error())
	}
	return fmt.Sprintf("%s", plaintext)
}
