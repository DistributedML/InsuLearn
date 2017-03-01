package main

import (
	"../bclass"
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/govec"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"golang.org/x/net/context"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const hb = 5
const BUFFSIZE = 1048576

var (
	naddr   map[int]string
	logger  *govec.GoLog
	nID     int
	myaddr  *net.TCPAddr
	channel chan message
	models  map[int]bclass.Model
	modelC  map[int]int
	modelD  int
	gmodel  bclass.GlobalModel
	gempty  bclass.GlobalModel
	mynode  *node
)

type node struct {
	id        uint64
	ctx       context.Context
	store     *raft.MemoryStorage
	cfg       *raft.Config
	raft      raft.Node
	propID    map[int]bool
	maxnode   int
	cnum      int
	cnumhist  map[int]int
	client    map[string]int
	tempmodel map[int]aggregate
	testqueue map[int]map[int]bool
	claddr    map[int]*net.TCPAddr
	ticker    <-chan time.Time
	done      <-chan struct{}
}

type state struct {
	PropID int
	Msg    message
}

type aggregate struct {
	Cnum  int
	Model bclass.Model
	C     int
	D     int
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

// Function to initialize a new Raft node
func newNode(id uint64, peers []raft.Peer) *node {
	store := raft.NewMemoryStorage()
	n := &node{
		id:    id,
		ctx:   context.TODO(),
		store: store,
		cfg: &raft.Config{
			ID:              id,
			ElectionTick:    20 * hb,
			HeartbeatTick:   4 * hb,
			Storage:         store,
			MaxSizePerMsg:   math.MaxUint16,
			MaxInflightMsgs: 1024,
		},
		propID:    make(map[int]bool),
		maxnode:   0,
		cnum:      0,
		client:    make(map[string]int),
		claddr:    make(map[int]*net.TCPAddr),
		tempmodel: make(map[int]aggregate),
		testqueue: make(map[int]map[int]bool),
		cnumhist:  make(map[int]int),
		ticker:    time.Tick(time.Second / 10),
		done:      make(chan struct{}),
	}

	n.raft = raft.StartNode(n.cfg, peers)
	return n
}

// Raft state machine
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

// Raft operations for saving snapshots
func (n *node) saveToStorage(hardState raftpb.HardState, entries []raftpb.Entry, snapshot raftpb.Snapshot) {
	n.store.Append(entries)
	if !raft.IsEmptyHardState(hardState) {
		n.store.SetHardState(hardState)
	}
	if !raft.IsEmptySnap(snapshot) {
		n.store.ApplySnapshot(snapshot)
	}
}

// Raft send function using gob encoder [between Raft nodes]
func (n *node) send(messages []raftpb.Message) {
	for _, m := range messages {
		//outBuf := logger.PrepareSend("Sending message to other node", m)
		conn, err := net.Dial("tcp", naddr[int(m.To)])
		if err == nil {
			enc := gob.NewEncoder(conn)
			enc.Encode(m)
		} else {
			fmt.Printf("*** Could not send message to Raft node: %v.\n", int(m.To))
		}
	}
}

// Raft receive function using gob encoder [between Raft nodes]
func (n *node) receive(conn *net.TCPConn) {
	// Echo all incoming data.
	var imsg raftpb.Message
	dec := gob.NewDecoder(conn)
	err := dec.Decode(&imsg)
	checkError(err)
	conn.Close()
	n.raft.Step(n.ctx, imsg)
}

// Raft function for loading snapshots [not implemented]
func (n *node) processSnapshot(snapshot raftpb.Snapshot) {
	panic(fmt.Sprintf("Applying snapshot on node %v is not implemented", n.id))
}

// Raft process functions, idempotent modifications to the key-value stores and join/commit history updates
func (n *node) process(entry raftpb.Entry) {
	if entry.Type == raftpb.EntryNormal && entry.Data != nil {
		var repstate state
		var buf bytes.Buffer
		dec := gob.NewDecoder(&buf)
		buf.Write(entry.Data)
		dec.Decode(&repstate)
		msg := repstate.Msg
		switch msg.Type {
		case "join_request":
			id := n.maxnode
			n.maxnode++
			n.client[msg.NodeName] = id
			n.claddr[id], _ = net.ResolveTCPAddr("tcp", msg.NodeIp)
			queue := make(map[int]bool)
			for k, _ := range n.tempmodel {
				queue[k] = true
			}
			n.testqueue[id] = queue
			fmt.Printf("--- Added %v as node%v.\n", msg.NodeName, id)
		case "rejoin_request":
			id := n.client[msg.NodeName]
			n.claddr[id], _ = net.ResolveTCPAddr("tcp", msg.NodeIp)
		case "commit_request":
			tempcnum := n.cnum
			n.cnum++
			n.cnumhist[tempcnum] = n.client[msg.NodeName]
			//initialize new aggregate
			n.tempmodel[n.client[msg.NodeName]] = aggregate{tempcnum, msg.Model, msg.C, msg.D}
			for _, id := range n.client {
				if id != n.client[msg.NodeName] {
					if queue, ok := n.testqueue[id]; !ok {
						queue := make(map[int]bool)
						queue[n.cnumhist[tempcnum]] = true
						n.testqueue[id] = queue
					} else {
						queue[n.cnumhist[tempcnum]] = true
					}
				}
			}
			fmt.Printf("--- Processed commit %v for node %v.\n", tempcnum, msg.NodeName)
		case "test_complete":
			n.testqueue[n.client[msg.NodeName]][n.cnumhist[msg.Id]] = false
			channel <- msg
		default:
			// Do nothing
		}
		n.propID[repstate.PropID] = true
	}
}

// Replicate function that coordinates Raft nodes and blocks until all nodes have been synced
//   The map used to check commit grows as more replication proccesses are done and will start to
//   cause problems when the number of commits increase past a certain amount. It would be wise to
//   replace this with a better alternative.
func replicate(m state) bool {
	flag := false

	r := rand.Intn(999999999999)
	m.PropID = r
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	err = mynode.raft.Propose(mynode.ctx, buf.Bytes())

	if err == nil {
		//block and check the status of the proposal
		for !flag {
			time.Sleep(time.Duration(1 * time.Second))
			if mynode.propID[r] {
				flag = true
			}
		}
	}

	return flag
}

func main() {
	//Parsing inputargs
	parseArgs()

	raftaddr, _ := net.ResolveTCPAddr("tcp", naddr[nID])
	sl, err := net.ListenTCP("tcp", raftaddr)
	checkError(err)
	cl, err := net.ListenTCP("tcp", myaddr)
	checkError(err)

	// Wait for proposed entry to be commited in cluster.
	// Apperently when should add an uniq id to the message and wait until it is
	// commited in the node.
	models = make(map[int]bclass.Model)
	modelC = make(map[int]int)
	modelD = 0
	gmodel = bclass.GlobalModel{models, modelC, modelD}
	channel = make(chan message)

	// start a small cluster
	mynode = newNode(uint64(nID), []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})

	go mynode.run()

	if nID == 1 {
		time.Sleep(time.Duration(5 * time.Second))
		mynode.raft.Campaign(mynode.ctx)
	}

	go updateGlobal(channel)

	go clientListener(cl)

	go printLeader()

	for {
		conn, err := sl.AcceptTCP()
		if err == nil {
			go mynode.receive(conn)
		} else {
			fmt.Printf("*** Could not accept connection from Raft node: %s.", err)
		}
	}

}

// Function to periodically print Raft Leader
func printLeader() {
	for {
		time.Sleep(time.Duration(5 * time.Second))
		sts := mynode.raft.Status()
		fmt.Printf("--- Current leader is %v.\n", sts.Lead)
	}
}

// Client listener function
func clientListener(listen *net.TCPListener) {
	for {
		connC, err := listen.AcceptTCP()
		if err == nil {
			go connHandler(connC)
		} else {
			fmt.Printf("*** Could not accept connection from Client node: %s.", err)
		}
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
		flag := checkQueue(mynode.client[msg.NodeName])
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
		//node is submitting test results, will update its queue
		fmt.Printf("<-- Received completed test results from %v.\n", msg.NodeName)
		if mynode.testqueue[mynode.client[msg.NodeName]][mynode.cnumhist[msg.Id]] {
			repstate := state{0, msg}
			flag := replicate(repstate)
			if flag {
				conn.Write([]byte("OK"))
			} else {
				// if testqueue could not be replicated
				conn.Write([]byte("Try again"))
				fmt.Printf("--> Could not process test from %v.\n", msg.NodeName)
			}
		} else {
			// if testqueue is already empty
			conn.Write([]byte("Duplicate Test"))
			fmt.Printf("--> Ignored test results from %v.\n", msg.NodeName)
		}

		conn.Close()
	case "join_request":
		// node is requesting to join or rejoin
		fmt.Printf("<-- Received join request from %v.\n", msg.NodeName)
		flag := processJoin(msg)
		if flag {
			conn.Write([]byte("Joined"))
		} else {
			fmt.Printf("*** Could not process join for node %v.\n", msg.NodeName)
			conn.Write([]byte("Failed Join"))
		}
		conn.Close()
	default:
		conn.Write([]byte("Unknown Request"))
		fmt.Printf("something weird happened!\n")
		conn.Close()
	}
}

// Global model update function
func updateGlobal(ch chan message) {
	// Function that aggregates the global model and commits when ready
	for {
		m := <-ch
		id := mynode.cnumhist[m.Id]
		tempAggregate := mynode.tempmodel[id]
		tempAggregate.C += m.C
		tempAggregate.D += m.D
		mynode.tempmodel[id] = tempAggregate
		if modelD < tempAggregate.D {
			modelD = tempAggregate.D
		}

		if float64(tempAggregate.D) > float64(modelD)*0.6 {
			models[id] = tempAggregate.Model
			modelC[id] = tempAggregate.C
			t := time.Now()
			logger.LogLocalEvent(fmt.Sprintf("%s - Committed model%v by %v at partial commit %v.", t.Format("15:04:05.0000"), id, mynode.client[m.NodeName], tempAggregate.D/modelD*100.0))
			//logger.LogLocalEvent("commit_complete")
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.Cnum)
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
	repstate := state{0, m}
	flag := replicate(repstate)
	if flag {
		//sanitize the model for testing
		tempcnum := 0
		//get the latest cnum, necessary as cnum is updated in raft
		for k, v := range mynode.cnumhist {
			if v == mynode.client[m.NodeName] {
				tempcnum = k
			}
		}
		conn.Write([]byte("OK"))
		conn.Close()
		for name, id := range mynode.client {
			if id != mynode.client[m.NodeName] {
				sendTestRequest(name, id, tempcnum, m.Model)
			}
		}
	} else {
		conn.Write([]byte("Try again"))
		conn.Close()
		fmt.Printf("--> Failed to commit request from %v.\n", m.NodeName)
	}
}

// Function that sends test requests via TCP
func sendTestRequest(name string, id, tcnum int, tmodel bclass.Model) {
	//create test request (sanitized)
	msg := message{tcnum, "server", "server", "test_request", 0, 0, tmodel, gempty}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", mynode.cnumhist[tcnum], name)
	err := tcpSend(mynode.claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO]\n*** Could not send test request to %v.\n", name)
	}
}

// Function to forward global model
func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.NodeName)
	msg := message{m.Id, "server", "server", "global_grant", 0, 0, m.Model, gmodel}
	tcpSend(mynode.claddr[mynode.client[m.NodeName]], msg)
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
	for _, v := range mynode.testqueue[id] {
		if flag && v {
			flag = false
		}
	}
	return flag
}

