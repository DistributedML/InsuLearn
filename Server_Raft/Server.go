package main

import (
	"bytes"
	"fmt"
	"github.com/arcaneiceman/GoVector/govec"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"golang.org/x/net/context"
	"github.com/4180122/distbayes/bclass"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"time"
	"flag"
)

var (
	ln    	 	*net.TCPListener
	naddr  	 	map[int]string
	logger 		*govec.GoLog
	nID    		uint64
	cnum      	int = 0
	maxnode   	int = 0
	myaddr    	*net.TCPAddr
	cnumhist  	map[int]int
	client    	map[string]int
	claddr    	map[int]*net.TCPAddr
	tempmodel 	map[int]aggregate
	testqueue 	map[int]map[int]bool
	models    	map[int]bclass.Model
	modelC    	map[int]int
	modelD   	int
	channel   	chan message
	l         	*net.TCPListener
	gmodel    	bclass.GlobalModel
	gempty    	bclass.GlobalModel
	sID			string
)

const hb = 1

// clean this up maybe?
type Message struct {
	Msg raftpb.Message
}
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
type node struct {
	id     uint64
	ctx    context.Context
	pstore map[string]string
	store  *raft.MemoryStorage
	cfg    *raft.Config
	raft   raft.Node
	ticker <-chan time.Time
	done   <-chan struct{}
}

func newNode(id uint64, peers []raft.Peer) *node {
	store := raft.NewMemoryStorage()
	n := &node{
		id:    id,
		ctx:   context.TODO(),
		store: store,
		cfg: &raft.Config{
			ID:              id,
			ElectionTick:    5 * hb,
			HeartbeatTick:   hb,
			Storage:         store,
			MaxSizePerMsg:   math.MaxUint16,
			MaxInflightMsgs: 256,
		},
		pstore: make(map[string]string),
		ticker: time.Tick(time.Second),
		done:   make(chan struct{}),
	}

	n.raft = raft.StartNode(n.cfg, peers)
	return n
}

func (n *node) run() {
	for {
		select {
		case <-n.ticker:
			n.raft.Tick()
		case rd := <-n.raft.Ready():
			n.saveToStorage(rd.HardState, rd.Entries, rd.Snapshot)
			n.send(rd.Messages)
			if !raft.IsEmptySnap(rd.Snapshot) {
				n.processSnapshot(rd.Snapshot)
			}
			for _, entry := range rd.CommittedEntries {
				n.process(entry)
				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					cc.Unmarshal(entry.Data)
					n.raft.ApplyConfChange(cc)
				}
			}
			n.raft.Advance()
		case <-n.done:
			return
		}
	}
}

func (n *node) saveToStorage(hardState raftpb.HardState, entries []raftpb.Entry, snapshot raftpb.Snapshot) {
	n.store.Append(entries)

	if !raft.IsEmptyHardState(hardState) {
		n.store.SetHardState(hardState)
	}

	if !raft.IsEmptySnap(snapshot) {
		n.store.ApplySnapshot(snapshot)
	}
}

func (n *node) send(messages []raftpb.Message) {
	for _, m := range messages {
		//log.Println(raft.DescribeMessage(m, nil))
		msg := Message{m}
		outBuf := logger.PrepareSend("Sending message to other node", msg)
		conn, err := net.Dial("tcp", naddr[int(m.To)])
		if err == nil {
			conn.Write(outBuf)
		}
		// send message to other node
	}
}

func (n *node) processSnapshot(snapshot raftpb.Snapshot) {
	panic(fmt.Sprintf("Applying snapshot on node %v is not implemented", n.id))
}

func (n *node) process(entry raftpb.Entry) {
	log.Printf("node %v: processing entry: %v\n", n.id, entry)
	if entry.Type == raftpb.EntryNormal && entry.Data != nil {
		parts := bytes.SplitN(entry.Data, []byte(":"), 2)
		n.pstore[string(parts[0])] = string(parts[1])
	}
}

func (n *node) receive(conn *net.TCPConn) {
	// Echo all incoming data.
	var imsg Message
	buf := make([]byte, 2048)
	conn.Read(buf)
	logger.UnpackReceive("Received Message From Client", buf, &imsg)
	conn.Close()
	//leadern := n.raft.Status().Lead
	//fmt.Println("leader: ", leadern)
	n.raft.Step(n.ctx, imsg.Msg)
}

var (
	nodes = make(map[int]*node)
)

