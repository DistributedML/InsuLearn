package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"time"
	"net"
	"os"
	"strconv"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"golang.org/x/net/context"
	"github.com/arcaneiceman/GoVector/capture"
  	"github.com/arcaneiceman/GoVector/govec"
)
var (
  ln                    net.Listener
  IncommingMessage      Message
  Outgoing				Message
  buf [512]             byte
  naddr    				map[int]string
  Logger				*govec.GoLog
	nID 				uint64
)
const hb = 1

type Message struct{
	Ctx   context.Context
	Msg   raftpb.Message
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
			ElectionTick:    10 * hb,
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
		log.Println(raft.DescribeMessage(m, nil))
		Outgoing.Ctx= n.ctx
		Outgoing.Msg=m
		conn, _ := net.Dial("tcp", naddr[int(m.To)])
    	 outBuf := Logger.PrepareSend("Sending message to other node",Outgoing)
     	 capture.Write(conn.Write, outBuf)
         conn.Close()
		// send message to other node

		//nodes[int(m.To)].receive(n.ctx, m)
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

func (n *node) receive() {
      conn, _ := ln.Accept()
      // Echo all incoming data.
      capture.Read(conn.Read, buf[0:])
      Logger.UnpackReceive("Received Message From Client", buf[0:],&IncommingMessage)

      // output message received
      fmt.Print("Message Received:", IncommingMessage)
      // sample process for string received
    conn.Close()
	n.raft.Step(IncommingMessage.Ctx,IncommingMessage.Msg)
}

var (
	nodes = make(map[int]*node)
)

func main() {
	sID:=os.Args[1]
	nID,_=strconv.ParseUint(sID,10,64)
	InID:=int(nID)

	naddr= make(map[int]string)
	logfile := os.Args[2]
	naddr[1]="127.0.0.1:6001"
	naddr[2]="127.0.0.1:6002"
	naddr[3]="127.0.0.1:6003"
	naddr[4]="127.0.0.1:6004"
	naddr[5]="127.0.0.1:6005"
	
	 ln, _ = net.Listen("tcp", naddr[InID])
 	 Logger = govec.Initialize(logfile,logfile)
	// start a small cluster
	nodes[InID] = newNode(nID, []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
	nodes[InID].raft.Campaign(nodes[InID].ctx)
	go nodes[InID].receive()
	go nodes[InID].run()

	// nodes[2] = newNode(2, []raft.Peer{{ID: 1}, {ID: 2}, {ID: 3}})
	// go nodes[2].run()

	// nodes[3] = newNode(3, []raft.Peer{})
	// go nodes[3].run()
	// nodes[2].raft.ProposeConfChange(nodes[2].ctx, raftpb.ConfChange{
	// 	ID:      3,
	// 	Type:    raftpb.ConfChangeAddNode,
	// 	NodeID:  3,
	// 	Context: []byte(""),
	// })
	if InID==1{
		for nodes[1].raft.Status().Lead != 1 {
			time.Sleep(100 * time.Millisecond)
		}
		nodes[InID].raft.Propose(nodes[2].ctx, []byte("mykey1:myvalue1"))
	}

	//nodes[InID].raft.Propose(nodes[2].ctx, []byte("mykey1:myvalue1"))
	// nodes[2].raft.Propose(nodes[2].ctx, []byte("mykey2:myvalue2"))
	// nodes[3].raft.Propose(nodes[2].ctx, []byte("mykey3:myvalue3"))

	// Wait for proposed entry to be commited in cluster.
	// Apperently when should add an uniq id to the message and wait until it is
	// commited in the node.
	fmt.Printf("** Sleeping to visualize heartbeat between nodes **\n")
	time.Sleep(2000 * time.Millisecond)

	// Just check that data has been persited
	for i, node := range nodes {
		fmt.Printf("** Node %v **\n", i)
		for k, v := range node.pstore {
			fmt.Printf("%v = %v\n", k, v)
		}
		fmt.Printf("*************\n")
		for true{

		}


	}
}