// Function that processes join requests and forwards response to Raft nodes
func processJoin(m message) bool {
	//process depending on if it is a new node or a returning one
	flag := false

	if id, ok := mynode.client[m.NodeName]; !ok {
		//adding a node that has never been added before
		repstate := state{0, m}
		flag = replicate(repstate)
		if flag {
			for _, v := range mynode.tempmodel {
				sendTestRequest(m.NodeName, mynode.client[m.NodeName], v.Cnum, v.Model)
			}
		}
	} else {
		//node is rejoining, update address and resend the unfinished test requests
		m.Type = "rejoin_request"
		repstate := state{0, m}
		flag = replicate(repstate)
		if flag {
			//time.Sleep(time.Duration(2 * time.Second))
			for k, v := range mynode.testqueue[id] {
				if v {
					aggregate := mynode.tempmodel[k]
					sendTestRequest(m.NodeName, id, aggregate.Cnum, aggregate.Model)
				}
			}
		}
	}
	return flag
}

// Input parser
func parseArgs() {
	naddr = make(map[int]string)
	flag.Parse()
	inputargs := flag.Args()
	var err error
	if len(inputargs) < 2 {
		fmt.Printf("Not enough inputs.\n")
		return
	}
	myaddr, err = net.ResolveTCPAddr("tcp", inputargs[0])
	checkError(err)
	getNodeAddr(inputargs[1])
	temp, _ := strconv.ParseInt(inputargs[2], 10, 64)
	nID = int(temp)
	logger = govec.Initialize(inputargs[3], inputargs[3])
}

// Function for reading server addresses from file
func getNodeAddr(slavefile string) {
	dat, err := ioutil.ReadFile(slavefile)
	checkError(err)
	nodestr := strings.Split(string(dat), " ")
	for i := 0; i < len(nodestr)-1; i++ {
		naddr[i+1] = nodestr[i]
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		//os.Exit(1)
	}
}
