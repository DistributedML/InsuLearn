# distbayes
This project is protoype of a secured distributed ML framework using naive Bayesian classification, designed in partial fulfillment of CPSC538B 2016W1 Course Project.

This prototype shows the feasibility of data mining on highly sensitive distributed data and is aimed at applications where data security is paramount and thus data cannot be transferred from one node to another. The prototype is designed in such a way that multiple local statistical classification models are aggregated and incorporated into a global model at a centralized server using a Bayesian model averaging technique. The implementation of the prototype also includes a fault-tolerant implementation in which the server functions are replicated using the [Raft](https://raft.github.io/) consensus algorithm.

* bclass/       : Contains the classification library
* client/       : Contains an example of client implementation using the bclass library instrumented with GoVector
* client_raft/  : Contains an example of client implementation to interface with a raft replicated server, also instrumented with GoVector
* server/       : Contains an example of server implementation using the bclass library instrumented with GoVector
* server_raft/  : Contains an example of Raft replicated server implementation using the bclass library instrumented with GoVector
* testdata/     : Example training and testing data for an execution with 5 nodes.

### Installation

To use distbayes you must have a correctly configured go development environment, see [How to write Go Code](https://golang.org/doc/code.html)

Once you set up your environment, distbayes can be installed with the go tool command:

```
go get github.com/4180122/distbayes
```

The distbayes prototype and the bclass polynomial classification library require the following libraries, which can also be installed with the go tool command:

```
go get github.com/arcaneiceman/GoVector/govec
go get github.com/gonum/matrix/mat64
go get github.com/coreos/etcd/raft
```

### Client-Side Commands

The implementation of the client prompts the user for the following commands.

* read  : Reads data from disk.
* push  : Pushes trained model to server.
* pull  : Request global model from server.
* train : Train local model from local data (reports error).
* valid : Validate global model with local data.
* test  : Test local model with test data.
* testg : Test global model with test data.
* who   : Print node name.

### Examples

Here are a list of command line arguments that need to be passed to each example program.

#### client
* name           : A string representing the unique name of the node in the system
* ip:port        : Address that the client uses to listen to the server
* ip:port        : Address of the server
* data.txt       : Name of the file containing the features of training data used to train the local model
* label.txt      : Name of the file containing the labels of training data used to train the local model
* id             : A string representing the name of the node for GoVec log

#### client_raft
* name           : A string representing the unique name of the node in the system
* ip:port        : Address that the client uses to listen to the server
* serverlist.txt : Address of the servers (ip:port)
* data.txt       : Name of the file containing the features of training data used to train the local model
* label.txt      : Name of the file containing the labels of training data used to train the local model
* id             : A string representing the name of the node for GoVec log

#### server
* ip:port        : Address that the server uses to listen to the server
* id             : A string representing the name of the server for GoVec log

#### server_raft
* ip:port        : Address that the server uses to listen to the server
* raftlist.txt   : Address that the raft servers use to communicate to each other (ip:port)
* index          : Integer number mapping this server to an entry in the raftlist.txt
* id             : A string representing the name of the hospital for GoVec log

### Scripts
Four bash and batch scripts have been provided to initialize a regular run and a raft replicated run of the system in either Windows or Linux environment.

To execute a normal run of the system, run:

```sh
sh run.sh
```
or
```
run.bat
```

To execute a Raft replicated run of the system, run:

```sh
sh run_raft.sh
```
or
```
run_raft.bat
```
