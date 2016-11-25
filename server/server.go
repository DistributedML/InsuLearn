package main

import (
	"cpsc538b/proj/bclass"
	"flag"
	"fmt"
	"github.com/arcaneiceman/GoVector/govec"
	//"github.com/gonum/matrix/mat64"
	"io/ioutil"
	"net"
	//"strconv"
	"os"
	"strings"
)

var (
	cnum   int = 0
	myaddr *net.TCPAddr
	client map[string]int
	ctable map[int]int
	claddr map[int]*net.TCPAddr
	models map[int]bclass.Model
	logger *govec.GoLog
	l      *net.TCPListener
	gmodel bclass.GlobalModel
	gempty bclass.GlobalModel
)

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
	ctable = make(map[int]int)
	claddr = make(map[int]*net.TCPAddr)
	models = make(map[int]bclass.Model)

	//Parsing inputargs
	parseArgs()

	//Initialize TCP Connection and listener
	l, _ = net.ListenTCP("tcp", myaddr)
	fmt.Println("server initialized")

	for {
		conn, err := l.AcceptTCP()
		checkError(err)
		go connHandler(conn)
	}

	//xg := ReadData("x.txt")
	//yg := ReadData("y.txt")

	//modlist := make(map[int]bclass.Model)
	//X := make(map[int]*mat64.Dense)
	//Y := make(map[int]*mat64.Dense)
	//C := make(map[int]int)

	//X[1] = ReadData("x1.txt")
	//X[2] = ReadData("x2.txt")
	//X[3] = ReadData("x3.txt")
	//Y[1] = ReadData("y1.txt")
	//Y[2] = ReadData("y2.txt")
	//Y[3] = ReadData("y3.txt")

	//for i := range X {
	//	modlist[i] = bclass.RegLSBasisC(X[i], Y[i], 0.01, 2)
	//	modlist[i].Print()
	//}

	//dmax := 0
	//for k, model := range modlist {
	//	C[k] = 0
	//	dtemp := 0
	//	for ktest := range modlist {
	//		y := model.Predict(X[ktest])
	//		c, d := bclass.TestResults(y, Y[ktest])
	//		C[k] = C[k] + c
	//		dtemp = dtemp + d
	//	}
	//	if dtemp > dmax {
	//		dmax = dtemp
	//	}
	//}

	//modelg := bclass.GlobalModel{modlist, C, dmax}

	//yhat := modelg.Predict(xg)
	//cg, dg := bclass.TestResults(yhat, yg)

	////w := mat64.Formatted(Y[1], mat64.Prefix("    "))
	////fmt.Printf("X_1:\nw = %v\n\n", w)

	//modtot := bclass.RegLSBasisC(xg, yg, 0.01, 2)
	//yhatot := modtot.Predict(xg)
	//ct, dt := bclass.TestResults(yhatot, yg)

	//fmt.Println(float64(cg)/float64(dg), float64(ct)/float64(dt))

}

func connHandler(conn *net.TCPConn) {
	var msg message
	p := make([]byte, 1048576)
	conn.Read(p)
	logger.UnpackReceive("Received message", p, &msg)
	switch msg.Type {
	case "commit_request":
		// node is sending a model, must forward to others for testing
		fmt.Println("Received commit request.")
		conn.Close()
		go sendTestRequest(msg)
	case "global_request":
		conn.Close()
		// node is asking for global model
	case "test_complete":
		conn.Close()
		//update the pending commit and merge if complete
	default:
		// respond to ping
	}
}

func sendTestRequest(m message) {
	cnum++
	for name, id := range client {
		if id != client[m.Name] {
			//forward test requst
			msg := message{cnum, "server", "test_request", 0, 0, m.Model, gempty}
			tcpSend(claddr[id], msg)
			fmt.Printf("Sent test request to %v.\n", name)
		}
	}
}

func tcpSend(addr *net.TCPAddr, msg message) {
	conn, err := net.DialTCP("tcp", nil, addr)
	checkError(err)
	outbuf := logger.PrepareSend(msg.Type, msg)
	_, err = conn.Write(outbuf)
	checkError(err)
}

//func tcpReply(msg message, conn *net.TCPConn) {
//	//conn, err := net.DialTCP("TCP", svaddr)
//	//checkError(err)
//	outbuf := logger.PrepareSend(msg.Type, msg)
//	conn.Write(outbuf)
//}

func parseArgs() {
	flag.Parse()
	inputargs := flag.Args()
	var err error
	if len(inputargs) < 2 {
		fmt.Println("Not enough inputs")
		return
	}
	myaddr, err = net.ResolveTCPAddr("tcp", inputargs[0])
	checkError(err)
	getNodeAddr(inputargs[1])
	logger = govec.Initialize(inputargs[2], inputargs[2])
}

func getNodeAddr(slavefile string) {
	dat, err := ioutil.ReadFile(slavefile)
	checkError(err)
	nodestr := strings.Split(string(dat), "\n")
	for i := 0; i < len(nodestr)-1; i++ {
		nodeaddr := strings.Split(nodestr[i], " ")
		client[nodeaddr[0]] = i
		claddr[i], _ = net.ResolveTCPAddr("tcp", nodeaddr[1])
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
