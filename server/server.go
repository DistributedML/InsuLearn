package main

import (
	"flag"
	"fmt"
	"github.com/4180122/distbayes/bclass"
	"github.com/arcaneiceman/GoVector/govec"
	"net"
	"os"
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
	Id     int
	Name   string
	Type   string
	C      int
	D      int
	Model  bclass.Model
	GModel bclass.GlobalModel
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
	fmt.Printf("server initialized.\n")

	for {
		conn, err := l.AcceptTCP()
		checkError(err)
		go connHandler(conn)
	}

}

func connHandler(conn *net.TCPConn) {
	var msg message
	p := make([]byte, 1048576)
	conn.Read(p)
	logger.UnpackReceive("Received message", p, &msg)
	switch msg.Type {
	case "commit_request":
		//node is sending a model, must forward to others for testing
		fmt.Printf("Received commit request from %v.\n", msg.Name)
		flag := checkQueue(client[msg.Name])
		if flag {
			conn.Close()
			sendTestRequest(msg)
		} else {
			//denied
			conn.Close()
		}
	case "global_request":
		//node is requesting the global model, will forward
		fmt.Printf("Received global model request from %v.\n", msg.Name)
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		//node is submitting test results, will update its queue
		fmt.Printf("Received completed test results from %v.\n", msg.Name)
		testqueue[client[msg.Name]][cnumhist[msg.Id]] = false
		conn.Close()
		//update the pending commit and merge if complete
		channel <- msg
	default:
		processJoin(msg)
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
		if float64(tempAggregate.d) > float64(modelD)*0.8 {
			models[id] = tempAggregate.model
			modelC[id] = tempAggregate.c
			gmodel = bclass.GlobalModel{models, modelC, modelD}
			fmt.Printf("Committed model%v for commit number: %v.\n", id, tempAggregate.cnum)
		}
	}
}

func sendTestRequest(m message) {
	cnum++
	//initialize new aggregate
	tempmodel[client[m.Name]] = aggregate{cnum, m.Model, m.C, m.D}
	cnumhist[cnum] = client[m.Name]
	for name, id := range client {
		if id != client[m.Name] {
			//forward test request
			msg := message{cnum, "server", "test_request", 0, 0, m.Model, gempty}
			tcpSend(claddr[id], msg)
			//increment tests in queue for that node
			if queue, ok := testqueue[id]; !ok {
				queue := make(map[int]bool)
				queue[cnumhist[cnum]] = true
				testqueue[id] = queue
			} else {
				queue[cnumhist[cnum]] = true
			}
			fmt.Printf("Sent test request to %v.\n", name)
		}
	}
}

func sendGlobal(m message) {
	msg := message{m.Id, "server", "global_grant", 0, 0, m.Model, gmodel}
	tcpSend(claddr[client[m.Name]], msg)
}

func tcpSend(addr *net.TCPAddr, msg message) {
	conn, err := net.DialTCP("tcp", nil, addr)
	checkError(err)
	outbuf := logger.PrepareSend(msg.Type, msg)
	_, err = conn.Write(outbuf)
	checkError(err)
}

func checkQueue(id int) bool {
	flag := true
	for _, v := range testqueue[id] {
		if flag && v {
			flag = false
		}
	}
	return flag
}

func processJoin(m message) {
	id := maxnode
	maxnode++
	client[m.Name] = id
	claddr[id], _ = net.ResolveTCPAddr("tcp", m.Type)
	fmt.Printf("Added %v as node%v.\n", m.Name, id)
}

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

//func getNodeAddr(slavefile string) {
//	dat, err := ioutil.ReadFile(slavefile)
//	checkError(err)
//	nodestr := strings.Split(string(dat), "\n")
//	for i := 0; i < len(nodestr)-1; i++ {
//		nodeaddr := strings.Split(nodestr[i], " ")
//		client[nodeaddr[0]] = i
//		claddr[i], _ = net.ResolveTCPAddr("tcp", nodeaddr[1])
//		//testqueue[i] = 0
//	}
//}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
