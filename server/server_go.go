package main

import (
	"../bclass"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/govec"
	"net"
	"os"
	"time"
)

const BUFFSIZE = 1048576

var (
	cnum      int = 0
	maxnode   int = 0
	myaddr    *net.TCPAddr
	cnumhist  map[int]int
	client    map[string]int
	claddr    map[int]*net.TCPAddr
	tempmodel map[int]aggregate
	testqueue map[int]map[int]bool
	models    map[int]bclass.Model
	modelC    map[int]int
	modelD    int
	channel   chan message
	logger    *govec.GoLog
	l         *net.TCPListener
	gmodel    bclass.GlobalModel
	gempty    bclass.GlobalModel
)

type aggregate struct {
	cnum  int
	model bclass.Model
	c     int
	d     int
}

type message struct {
	Id       int
	NodeIp   string
	NodeName string
	Type     string
	C        int
	D        int
	Model    bclass.Model
	GModel   bclass.GlobalModel
}

func main() {
	//Initialize stuff
	client = make(map[string]int)
	claddr = make(map[int]*net.TCPAddr)
	models = make(map[int]bclass.Model)
	modelC = make(map[int]int)
	modelD = 0
	gmodel = bclass.GlobalModel{models, modelC, modelD}
	tempmodel = make(map[int]aggregate)
	testqueue = make(map[int]map[int]bool)
	cnumhist = make(map[int]int)
	channel = make(chan message)

	go updateGlobal(channel)

	//Parsing inputargs
	parseArgs()

	//Initialize TCP Connection and listener
	l, _ = net.ListenTCP("tcp", myaddr)
	fmt.Printf("Server initialized.\n")

	for {
		conn, err := l.AcceptTCP()
		checkError(err)
		go connHandler(conn)
	}

}

// Function for handling client requests
func connHandler(conn *net.TCPConn) {
	var msg message
	p := make([]byte, BUFFSIZE)
	conn.Read(p)
	logger.UnpackReceive("Received message", p, &msg)
	switch msg.Type {
	case "commit_request":
		// node is sending a model, checking to see if testing is complete
		flag := checkQueue(client[msg.NodeName])
		fmt.Printf("<-- Received commit request from %v.\n", msg.NodeName)
		if flag {
			// accept commit from node and process outgoing test requests
			processTestRequest(msg, conn)
		} else {
			//denied
			conn.Write([]byte("Pending tests are not complete"))
			fmt.Printf("--> Denied commit request from %v.\n", msg.NodeName)
			conn.Close()
		}
	case "global_request":
		//node is requesting the global model, will forward
		conn.Write([]byte("OK"))
		fmt.Printf("<-- Received global model request from %v.\n", msg.NodeName)
		genGlobalModel()
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		// node is submitting test results, update testqueue on all replicas
		fmt.Printf("<-- Received completed test results from %v.\n", msg.NodeName)
		//update the pending commit and merge if complete
		if testqueue[client[msg.NodeName]][cnumhist[msg.Id]] {
			testqueue[client[msg.NodeName]][cnumhist[msg.Id]] = false
			channel <- msg
			conn.Write([]byte("OK"))
		} else {
			// if testqueue is already empty
			conn.Write([]byte("Duplicate Test"))
			fmt.Printf("--> Ignored test results from %v.\n", msg.NodeName)
		}
		conn.Close()
	case "join_request":
		// node is requesting to join or rejoin
		processJoin(msg)
		conn.Write([]byte("Joined"))
		conn.Close()
	default:
		conn.Write([]byte("Unknown Request"))
		fmt.Printf("something weird happened!\n")
		conn.Close()
	}
}

func updateGlobal(ch chan message) {
	// Function that aggregates the global model and commits when ready
	for {
		m := <-ch
		id := cnumhist[m.Id]
		tempAggregate := tempmodel[id]
		tempAggregate.c += m.C
		tempAggregate.d += m.D
		tempmodel[id] = tempAggregate
		if modelD < tempAggregate.d {
			modelD = tempAggregate.d
		}
		if float64(tempAggregate.d) > float64(modelD)*0.6 {
			models[id] = tempAggregate.model
			modelC[id] = tempAggregate.c
			t := time.Now()
			logger.LogLocalEvent(fmt.Sprintf("%s - Committed model%v by %v at partial commit %v.", t.Format("15:04:05.0000"), id, client[m.NodeName], tempAggregate.d/modelD*100.0))
			//logger.LogLocalEvent("commit_complete")
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.cnum)
		}
	}
}

