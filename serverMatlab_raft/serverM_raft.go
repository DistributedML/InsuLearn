package main

import (
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
const BUFFSIZE = 200000
const hb = 1

var (
	naddr   map[int]string
	logger  *govec.GoLog
	nID     int
	myaddr  *net.TCPAddr
	models  map[int]distmlMatlab.MatModel
	modelC  map[int]float64
	modelD  float64
	channel chan message
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
	propID    int
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
	PropID    int
	Maxnode   int
	Cnum      int
	Cnumhist  map[int]int
	Client    map[string]int
	Tempmodel map[int]aggregate
	Testqueue map[int]map[int]bool
	Claddr    map[int]string
	Msg       message
}
type aggregate struct {
	cnum  int
	model distmlMatlab.MatModel
	r     map[int]float64
	c     float64
	d     float64
}
type message struct {
	Id     int
	Ip     string
	Name   string
	Type   string
	Model  distmlMatlab.MatModel
	GModel distmlMatlab.MatGlobalModel
}
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
		propID:    0,
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
		outBuf := logger.PrepareSend("Sending message to other node", m)
		conn, err := net.Dial("tcp", naddr[int(m.To)])
		if err == nil {
			conn.Write(outBuf)
		}
	}
}

func (n *node) processSnapshot(snapshot raftpb.Snapshot) {
	panic(fmt.Sprintf("Applying snapshot on node %v is not implemented", n.id))
}

//process idempotent modifications to the key-value stores
func (n *node) process(entry raftpb.Entry) {
	if entry.Type == raftpb.EntryNormal && entry.Data != nil {
		var repstate state
		logger.UnpackReceive("got propose", entry.Data, &repstate)
		mynode.propID = repstate.PropID
		if repstate.Cnum > mynode.cnum {
			mynode.cnum = repstate.Cnum
		}
		if repstate.Maxnode > mynode.maxnode {
			mynode.maxnode = repstate.Maxnode
		}
		for k, v := range repstate.Cnumhist {
			mynode.cnumhist[k] = v
			//fmt.Println("Committed c_hist:", mynode.cnumhist)
		}
		for k, v := range repstate.Client {
			mynode.client[k] = v
			fmt.Println("Committed client:", mynode.client)
		}
		for k, v := range repstate.Tempmodel {
			mynode.tempmodel[k] = v
			//fmt.Println("Committed tempmodel:", mynode.tempmodel)
		}
		for k, v := range repstate.Claddr {
			mynode.claddr[k], _ = net.ResolveTCPAddr("tcp", v)
			//fmt.Println("Committed client address:", mynode.claddr)
		}
		for k, v := range repstate.Testqueue {
			for k2, v2 := range v {
				if queue, ok := mynode.testqueue[k]; !ok {
					queue := make(map[int]bool)
					queue[k2] = v2
					mynode.testqueue[k] = queue
				} else {
					queue[k2] = v2
				}
			}
			//fmt.Println("Committed testqueue:", mynode.testqueue)
		}
		if repstate.Msg.Type != "" {
			//fmt.Println("message :", repstate.Msg)
			channel <- repstate.Msg
			//time.Sleep(time.Second * 2)
			//fmt.Println("Committed gmodel:", gmodel)
		}
	}
}

func (n *node) receive(conn *net.TCPConn) {
	// Echo all incoming data.
	var imsg raftpb.Message
	buf := make([]byte, 1048576)
	conn.Read(buf)
	logger.UnpackReceive("Received Message From Client", buf, &imsg)
	conn.Close()
	n.raft.Step(n.ctx, imsg)
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
	models = make(map[int]distmlMatlab.MatModel)
	modelC = make(map[int]float64)
	modelD = 0.0
	gmodel = distmlMatlab.MatGlobalModel{nil}
	channel = make(chan message)
	// start a small cluster
	mynode = newNode(uint64(nID), []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
	if nID == 1 {
		mynode.raft.Campaign(mynode.ctx)
	}

	go mynode.run()
	//go printLeader()

	go updateGlobal(channel)
	//Hacky solution to the Matlab problem (Mathworks, please fix this!)
	// see: https://www.mathworks.com/matlabcentral/answers/305877-what-is-the-primary-message-table-for-module-77
	// and  https://github.com/JuliaInterop/MATLAB.jl/issues/47
	distmlMatlab.Hack()
	//Initialize TCP Connection and listenerfmt.Printf("Server initialized.\n")

	go clientListener(cl)

	for {
		conn, err := sl.AcceptTCP()
		checkError(err)
		go mynode.receive(conn)
	}

}
func clientListener(listen *net.TCPListener) {
	for {
		connC, err := listen.AcceptTCP()
		checkError(err)
		go connHandler(connC)
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
		flag := checkQueue(mynode.client[msg.Name])
		fmt.Printf("<-- Received commit request from %v.\n", msg.Name)
		if flag {
			processTestRequest(msg, conn)
		} else {
			//denied
			conn.Write([]byte("Pending tests are not complete"))
			fmt.Printf("--> Denied commit request from %v.\n", msg.Name)
			conn.Close()
		}
	case "global_request":
		//node is requesting the global model, will forward
		conn.Write([]byte("OK"))
		fmt.Printf("<-- Received global model request from %v.\n", msg.Name)
		genGlobalModel()
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		//node is submitting test results, will update its queue
		fmt.Printf("<-- Received completed test results from %v.\n", msg.Name)

		//replication
		temptestqueue := make(map[int]map[int]bool)
		queue := make(map[int]bool)
		queue[mynode.cnumhist[msg.Id]] = false
		temptestqueue[mynode.client[msg.Name]] = queue
		repstate := state{0, 0, 0, nil, nil, nil, temptestqueue, nil, msg}
		flag := replicate(repstate)

		if flag {
			conn.Write([]byte("OK"))
		} else {
			conn.Write([]byte("Try again"))
			fmt.Printf("--> Could not process test from %v.\n", msg.Name)
		}
		conn.Close()
	default:
		conn.Write([]byte("OK"))
		processJoin(msg, conn)
		conn.Close()
	}
}
func updateGlobal(ch chan message) {
	// Function that aggregates the global model and commits when ready
	for {
		m := <-ch
		id := mynode.cnumhist[m.Id]
		tempAggregate := mynode.tempmodel[id]
		tempAggregate.d += mynode.m.Model.Size
		tempAggregate.r[client[m.Name]] += mynode.m.Model.Weight
		mynode.tempmodel[id] = tempAggregate
		if modelD < tempAggregate.d {
			modelD = tempAggregate.d
		}

		//TODO replicate globalmodel
		if float64(tempAggregate.d) > float64(modelD)*0.6 {
			models[id] = tempAggregate.Model
			modelC[id] = tempAggregate.r
			modelC[id] = tempAggregate.c
			logger.LogLocalEvent("commit_complete")
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.Cnum)
		}
	}
}

