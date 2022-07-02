package main

import (
	"bufio"
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
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	NOTIF_SERVER_GET_NOTIFS_URL       = "http://localhost:8080/notifs"
	NOTIF_SERVER_CREATE_NEW_NOTIF_URL = "http://localhost:8080/notifs/add"
	NODE_DIRECTORRY_GET_RELAYS_URL    = "http://localhost:9090/nodes"
	NODE_DIRECTORRY_SELECT_RELAYS_URL = "http://localhost:9090/nodes/select"

	PAYLOAD_ENCRYPTED_TYPE = "ENC"
	PAYLOAD_REQUEST_TYPE   = "REQ"

	GET_METHOD  = "GET"
	POST_METHOD = "POST"

	NEW_vC_PATH      = "/new-vc"
	FORWARD_API_PATH = "/forward"
)

type notification_struct struct {
	Author string `json:"author"`
	Text   string `json:"text"`
}

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
	Name           string   `json:"name"`
	Port           int      `json:"port"`
	Directory_node string   `json:"dir_node"`
	Key            *big.Int `json:"key"`
}

type vc_nodes_struct struct {
	Entry_node  relay_node_struct `json:"entry_node"`
	Middle_node relay_node_struct `json:"middle_node"`
	Exit_node   relay_node_struct `json:"exit_node"`
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
		Entry_node:  selected_nodes[0],
		Middle_node: selected_nodes[1],
		Exit_node:   selected_nodes[2],
	}
	vc_id, err := sendSelectedNodesAndGetVCID(new_vc)
	if err != nil {
		log.Println("Cant aquire a new vc_id : " + err.Error())
		return
	}
	log.Println("New vc_id aquired : " + vc_id)
	vc_w_entry := create_vc_with_entry_node(vc_id, &new_vc.Entry_node)
	if vc_w_entry == false {
		log.Println("Cant create vc with entry node.")
		return
	}
	vc_w_middle := extend_vc_with_middle_node(vc_id, &new_vc.Middle_node, new_vc.Entry_node)
	if vc_w_middle == false {
		log.Println("Cant extend vc with middle node.")
		return
	}
	log.Println("extending with middle node was successful")
	vc_w_exit := extend_vc_with_exit_node(vc_id, &new_vc.Exit_node, &new_vc.Middle_node, new_vc.Entry_node)
	if vc_w_exit == false {
		log.Println("Cant extend vc with exit node.")
		return
	}
	log.Println("extending vc with exit node was successful")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nEnter 1 to get all notifs\nOr 2 to create a new notif\norEnter 'end' to exit :")
		scanner.Scan()
		input := scanner.Text()
		// Get all notifs
		if input == "1" {
			notifs, err := getAllNotifs(vc_id, new_vc)
			if err != nil {
				log.Println("Cant get notif list from server. err : " + err.Error())
			}
			fmt.Println("Notif list on server : ")
			if len(notifs) == 0 {
				fmt.Println("notif-list is empty.")
			}
			for i := 0; i < len(notifs); i++ {
				fmt.Printf("\n\t%d. Author : %s\n\tText : %s", i+1, notifs[i].Author, notifs[i].Text)
			}
		}
		// Create a new notif
		if input == "2" {
			fmt.Println("Enter author's name :")
			scanner.Scan()
			author := scanner.Text()
			fmt.Println("Enter notif's text :")
			scanner.Scan()
			text := scanner.Text()
			log.Println("new notif created :\n\tauthor : " + author + "\n\ttext : " + text)
			res := createAndSendNewNotif(author, text, vc_id, new_vc)
			if res == true {
				log.Println("New notif has been added to notif_server's list.")
			}
		}
		if input == "end" {
			break
		}
	}

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

