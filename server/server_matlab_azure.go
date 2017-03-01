package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/4180122/distbayes/distmlMatlab"
	"github.com/arcaneiceman/GoVector/govec"
	"net"
	"os"
	"time"
)

var (
	cnum      int = 0
	maxnode   int = 0
	myaddr    *net.TCPAddr
	cnumhist  map[int]int
	client    map[string]int
	claddr    map[int]*net.TCPAddr
	tempmodel map[int]aggregate
	testqueue map[int]map[int]bool
	models    map[int]distmlMatlab.MatModel
	modelR    map[int]map[int]float64
	modelC    map[int]float64
	modelD    float64
	channel   chan message
	logger    *govec.GoLog
	l         *net.TCPListener
	gmodel    distmlMatlab.MatGlobalModel
	gempty    distmlMatlab.MatGlobalModel
)

type aggregate struct {
	cnum  int
	model distmlMatlab.MatModel
	r     map[int]float64
	d     float64
	c     float64
}

type message struct {
	Id       int
	NodeIp   string
	NodeName string
	Type     string
	Model    distmlMatlab.MatModel
	GModel   distmlMatlab.MatGlobalModel
}

type response struct {
	Resp  string
	Error string
}

func main() {
	//Initialize
	client = make(map[string]int)
	claddr = make(map[int]*net.TCPAddr)
	models = make(map[int]distmlMatlab.MatModel)
	modelR = make(map[int]map[int]float64)
	modelC = make(map[int]float64)
	modelD = 0.0
	gmodel = distmlMatlab.MatGlobalModel{nil}
	tempmodel = make(map[int]aggregate)
	testqueue = make(map[int]map[int]bool)
	cnumhist = make(map[int]int)
	channel = make(chan message)

	go updateGlobal(channel)

	//Parsing inputargs
	parseArgs()

	//Hacky solution to the Matlab problem (Mathworks, please fix this!)
	// see: https://www.mathworks.com/matlabcentral/answers/305877-what-is-the-primary-message-table-for-module-77
	// and  https://github.com/JuliaInterop/MATLAB.jl/issues/47
	distmlMatlab.Hack()

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
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	err := dec.Decode(&msg)
	checkError(err)
	switch msg.Type {
	case "commit_request":
		//node is sending a model, must forward to others for testing
		flag := checkQueue(client[msg.NodeName])
		fmt.Printf("<-- Received commit request from %v.\n", msg.NodeName)
		if flag {
			// accept commit from node and process outgoing test requests
			processTestRequest(msg, conn)
		} else {
			// deny commit request
			enc.Encode(response{"NO", "Restart"})
			fmt.Printf("--> Denied commit request from %v.\n", msg.NodeName)
			conn.Close()
		}
	case "global_request":
		//node is requesting the global model, will forward
		enc.Encode(response{"OK", ""})
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
			enc.Encode(response{"OK", "Test Processed"})
		} else {
			// if testqueue is already empty
			enc.Encode(response{"NO", "Duplicate Test"})
			fmt.Printf("--> Ignored test results from %v.\n", msg.NodeName)
		}
		conn.Close()
	case "join_request":
		enc.Encode(response{"OK", "Joined"})
		processJoin(msg)
		conn.Close()
	default:
		fmt.Printf("something weird happened!\n")
		enc.Encode(response{"NO", "Unknown Request"})
		conn.Close()
	}

}

// Global model update function
func updateGlobal(ch chan message) {
	// Function that aggregates the global model and commits when ready
	for {
		m := <-ch
		id := cnumhist[m.Id]
		tempAggregate := tempmodel[id]
		tempAggregate.d += m.Model.Size
		tempAggregate.r[client[m.NodeName]] = m.Model.Weight
		tempmodel[id] = tempAggregate
		if modelD < tempAggregate.d {
			modelD = tempAggregate.d
		}
		if float64(tempAggregate.d) > float64(modelD)*0.6 {
			models[id] = tempAggregate.model
			modelR[id] = tempAggregate.r
			modelC[id] = tempAggregate.c
			t := int32(time.Now().Unix())
			//logger.LogLocalEvent(fmt.Sprintf("%s - Committed model%v by %v at partial commit %v.", t.Format("15:04:05.0000"), id, client[m.NodeName], tempAggregate.d/modelD*100.0))
			logger.LogLocalEvent(fmt.Sprintf("%v %v %v", t, id, tempAggregate.d))
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.cnum)
		}
	}
}

// Generate global model from partial commits
func genGlobalModel() {
	modelstemp := models
	modelRtemp := modelR
	modelCtemp := modelC
	modelDtemp := modelD
	gmodel = distmlMatlab.CompactGlobal(modelstemp, modelRtemp, modelCtemp, modelDtemp)
}

// Function that generates test request following a commit request
func processTestRequest(m message, conn *net.TCPConn) {
	tempcnum := cnum
	cnum++
	cnumhist[tempcnum] = client[m.NodeName]
	enc := gob.NewEncoder(conn)
	//initialize new aggregate
	tempweight := make(map[int]float64)
	r := m.Model.Weight
	c := m.Model.Size
	tempweight[client[m.NodeName]] = r
	tempmodel[client[m.NodeName]] = aggregate{tempcnum, m.Model, tempweight, c, c}
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
	//sanitize the model for testing
	m.Model.Weight = 0.0
	m.Model.Size = 0.0
	enc.Encode(response{"OK", "Committed"})
	conn.Close()
	for name, id := range client {
		if id != client[m.NodeName] {
			sendTestRequest(name, id, tempcnum, m.Model)
		}
	}
}

// Function that sends test requests via TCP
func sendTestRequest(name string, id, tcnum int, tmodel distmlMatlab.MatModel) {
	//create test request
	msg := message{tcnum, myaddr.String(), "server", "test_request", tmodel, gempty}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", cnumhist[tcnum], name)
	err := tcpSend(claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	}
	// // Moved the incrementation of the testqueue to join process
	// if queue, ok := testqueue[id]; !ok {
	// 	queue := make(map[int]bool)
	// 	queue[cnumhist[tcnum]] = true
	// 	testqueue[id] = queue
	// 	//send the request
	// 	fmt.Printf("--> Sending test request from %v to %v.", cnumhist[tcnum], name)
	// 	err := tcpSend(claddr[id], msg)
	// 	if err != nil {
	// 		fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	// 	}
	// } else {
	// 	if !queue[cnumhist[cnum]] {
	// 		queue[cnumhist[tcnum]] = true
	// 		//send the request
	// 		fmt.Printf("--> Sending test request from %v to %v.", cnumhist[tcnum], name)
	// 		err := tcpSend(claddr[id], msg)
	// 		if err != nil {
	// 			fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	// 		}
	// 	}
	// }
}

// Function to forward global model
func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.NodeName)
	msg := message{m.Id, myaddr.String(), "server", "global_grant", m.Model, gmodel}
	tcpSend(claddr[client[m.NodeName]], msg)
}

// Function for sending messages to nodes via TCP
func tcpSend(addr *net.TCPAddr, msg message) error {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err == nil {
		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)
		err := enc.Encode(msg)
		checkError(err)
		var r response
		err = dec.Decode(&r)
		checkError(err)
		if r.Resp == "OK" {
			fmt.Printf(" [OK]\n")
		} else {
			fmt.Printf(" [%s]\n<-- Request was denied by node: %v.\nEnter command: ", r.Resp, r.Error)
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
		//os.Exit(1)
	}
}
