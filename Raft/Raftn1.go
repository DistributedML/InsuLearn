package main

import (
	"bytes"
	"fmt"
	"github.com/arcaneiceman/GoVector/govec"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"golang.org/x/net/context"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	ln     *net.TCPListener
	naddr  map[int]string
	Logger *govec.GoLog
	nID    uint64
)

const hb = 1

// clean this up maybe?
type Message struct {
	Msg raftpb.Message
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
		outBuf := Logger.PrepareSend("Sending message to other node", msg)
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
	Logger.UnpackReceive("Received Message From Client", buf, &imsg)
	conn.Close()
	//leadern := n.raft.Status().Lead
	//fmt.Println("leader: ", leadern)
	n.raft.Step(n.ctx, imsg.Msg)
}

var (
	nodes = make(map[int]*node)
)

func main() {
	sID := os.Args[1]
	nID, _ = strconv.ParseUint(sID, 10, 64)
	InID := int(nID)

	naddr = make(map[int]string)
	logfile := os.Args[2]
	naddr[1] = "127.0.0.1:6001"
	naddr[2] = "127.0.0.1:6002"
	naddr[3] = "127.0.0.1:6003"
	naddr[4] = "127.0.0.1:6004"
	naddr[5] = "127.0.0.1:6005"

	nodeaddr, _ := net.ResolveTCPAddr("tcp", naddr[InID])
	ln, _ = net.ListenTCP("tcp", nodeaddr)
	Logger = govec.Initialize(logfile, logfile)
	// start a small cluster
	nodes[InID] = newNode(nID, []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
	nodes[InID].raft.Campaign(nodes[InID].ctx)
	go nodes[InID].run()
	go printLeader(nodes[InID], InID)

	// Wait for proposed entry to be commited in cluster.
	// Apperently when should add an uniq id to the message and wait until it is
	// commited in the node.
	fmt.Printf("** Sleeping to visualize heartbeat between nodes **\n")

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
				fmt.Printf("%v = %v", k, v)
			}
		}
	}
}