func sendSelectedNodesAndGetVCID(vc_nodes vc_nodes_struct) (string, error) {
	json_body, err := json.Marshal(vc_nodes)
	if err != nil {
		return "", err
	}
	res, err := http.Post(NODE_DIRECTORRY_SELECT_RELAYS_URL, "application/json", bytes.NewBuffer(json_body))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func create_vc_with_entry_node(vc_id string, node *relay_node_struct) bool {
	p, g, a, g_a_mod_p := initializeValuesForDiffi_Hellman()
	args := new_vc_struct{
		VC_id:     vc_id,
		P:         p,
		G:         g,
		G_A_MOD_P: g_a_mod_p,
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

	}
	g_b_mod_p := new(big.Int)
	g_b_mod_p.SetBytes(body)
	key := new(big.Int)
	key.Exp(g_b_mod_p, a, p)
	node.Key = key
	// log.Println(key)
	log.Println("Entry node secret key : " + node.Key.String())
	return true
}

func initializeValuesForDiffi_Hellman() (*big.Int, *big.Int, *big.Int, *big.Int) {
	p, err := rand.Prime(rand.Reader, 256)
	if err != nil {
		log.Println("Cant create prime rand number. err : " + err.Error())
	}
	g, err := rand.Int(rand.Reader, p)
	if err != nil {
		log.Println("Cant create g (base) rand number. err : " + err.Error())
	}
	a, err := rand.Int(rand.Reader, p)
	if err != nil {
		log.Println("Cant create secret rand number. err : " + err.Error())
	}
	var g_a_mod_p *big.Int = new(big.Int)
	g_a_mod_p.Exp(g, a, p)
	return p, g, a, g_a_mod_p
}

func extend_vc_with_middle_node(vc_id string, middle_node *relay_node_struct, entry_node relay_node_struct) bool {
	p, g, a, g_a_mod_p := initializeValuesForDiffi_Hellman()
	args := new_vc_struct{
		VC_id:          vc_id,
		Incomming_port: entry_node.Port,
		P:              p,
		G:              g,
		G_A_MOD_P:      g_a_mod_p,
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
	// log.Println(json_payload)
	// log.Println(string(json_payload))
	encrypted_node_1_payload := AES_encryptor(entry_node.Key.Bytes(), string(json_payload))
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      []byte(encrypted_node_1_payload),
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		log.Println(err)
		return false
	}

	log.Println(string(encrypted_node_1_payload))
	log.Println(json_node_1)
	resp, err := http.Post("http://localhost:"+strconv.Itoa(entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	decrypted_body := AES_decryptor(entry_node.Key.Bytes(), string(body))
	g_b_mod_p := new(big.Int)
	g_b_mod_p.SetBytes([]byte(decrypted_body))
	key := new(big.Int)
	key.Exp(g_b_mod_p, a, p)
	middle_node.Key = key
	log.Println("Middle node secret key : " + middle_node.Key.String())
	return true
}

type new_vc_struct struct {
	VC_id          string   `json:"vc_id"`
	Incomming_port int      `json:"incomming_port"`
	P              *big.Int `json:"p"`
	G              *big.Int `json:"g"`
	G_A_MOD_P      *big.Int `json:"g_a_mod_p"`
}

func extend_vc_with_exit_node(vc_id string, exit_node *relay_node_struct, middle_node *relay_node_struct, entry_node relay_node_struct) bool {
	p, g, a, g_a_mod_p := initializeValuesForDiffi_Hellman()
	args := new_vc_struct{
		VC_id:          vc_id,
		Incomming_port: middle_node.Port,
		P:              p,
		G:              g,
		G_A_MOD_P:      g_a_mod_p,
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

	encrypted_node_2_payload := AES_encryptor(middle_node.Key.Bytes(), string(json_payload))
	node_2_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      []byte(encrypted_node_2_payload),
	}
	json_node_2, err := json.Marshal(node_2_payload)
	if err != nil {
		log.Println(err)
		return false
	}
	encrypted_node_1_payload := AES_encryptor(entry_node.Key.Bytes(), string(json_node_2))
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      []byte(encrypted_node_1_payload),
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		log.Println(err)
		return false
	}

	// log.Println(string(json_node_1))
	resp, err := http.Post("http://localhost:"+strconv.Itoa(entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	if resp.StatusCode != http.StatusCreated {
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	node_1_decrypted_body := AES_decryptor(entry_node.Key.Bytes(), string(body))
	node_2_decrypted_body := AES_decryptor(middle_node.Key.Bytes(), string(node_1_decrypted_body))
	g_b_mod_p := new(big.Int)
	g_b_mod_p.SetBytes([]byte(node_2_decrypted_body))
	key := new(big.Int)
	key.Exp(g_b_mod_p, a, p)
	exit_node.Key = key
	log.Println(key)
	return true
}

func getAllNotifs(vc_id string, vc_nodes vc_nodes_struct) ([]notification_struct, error) {
	final_payload := final_payload_struct{
		Method: GET_METHOD,
		URL:    NOTIF_SERVER_GET_NOTIFS_URL,
	}
	json_payload, err := json.Marshal(final_payload)
	if err != nil {
		return nil, err
	}
	encrypted_node_3_payload := AES_encryptor(vc_nodes.Exit_node.Key.Bytes(), string(json_payload))
	node_3_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      []byte(encrypted_node_3_payload),
	}
	json_node_3, err := json.Marshal(node_3_payload)
	if err != nil {
		return nil, err
	}
	encrypted_node_2_payload := AES_encryptor(vc_nodes.Middle_node.Key.Bytes(), string(json_node_3))
	node_2_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      []byte(encrypted_node_2_payload),
	}
	json_node_2, err := json.Marshal(node_2_payload)
	if err != nil {
		return nil, err

	}
	encrypted_node_1_payload := AES_encryptor(vc_nodes.Entry_node.Key.Bytes(), string(json_node_2))
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      []byte(encrypted_node_1_payload),
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		return nil, err
	}
	// log.Println(string(json_node_1))

	resp, err := http.Post("http://localhost:"+strconv.Itoa(vc_nodes.Entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	node_1_decrypted_body := AES_decryptor(vc_nodes.Entry_node.Key.Bytes(), string(body))
	node_2_decrypted_body := AES_decryptor(vc_nodes.Middle_node.Key.Bytes(), string(node_1_decrypted_body))
	node_3_decrypted_body := AES_decryptor(vc_nodes.Exit_node.Key.Bytes(), string(node_2_decrypted_body))
	var notifs []notification_struct
	if err := json.Unmarshal([]byte(node_3_decrypted_body), &notifs); err != nil {
		return nil, err
	}
	return notifs, nil
}

func createAndSendNewNotif(author string, text string, vc_id string, vc_nodes vc_nodes_struct) bool {
	args := notification_struct{
		Author: author,
		Text:   text,
	}
	args_json, err := json.Marshal(args)
	if err != nil {
		log.Println("Cant marshal args. err : " + err.Error())
		return false
	}
	final_payload := final_payload_struct{
		Method: POST_METHOD,
		URL:    NOTIF_SERVER_CREATE_NEW_NOTIF_URL,
		Body:   args_json,
	}
	json_payload, err := json.Marshal(final_payload)
	if err != nil {
		log.Println("Cant marshal final_payload. err : " + err.Error())
		return false
	}
	encrypted_node_3_payload := AES_encryptor(vc_nodes.Exit_node.Key.Bytes(), string(json_payload))
	node_3_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_REQUEST_TYPE,
		Payload:      []byte(encrypted_node_3_payload),
	}
	json_node_3, err := json.Marshal(node_3_payload)
	if err != nil {
		log.Println("Cant marshal node_3_payload. err : " + err.Error())
		return false
	}
	encrypted_node_2_payload := AES_encryptor(vc_nodes.Middle_node.Key.Bytes(), string(json_node_3))
	node_2_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      []byte(encrypted_node_2_payload),
	}
	json_node_2, err := json.Marshal(node_2_payload)
	if err != nil {
		log.Println("Cant marshal node_2_payload. err : " + err.Error())
		return false
	}
	encrypted_node_1_payload := AES_encryptor(vc_nodes.Entry_node.Key.Bytes(), string(json_node_2))
	node_1_payload := relay_payload_struct{
		VC_ID:        vc_id,
		Payload_type: PAYLOAD_ENCRYPTED_TYPE,
		Payload:      []byte(encrypted_node_1_payload),
	}
	json_node_1, err := json.Marshal(node_1_payload)
	if err != nil {
		log.Println("Cant marshal node_1_payload. err : " + err.Error())
		return false
	}
	// log.Println(string(json_node_1))
	resp, err := http.Post("http://localhost:"+strconv.Itoa(vc_nodes.Entry_node.Port)+FORWARD_API_PATH, "application/json", bytes.NewBuffer(json_node_1))
	if err != nil {
		log.Println("Cant post newNotif msg. err : " + err.Error())
		return false
	}
	defer resp.Body.Close()
	log.Println("response statusCode for newNotif request. Response status : " + resp.Status)
	if resp.StatusCode != http.StatusCreated {
		return false
	}
	return true
}

func AES_encryptor(key []byte, stringToEncrypt string) (encryptedString string) {
	plaintext := []byte(stringToEncrypt)

	c, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	aesGCM, err := cipher.NewGCM(c)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext)
}

func AES_decryptor(key []byte, encryptedString string) (decryptedString string) {
	enc, _ := hex.DecodeString(encryptedString)

	c, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	aesGCM, err := cipher.NewGCM(c)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return fmt.Sprintf("%s", plaintext)
}
