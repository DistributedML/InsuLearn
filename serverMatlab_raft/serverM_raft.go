package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/4180122/distbayes/distmlMatlab"
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

var (
	naddr   map[int]string
	logger  *govec.GoLog
	nID     int
	myaddr  *net.TCPAddr
	channel chan message
	models  map[int]distmlMatlab.MatModel
	modelR  map[int]map[int]float64
	modelC  map[int]float64
	modelD  float64
	gempty  distmlMatlab.MatGlobalModel
	gmodel  distmlMatlab.MatGlobalModel
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
	Model distmlMatlab.MatModel
	R     map[int]float64
	C     float64
	D     float64
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

// Function to initialize a new Raft node
func newNode(id uint64, peers []raft.Peer) *node {
	store := raft.NewMemoryStorage()
	n := &node{
		id:    id,
		ctx:   context.TODO(),
		store: store,
		cfg: &raft.Config{
			ID:              id,
			ElectionTick:    10 * hb,
			HeartbeatTick:   1 * hb,
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
			fmt.Println(n.testqueue[id])
			fmt.Printf("--- Added %v as node%v.\n", msg.NodeName, id)
		case "rejoin_request":
			id := n.client[msg.NodeName]
			n.claddr[id], _ = net.ResolveTCPAddr("tcp", msg.NodeIp)
			fmt.Printf("###########--- %v at node%v is back online.\n", msg.NodeName, id)
		case "commit_request":
			tempcnum := n.cnum
			n.cnum++
			n.cnumhist[tempcnum] = n.client[msg.NodeName]
			//initialize new aggregate
			tempweight := make(map[int]float64)
			r := msg.Model.Weight
			c := msg.Model.Size
			tempweight[n.client[msg.NodeName]] = r
			n.tempmodel[n.client[msg.NodeName]] = aggregate{tempcnum, msg.Model, tempweight, c, c}
			for _, id := range n.client {
				if id != n.client[msg.NodeName] {
					queue := make(map[int]bool)
					queue[n.cnumhist[tempcnum]] = true
					n.testqueue[id] = queue
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

func main() {
	//Parsing inputargs
	parseArgs()
	nodeRaftIP := strings.Split(naddr[nID], ":")
	nodeRaftPort := ":" + nodeRaftIP[1]
	fmt.Println(nodeRaftPort)
	raftaddr, _ := net.ResolveTCPAddr("tcp", naddr[nID])
	sl, err := net.ListenTCP("tcp", raftaddr)
	checkError(err)
	cl, err := net.ListenTCP("tcp", myaddr)
	checkError(err)

	// Wait for proposed entry to be commited in cluster.
	// Apperently when should add an uniq id to the message and wait until it is
	// commited in the node.
	models = make(map[int]distmlMatlab.MatModel)
	modelR = make(map[int]map[int]float64)
	modelC = make(map[int]float64)
	modelD = 0.0
	gmodel = distmlMatlab.MatGlobalModel{nil}
	channel = make(chan message)
	// start a cluster R = 7
	//mynode = newNode(uint64(nID), []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}, {ID: 6}, {ID: 7}})
	// start a cluster R = 5
	mynode = newNode(uint64(nID), []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})

	go mynode.run()

	if nID == 1 {
		time.Sleep(time.Duration(5 * time.Second))
		mynode.raft.Campaign(mynode.ctx)
	}

	//Hacky solution to the Matlab problem (Mathworks, please fix this!)
	// see: https://www.mathworks.com/matlabcentral/answers/305877-what-is-the-primary-message-table-for-module-77
	// and  https://github.com/JuliaInterop/MATLAB.jl/issues/47
	distmlMatlab.Hack()
	//Initialize TCP Connection and listenerfmt.Printf("Server initialized.\n")

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

// Function for handling client requests and replicating on Raft if necessary
func connHandler(conn *net.TCPConn) {
	var msg message
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	err := dec.Decode(&msg)
	checkError(err)
	switch msg.Type {
	case "commit_request":
		// node is sending a model, checking to see if testing is complete
		flag := checkQueue(mynode.client[msg.NodeName])
		if flag {
			// accept commit from node and process outgoing test requests
			fmt.Printf("<-- Received commit request from %v.\n", msg.NodeName)
			enc.Encode(response{"OK", "Committed"})
			conn.Close()
			processTestRequest(msg, conn)
		} else {
			// deny commit request
			enc.Encode(response{"NO", "Restart"})
			fmt.Printf("--> Denied commit request from %v.\n", msg.NodeName)
			conn.Close()
		}
	case "global_request":
		// node is requesting the global model -> generate and forward
		enc.Encode(response{"OK", ""})
		fmt.Printf("<-- Received global model request from %v.\n", msg.NodeName)
		genGlobalModel()
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		// node is submitting test results, update testqueue on all replicas
		fmt.Printf("<-- Received completed test results from %v.\n", msg.NodeName)
		repstate := state{0, msg}
		flag := replicate(repstate)
		if flag {
			enc.Encode(response{"OK", "Test Processed"})
		} else {
			// if testqueue could not be replicated
			enc.Encode(response{"NO", "Try Again"})
			fmt.Printf("--> Could not process test from %v.\n", msg.NodeName)
		}
		conn.Close()
	case "join_request":
		// node is requesting to join or rejoin
		fmt.Printf("<-- Received join request from %v.\n", msg.NodeName)
		flag := processJoin(msg, conn)
		if flag {
			enc.Encode(response{"OK", "Joined"})
		} else {
			fmt.Printf("*** Could not process join for node %v.\n", msg.NodeName)
			enc.Encode(response{"NO", "Failed Join"})
		}
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
		id := mynode.cnumhist[m.Id]
		tempAggregate := mynode.tempmodel[id]
		tempAggregate.D += m.Model.Size
		tempAggregate.R[mynode.client[m.NodeName]] += m.Model.Weight
		mynode.tempmodel[id] = tempAggregate
		if modelD < tempAggregate.D {
			modelD = tempAggregate.D
		}

		// partial commit
		if float64(tempAggregate.D) > float64(modelD)*0.6 {
			models[id] = tempAggregate.Model
			modelR[id] = tempAggregate.R
			modelC[id] = tempAggregate.C
			t := time.Now()
			logger.LogLocalEvent(fmt.Sprintf("%s - Committed model%v by %v at partial commit %v.", t.Format("15:04:05.0000"), id, mynode.client[m.NodeName], tempAggregate.D/modelD*100.0))
			//logger.LogLocalEvent(fmt.Sprintf("%v %v %v", t, id, tempAggregate.d))
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.Cnum)
		}
	}
}

// Replicate function that coordinates Raft nodes and blocks until all nodes have been synced
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
	repstate := state{0, m}
	flag := replicate(repstate)
	enc := gob.NewEncoder(conn)
	if flag {
		enc.Encode(response{"OK", ""})
		conn.Close()
		//sanitize the model for testing
		m.Model.Weight = 0.0
		m.Model.Size = 0.0
		tempcnum := 0
		//get the latest cnum
		for k, v := range mynode.cnumhist {
			if v == mynode.client[m.NodeName] {
				tempcnum = k
			}
		}
		for name, id := range mynode.client {
			if id != mynode.client[m.NodeName] {
				sendTestRequest(name, id, tempcnum, m.Model)
			}
		}
	} else {
		enc.Encode(response{"NO", "Try Again!"})
		conn.Close()
		fmt.Printf("--> Failed to commit request from %v.\n", m.NodeName)
	}
}

// Function that sends test requests via TCP
func sendTestRequest(name string, id, tcnum int, tmodel distmlMatlab.MatModel) {
	//create test request
	msg := message{tcnum, "server", "server", "test_request", tmodel, gempty}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", mynode.cnumhist[tcnum], name)
	err := tcpSend(mynode.claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	}
}

// Function to forward global model
func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.NodeName)
	msg := message{m.Id, myaddr.String(), "server", "global_grant", m.Model, gmodel}
	tcpSend(mynode.claddr[mynode.client[m.NodeName]], msg)
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
	for _, v := range mynode.testqueue[id] {
		if flag && v {
			flag = false
		}
	}
	return flag
}

// Function that processes join requests and forwards response to Raft nodes
func processJoin(m message, conn *net.TCPConn) bool {
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
					fmt.Println("########## It's working")
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
	logger = govec.Initialize(inputargs[3], inputargs[3])
	getNodeAddr(inputargs[1])
	temp, _ := strconv.ParseInt(inputargs[2], 10, 64)
	nID = int(temp)
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
