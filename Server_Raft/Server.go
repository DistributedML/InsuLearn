package main

import (
	"flag"
	"fmt"
	"github.com/4180122/distbayes/bclass"
	"github.com/arcaneiceman/GoVector/govec"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"golang.org/x/net/context"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	naddr   map[int]string
	logger  *govec.GoLog
	nID     int
	maxnode int = 0
	myaddr  *net.TCPAddr
	models  map[int]bclass.Model
	modelC  map[int]int
	modelD  int
	channel chan message
	gmodel  bclass.GlobalModel
	gempty  bclass.GlobalModel
	mynode  *node
)

const hb = 1

type aggregate struct {
	Cnum  int
	Model bclass.Model
	C     int
	D     int
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
	id        uint64
	ctx       context.Context
	store     *raft.MemoryStorage
	cfg       *raft.Config
	raft      raft.Node
	cnum      int
	cnumhist  map[int]int
	client    map[string]int
	tempmodel map[int]aggregate
	testqueue map[int]map[int]bool
	claddr    map[int]*net.TCPAddr
	cstate    state
	ticker    <-chan time.Time
	done      <-chan struct{}
}

type state struct {
	Cnum      int
	Cnumhist  map[int]int
	Client    map[string]int
	Tempmodel map[int]aggregate
	Testqueue map[int]map[int]bool
	Claddr    map[int]string
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
			HeartbeatTick:   hb,
			Storage:         store,
			MaxSizePerMsg:   math.MaxUint16,
			MaxInflightMsgs: 1024,
		},
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
		if repstate.Cnum > mynode.cnum {
			mynode.cnum = repstate.Cnum
		}
		for k, v := range repstate.Cnumhist {
			mynode.cnumhist[k] = v
		}
		for k, v := range repstate.Client {
			mynode.client[k] = v
		}
		for k, v := range repstate.Tempmodel {
			mynode.tempmodel[k] = v
		}
		for k, v := range repstate.Claddr {
			mynode.claddr[k], _ = net.ResolveTCPAddr("tcp", v)
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
		}
		mynode.cstate = repstate
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
	models = make(map[int]bclass.Model)
	modelC = make(map[int]int)
	modelD = 0
	gmodel = bclass.GlobalModel{models, modelC, modelD}
	channel = make(chan message)

	// start a small cluster
	mynode = newNode(uint64(nID), []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
	if nID == 1 {
		mynode.raft.Campaign(mynode.ctx)
	}

	go mynode.run()
	go printLeader()

	go updateGlobal(channel)
	fmt.Println("my addre", myaddr)

	//Initialize TCP Connection and listener

	fmt.Printf("Server initialized.\n")

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

//useless function!!
func printLeader() {
	for {
		time.Sleep(2 * time.Second)
		leadern := mynode.raft.Status().Lead
		fmt.Println("leader: ", leadern)
		fmt.Println(mynode.client)
		//fmt.Println(mynode.pstore)
		//fmt.Println("my pstore: ", mynode.pstore)
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
		sendGlobal(msg)
		conn.Close()
	case "test_complete":
		//node is submitting test results, will update its queue
		fmt.Printf("<-- Received completed test results from %v.\n", msg.Name)
		mynode.testqueue[mynode.client[msg.Name]][mynode.cnumhist[msg.Id]] = false

		//replication
		temptestqueue := make(map[int]map[int]bool)
		queue := make(map[int]bool)
		queue[mynode.cnumhist[msg.Id]] = false
		temptestqueue[mynode.client[msg.Name]] = queue
		repstate := state{0, nil, nil, nil, temptestqueue, nil}
		flag := replicate(repstate)

		if flag {
			conn.Write([]byte("OK"))
			//update the pending commit and merge if complete
			channel <- msg
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
		tempAggregate.C += m.C
		tempAggregate.D += m.D
		mynode.tempmodel[id] = tempAggregate
		if modelD < tempAggregate.D {
			modelD = tempAggregate.D
		}

		//replicate
		temptempmodel := make(map[int]aggregate)
		temptempmodel[id] = tempAggregate
		repstate := state{0, nil, nil, temptempmodel, nil, nil}
		replicate(repstate)

		if float64(tempAggregate.D) > float64(modelD)*0.6 {
			models[id] = tempAggregate.Model
			modelC[id] = tempAggregate.C
			gmodel = bclass.GlobalModel{models, modelC, modelD}
			logger.LogLocalEvent("commit_complete")
			fmt.Printf("--- Committed model%v for commit number: %v.\n", id, tempAggregate.Cnum)
		}
	}
}

func replicate(m state) bool {
	flag := false
	buf := logger.PrepareSend("packing to servers", m)
	err := mynode.raft.Propose(mynode.ctx, buf)

	//keep checking the status of the proposal for 2 seconds then let client know
	//for !flag {
	//	time.Sleep(1000 * time.Millisecond)
	//	if reflect.DeepEqual(mynode.cstate, m) {
	//		flag = true
	//	}
	//	fmt.Println(flag, mynode.cstate, m)
	//}
	if err == nil {
		flag = true
	}

	return flag
}

func processTestRequest(m message, conn *net.TCPConn) {

	//replicate variables
	temptempmodel := make(map[int]aggregate)
	tempcnumhist := make(map[int]int)
	temptestqueue := make(map[int]map[int]bool)

	mynode.cnum++
	//update replicate cnum
	tempcnum := mynode.cnum

	//initialize new aggregate
	mynode.tempmodel[mynode.client[m.Name]] = aggregate{mynode.cnum, m.Model, m.C, m.D}
	mynode.cnumhist[mynode.cnum] = mynode.client[m.Name]
	//update temp tempmodel and cnumhist
	tempcnumhist[mynode.cnum] = mynode.client[m.Name]
	temptempmodel[mynode.client[m.Name]] = aggregate{mynode.cnum, m.Model, m.C, m.D}

	for _, id := range mynode.client {
		if id != mynode.client[m.Name] {
			//increment tests in queue for that node
			if queue, ok := mynode.testqueue[id]; !ok {
				queue := make(map[int]bool)
				queue[mynode.cnumhist[tempcnum]] = true
				mynode.testqueue[id] = queue
				//update replicate queue
				temptestqueue[id] = queue
			} else {
				queue[mynode.cnumhist[tempcnum]] = true
			}
		}
	}

	repstate := state{tempcnum, tempcnumhist, nil, temptempmodel, temptestqueue, nil}
	flag := replicate(repstate)

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
	msg := message{m.Id, "server", "global_grant", 0, 0, m.Model, gmodel}
	tcpSend(mynode.claddr[mynode.client[m.Name]], msg)
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
	for _, v := range mynode.testqueue[id] {
		if flag && v {
			flag = false
		}
	}
	return flag
}

func processJoin(m message, conn *net.TCPConn) {
	//process depending on if it is a new node or a returning one
	if _, ok := mynode.client[m.Name]; !ok {
		//adding a node that has never been added before
		id := maxnode
		maxnode++
		mynode.client[m.Name] = id
		mynode.claddr[id], _ = net.ResolveTCPAddr("tcp", m.Type)

		//replication
		tempaddr := make(map[int]string)
		tempclient := make(map[string]int)
		tempclient[m.Name] = id
		tempaddr[id] = m.Type
		repstate := state{0, nil, tempclient, nil, nil, tempaddr}
		replicate(repstate)

		fmt.Printf("--- Added %v as node%v.\n", m.Name, id)
		for _, v := range mynode.tempmodel {
			sendTestRequest(m.Name, id, v.Cnum, v.Model)
		}
	} else {
		//node is rejoining, update address and resend the unfinished test requests
		id := mynode.client[m.Name]
		mynode.claddr[id], _ = net.ResolveTCPAddr("tcp", m.Type)
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