func replicate(m state) bool {
	flag := false

	r := rand.Intn(999999999999)
	m.PropID = r
	buf := logger.PrepareSend("packing to servers", m)
	err := mynode.raft.Propose(mynode.ctx, buf)

	if err == nil {
		//block and check the status of the proposal
		//for !flag {
		//	if mynode.propID == r {
		flag = true
		//	}
		//}
	}

	return flag
}

func genGlobalModel() {
	modelstemp := models
	modelRtemp := modelR
	modelCtemp := modelC
	modelDtemp := modelD
	gmodel = distmlMatlab.CompactGlobal(modelstemp, modelRtemp, modelCtemp, modelDtemp)
}
func processTestRequest(m message, conn *net.TCPConn) {

	//initialize new aggregate
	tempweight := make(map[int]float64)
	r := m.Model.Weight
	d := m.Model.Size
	m.Model.Weight = 0.0
	m.Model.Size = 0.0
	tempweight[client[m.Name]] = r

	//replicate variables
	temptempmodel := make(map[int]aggregate)
	tempcnumhist := make(map[int]int)
	temptestqueue := make(map[int]map[int]bool)

	//update replicate cnum
	tempcnum := mynode.cnum + 1

	//update temp tempmodel and cnumhist
	tempcnumhist[tempcnum] = mynode.client[m.Name]
	temptempmodel[mynode.client[m.Name]] = aggregate{tempcnum,m.Model, tempweight, d, d}

	//update replicate queue
	for _, id := range mynode.client {
		if id != mynode.client[m.Name] {
			queue := make(map[int]bool)
			queue[tempcnumhist[tempcnum]] = true
			temptestqueue[id] = queue
		}
	}

	tempmsg := message{}
	repstate := state{0, 0, tempcnum, tempcnumhist, nil, temptempmodel, temptestqueue, nil, tempmsg}
	flag := replicate(repstate)
	time.Sleep(time.Second * 2)

	if flag {
		conn.Write([]byte("OK"))
		conn.Close()
		for name, id := range mynode.client {
			if id != mynode.client[m.Name] {
				sendTestRequest(name, id, mynode.cnum, m.Model)
			}
		}
	} else {
		conn.Write([]byte("Try again"))
		conn.Close()
		fmt.Printf("--> Denied commit request from %v.\n", m.Name)
	}
}
func sendTestRequest(name string, id, tcnum int, tmodel bclass.Model) {
	//create test request
	msg := message{tcnum, "server", "test_request", 0, 0, tmodel, gempty}
	//send the request
	fmt.Printf("--> Sending test request from %v to %v.", mynode.cnumhist[tcnum], name)
	err := tcpSend(mynode.claddr[id], msg)
	if err != nil {
		fmt.Printf(" [NO!]\n*** Could not send test request to %v.\n", name)
	}
}

func sendGlobal(m message) {
	fmt.Printf("--> Sending global model to %v.", m.Name)
	msg := message{m.Id, myaddr.String(), "server", "global_grant", m.Model, gmodel}
	tcpSend(claddr[client[m.Name]], msg)
}
func tcpSend(addr *net.TCPAddr, msg message) error {
	p := make([]byte, BUFFSIZE)
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
func processJoin(m message, conn *net.TCPConn) {
	//process depending on if it is a new node or a returning one
	tempaddr := make(map[int]string)
	tempclient := make(map[string]int)

	if _, ok := mynode.client[m.Name]; !ok {
		//adding a node that has never been added before
		id := mynode.maxnode
		tempmaxnode := mynode.maxnode + 1

		//replication
		tempclient[m.Name] = id
		tempaddr[id] = m.Type
		tempmsg := message{}
		repstate := state{0, tempmaxnode, 0, nil, tempclient, nil, nil, tempaddr, tempmsg}
		replicate(repstate)

		fmt.Printf("--- Added %v as node%v.\n", m.Name, id)
		for _, v := range mynode.tempmodel {
			sendTestRequest(m.Name, id, v.Cnum, v.Model)
		}
	} else {
		//node is rejoining, update address and resend the unfinished test requests
		id := mynode.client[m.Name]

		//replication
		tempaddr[id] = m.Type
		tempmsg := message{}
		repstate := state{0, 0, 0, nil, nil, nil, nil, tempaddr, tempmsg}
		replicate(repstate)

		fmt.Printf("--- %v at node%v is back online.\n", m.Name, id)
		for k, v := range mynode.testqueue[id] {
			if v {
				aggregate := mynode.tempmodel[k]
				sendTestRequest(m.Name, id, aggregate.Cnum, aggregate.Model)
			}
		}
	}
}
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
		os.Exit(1)
	}
}