// Generate global model from partial commits
func genGlobalModel() {
	modelstemp := models
	modelCtemp := modelC
	modelDtemp := modelD
	gmodel = bclass.GlobalModel{modelstemp, modelCtemp, modelDtemp}
}

// Function that generates test request following a commit request
func processTestRequest(m message, conn *net.TCPConn) {
	tempcnum := cnum
	cnum++
	cnumhist[tempcnum] = client[m.NodeName]
	//initialize new aggregate
	tempmodel[client[m.NodeName]] = aggregate{tempcnum, m.Model, m.C, m.D}
	for _, id := range client {
		if id != client[m.NodeName] {
			if queue, ok := testqueue[id]; !ok {
				queue := make(map[int]bool)
				queue[cnumhist[tempcnum]] = true
				testqueue[id] = queue
			} else {
				queue[cnumhist[tempcnum]] = true
			}
		}
	}
	fmt.Printf("--- Processed commit %v for node %v.\n", tempcnum, m.NodeName)
	conn.Write([]byte("OK"))
	conn.Close()
	for name, id := range client {
		if id != client[m.NodeName] {
			sendTestRequest(name, id, tempcnum, m.Model)
		}
	}
}

// Function that sends test requests via TCP
func sendTestRequest(name string, id, tcnum int, tmodel bclass.Model) {
	//create test request (sanitized)
	msg := message{tcnum, "server", "server", "test_request", 0, 0, tmodel, gempty}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", cnumhist[tcnum], name)
	err := tcpSend(claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO]\n*** Could not send test request to %v.\n", name)
	}
}

// Function to forward global model
func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.NodeName)
	msg := message{m.Id, "server", "server", "global_grant", 0, 0, m.Model, gmodel}
	tcpSend(claddr[client[m.NodeName]], msg)
}

// Function for sending messages to nodes via TCP
func tcpSend(addr *net.TCPAddr, msg message) error {
	p := make([]byte, BUFFSIZE)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err == nil {
		outbuf := logger.PrepareSend(msg.Type, msg)
		_, err = conn.Write(outbuf)
		//checkError(err)
		n, _ := conn.Read(p)
		if string(p[:n]) != "OK" {
			fmt.Printf(" [NO]\n<-- Request was denied by node: %v.\nEnter command: ", string(p[:n]))
		} else {
			fmt.Printf(" [OK]\n")
		}
	}
	return err
}

// Function that checks the testqueue for outstanding tests
func checkQueue(id int) bool {
	flag := true
	for _, v := range testqueue[id] {
		if flag && v {
			flag = false
		}
	}
	return flag
}

// Function that processes join requests
func processJoin(m message) {
	//process depending on if it is a new node or a returning one
	if _, ok := client[m.NodeName]; !ok {
		//adding a node that has never been added before
		id := maxnode
		maxnode++
		client[m.NodeName] = id
		claddr[id], _ = net.ResolveTCPAddr("tcp", m.NodeIp)
		fmt.Printf("--- Added %v as node%v.\n", m.NodeName, id)
		queue := make(map[int]bool)
		for k, _ := range tempmodel {
			queue[k] = true
		}
		testqueue[id] = queue
		for _, v := range tempmodel {
			sendTestRequest(m.NodeName, id, v.cnum, v.model)
		}
	} else {
		//node is rejoining, update address and resend the unfinished test requests
		id := client[m.NodeName]
		claddr[id], _ = net.ResolveTCPAddr("tcp", m.NodeIp)
		fmt.Printf("--- %v at node%v is back online.\n", m.NodeName, id)
		for k, v := range testqueue[id] {
			if v {
				aggregatesendtest := tempmodel[k]
				sendTestRequest(m.NodeName, id, aggregatesendtest.cnum, aggregatesendtest.model)
			}
		}
	}
}

// Input parser
func parseArgs() {
	flag.Parse()
	inputargs := flag.Args()
	var err error
	if len(inputargs) < 1 {
		fmt.Printf("Not enough inputs.\n")
		return
	}
	myaddr, err = net.ResolveTCPAddr("tcp", inputargs[0])
	checkError(err)
	logger = govec.Initialize(inputargs[1], inputargs[1])
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