func main() {
	//Parsing inputargs
	parseArgs()
	// C1:=make(chan 	*net.TCPListener)
	// C2:=make(chan 	*net.TCPListener)
	nID, _ = strconv.ParseUint(sID, 10, 64)
	InID := int(nID)

	naddr = make(map[int]string)
	naddr[1] = "127.0.0.1:6001"
	naddr[2] = "127.0.0.1:6002"
	naddr[3] = "127.0.0.1:6003"
	naddr[4] = "127.0.0.1:6004"
	naddr[5] = "127.0.0.1:6005"

	nodeaddr, _ := net.ResolveTCPAddr("tcp", naddr[InID])
	ln, _ = net.ListenTCP("tcp", nodeaddr)
	l, err := net.ListenTCP("tcp", myaddr)
	checkError(err)
	fmt.Println(l)
	// C1<-ln
	// C2<-l
	// start a small cluster
	nodes[InID] = newNode(nID, []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
	nodes[InID].raft.Campaign(nodes[InID].ctx)
	go nodes[InID].run()
	go printLeader(nodes[InID], InID)

	// Wait for proposed entry to be commited in cluster.
	// Apperently when should add an uniq id to the message and wait until it is
	// commited in the node.
	//fmt.Printf("** Sleeping to visualize heartbeat between nodes **\n")
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
	fmt.Println("my addre",myaddr)

	//Initialize TCP Connection and listener

	fmt.Printf("Server initialized.\n")
	go func() {
			connC, err := l.AcceptTCP()
			checkError(err)
			connHandler(connC)		
		}()

	for {

			conn, _ := ln.AcceptTCP()
			go nodes[InID].receive(conn)

	}

}

func printLeader(n *node, id int) {
	for {
		time.Sleep(2 * time.Second)
		leadern := n.raft.Status().Lead
		fmt.Println("leader: ", leadern)
		if int(leadern) == id {
			n.raft.Propose(n.ctx, []byte("foo:bar"))
		} else {
			for k, v := range n.pstore {
				fmt.Printf("%v = %v \n", k, v)
			}
		}
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
		flag := checkQueue(client[msg.Name])
		fmt.Printf("<-- Received commit request from %v.\n", msg.Name)
		if flag {
			conn.Write([]byte("OK"))
			conn.Close()
			processTestRequest(msg)
		} else {
			//denied
			conn.Write([]byte("Pending tests are not complete."))
			fmt.Printf("--> Denied commit request from %v.\n", msg.Name)
			conn.Close()
		}
	case "global_request":
		//node is requesting the global model, will forward
		conn.Write([]byte("OK"))
		fmt.Printf("<-- Received global model request from %v.\n", msg.Name)
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		//node is submitting test results, will update its queue
		conn.Write([]byte("OK"))
		fmt.Printf("<-- Received completed test results from %v.\n", msg.Name)
		testqueue[client[msg.Name]][cnumhist[msg.Id]] = false
		conn.Close()
		//update the pending commit and merge if complete
		channel <- msg
	default:
		conn.Write([]byte("OK"))
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
		if float64(tempAggregate.d) > float64(modelD)*0.6 {
			models[id] = tempAggregate.model
			modelC[id] = tempAggregate.c
			gmodel = bclass.GlobalModel{models, modelC, modelD}
			logger.LogLocalEvent("commit_complete")
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.cnum)
		}
	}
}

func processTestRequest(m message) {
	cnum++
	//initialize new aggregate
	tempmodel[client[m.Name]] = aggregate{cnum, m.Model, m.C, m.D}
	cnumhist[cnum] = client[m.Name]
	for name, id := range client {
		if id != client[m.Name] {
			sendTestRequest(name, id, cnum, m.Model)
		}
	}
}

func sendTestRequest(name string, id, tcnum int, tmodel bclass.Model) {
	//create test request
	msg := message{tcnum, "server", "test_request", 0, 0, tmodel, gempty}
	//increment tests in queue for that node
	if queue, ok := testqueue[id]; !ok {
		queue := make(map[int]bool)
		queue[cnumhist[tcnum]] = true
		testqueue[id] = queue
	} else {
		queue[cnumhist[tcnum]] = true
	}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", cnumhist[tcnum], name)
	err := tcpSend(claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	}
}

func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.Name)
	msg := message{m.Id, "server", "global_grant", 0, 0, m.Model, gmodel}
	tcpSend(claddr[client[m.Name]], msg)
}

func tcpSend(addr *net.TCPAddr, msg message) error {
	p := make([]byte, 1024)
	conn, err := net.DialTCP("tcp", nil, addr)
	//checkError(err)
	if err == nil {
		outbuf := logger.PrepareSend(msg.Type, msg)
		_, err = conn.Write(outbuf)
		//checkError(err)
		n, _ := conn.Read(p)
		if string(p[:n]) != "OK" {
			fmt.Printf(" [NO!]\n<-- Request was denied by node: %v.\nEnter command: ", string(p[:n]))
		} else {
			fmt.Printf(" [OK]\n")
		}
	}
	return err
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
	//process depending on if it is a new node or a returning one
	if _, ok := client[m.Name]; !ok {
		//adding a node that has never been added before
		id := maxnode
		maxnode++
		client[m.Name] = id
		claddr[id], _ = net.ResolveTCPAddr("tcp", m.Type)
		fmt.Printf("--- Added %v as node%v.\n", m.Name, id)
		for _, v := range tempmodel {
			sendTestRequest(m.Name, id, v.cnum, v.model)
		}
	} else {
		//node is rejoining, update address and resend the unfinished test requests
		id := client[m.Name]
		claddr[id], _ = net.ResolveTCPAddr("tcp", m.Type)
		fmt.Printf("--- %v at node%v is back online.\n", m.Name, id)
		for k, v := range testqueue[id] {
			if v {
				aggregate := tempmodel[k]
				sendTestRequest(m.Name, id, aggregate.cnum, aggregate.model)
			}
		}
	}
}

func parseArgs() {
	flag.Parse()
	inputargs := flag.Args()
	var err error
	if len(inputargs) < 2 {
		fmt.Printf("Not enough inputs.\n")
		return
	}
	myaddr, err = net.ResolveTCPAddr("tcp", inputargs[0])
	checkError(err)
	sID=inputargs[1]
	logger = govec.Initialize(inputargs[2], inputargs[2])
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
