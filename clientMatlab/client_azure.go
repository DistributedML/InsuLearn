package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/4180122/distbayes/distmlMatlab"
	"github.com/arcaneiceman/GoVector/govec"
	//"github.com/gonum/matrix/mat64"
	//"io/ioutil"
	"net"
	"os"
	//"strconv"
	//"strings"
	"time"
)

const BUFFSIZE = 200000

//10485760

var (
	cnum      int = 0
	name      string
	inputargs []string
	myaddr    *net.TCPAddr
	svaddr    *net.TCPAddr
	model     distmlMatlab.MatModel
	logger    *govec.GoLog
	X         string
	Y         string
	Xt        string
	Yt        string
	l         *net.TCPListener
	gmodel    distmlMatlab.MatGlobalModel
	gempty    distmlMatlab.MatGlobalModel
	committed bool
)

type message struct {
	Id     int
	Ip     string
	Name   string
	Type   string
	Model  distmlMatlab.MatModel
	GModel distmlMatlab.MatGlobalModel
}

func main() {
	//Parsing inputargs
	parseArgs()

	//Hacky solution to the Matlab problem (Mathworks, please fix this!)
	// see: https://www.mathworks.com/matlabcentral/answers/305877-what-is-the-primary-message-table-for-module-77
	// and  https://github.com/JuliaInterop/MATLAB.jl/issues/47
	distmlMatlab.Hack()

	//Initialize stuff
	model = distmlMatlab.NewModel(X, Y)

	//Initialize TCP Connection and listener
	l, _ = net.ListenTCP("tcp", myaddr)
	fmt.Printf("Node initialized as %v.\n", name)
	go listener()
	requestJoin()

	committed = false

	//Main function of this server
	for {
		//parseUserInput()
		time.Sleep(time.Duration(1 * time.Second))
		if !committed {
			requestCommit()
		}
	}
}

func listener() {
	for {
		conn, err := l.AcceptTCP()
		checkError(err)
		go connHandler(conn)
	}
}

func connHandler(conn *net.TCPConn) {
	p := make([]byte, BUFFSIZE)
	conn.Read(p)
	var msg message
	logger.UnpackReceive("Received message", p, &msg)
	switch msg.Type {
	case "test_request":
		// server is asking me to test
		conn.Write([]byte("OK"))
		go testModel(msg.Id, msg.Model)
	case "global_grant":
		// server is sending global model
		conn.Write([]byte("OK"))
		gmodel = msg.GModel
		fmt.Printf("\n <-- Pulled global model from server.\nEnter command: ")
		//go testGlobal(msg.GModel)
	default:
		// respond to ping
		conn.Write([]byte("Unknown command"))
	}
	conn.Close()
}

func parseUserInput() {
	var ident string
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter command: ")
	text, _ := reader.ReadString('\n')
	//Windows adds its own strange carriage return, the following lines fix it
	if text[len(text)-2] == '\r' {
		ident = text[0 : len(text)-2]
	} else {
		ident = text[0 : len(text)-1]
	}
	switch ident {
	case "read":
		//x = readData(inputargs[3])
		//y = readData(inputargs[4])
		fmt.Printf(" --- Local data updated.\n")
	case "train":
		model = distmlMatlab.NewModel(X, Y) //GOOD
		fmt.Printf(" --- Local model error on local data is: %v.\n", model.Weight)
	case "push":
		requestCommit()
	case "pull":
		requestGlobal()
	case "valid":
		acc, _ := distmlMatlab.GetErrorGlobal(X, Y, gmodel)
		fmt.Printf(" --- Global model error on local data is: %v.\n", acc)
	case "test":
		acc := distmlMatlab.GetError(Xt, Yt, model)
		fmt.Printf(" --- Local model error on test data is: %v.\n", acc)
	case "testg":
		acc, _ := distmlMatlab.GetErrorGlobal(Xt, Yt, gmodel)
		fmt.Printf(" --- Global model error on test data is: %v.\n", acc)
	case "who":
		fmt.Printf("%v\n", name)
	default:
		fmt.Printf(" Command not recognized: %v.\n\n", ident)
		fmt.Printf("  Choose from the following commands\n")
		fmt.Printf("  read  -- Read data from disk\n")
		fmt.Printf("  push  -- Push trained model to server\n")
		fmt.Printf("  pull  -- Obtain global model from server\n")
		fmt.Printf("  train -- Train model from data (reports error)\n")
		fmt.Printf("  valid -- Validate global model with local data\n")
		fmt.Printf("  test  -- Test local model with test data\n")
		fmt.Printf("  testg -- Test global model with test data\n")
		fmt.Printf("  who   -- Print node name\n\n")
	}
}

func requestJoin() {
	msg := message{cnum, myaddr.String(), name, "join_request", model, gempty}
	fmt.Printf(" --> Asking server to join.")
	tcpSend(msg)
}

func requestCommit() {
	cnum++
	msg := message{cnum, myaddr.String(), name, "commit_request", model, gempty}
	fmt.Printf(" --> Pushing local model to server.")
	tcpSend(msg)
}

func requestGlobal() {
	msg := message{cnum, myaddr.String(), name, "global_request", model, gempty}
	fmt.Printf(" --> Requesting global model from server.")
	tcpSend(msg)
}

func testModel(id int, testmodel distmlMatlab.MatModel) {
	//func testModel(id int, testmodel bclass.Model) {
	fmt.Printf("\n <-- Received test requset.\nEnter command: ")
	distmlMatlab.TestModel(X, Y, &testmodel)
	msg := message{id, myaddr.String(), name, "test_complete", testmodel, gempty}
	fmt.Printf("\n --> Sending completed test requset.")
	tcpSend(msg)
	fmt.Printf("Enter command: ")
}

func tcpSend(msg message) {
	p := make([]byte, BUFFSIZE)
	conn, err := net.DialTCP("tcp", nil, svaddr)
	checkError(err)
	outbuf := logger.PrepareSend(msg.Type, msg)
	_, err = conn.Write(outbuf)
	checkError(err)
	n, _ := conn.Read(p)
	if string(p[:n]) != "OK" {
		fmt.Printf(" [NO!]\n *** Request was denied by server: %v.\nEnter command: ", string(p[:n]))
	} else {
		fmt.Printf(" [OK]\n")
		if msg.Type == "commit_request" {
			committed = true
		}
	}
}

func parseArgs() {
	flag.Parse()
	inputargs = flag.Args()
	var err error
	if len(inputargs) < 2 {
		fmt.Printf("Not enough inputs.\n")
		return
	}
	name = inputargs[0]
	myaddr, err = net.ResolveTCPAddr("tcp", inputargs[1])
	checkError(err)
	svaddr, err = net.ResolveTCPAddr("tcp", inputargs[2])
	checkError(err)
	X = inputargs[3]
	Y = inputargs[4]
	Xt = "C:/work/src/github.com/4180122/distbayes/testdata/xv.txt"
	Yt = "C:/work/src/github.com/4180122/distbayes/testdata/yv.txt"
	logger = govec.Initialize(inputargs[0], inputargs[5])
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
