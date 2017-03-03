# distbayes
This project is protoype of a secured distributed ML framework using naive Bayesian classification, designed in partial fulfillment of CPSC538B 2016W1 Course Project.

This prototype shows the feasibility of data mining on highly sensitive distributed data and is aimed at applications where data security is paramount and thus data cannot be transferred from one node to another. The prototype is designed in such a way that multiple local statistical classification models are aggregated and incorporated into a global model at a centralized server using a Bayesian model averaging technique. The implementation of the prototype also includes a fault-tolerant implementation in which the server functions are replicated using the [Raft](https://raft.github.io/) consensus algorithm.

* bclass/       : A simple classification library and Bayesian aggregation scheme implemented in Go
* client/       : Examples of different client implementations using the bclass or distmlMatlab libraries with and without replication, some of which are instrumented with GoVector
* distmlMatlab  : Library for interfacing with built-in MATLAB classification, regression, and ensemble techniques
* server/       : Examples of different client implementations using the bclass or distmlMatlab libraries with and without replication, some of which are instrumented with GoVector
* testdata/     : Example training and testing data for an execution with 5 nodes.

## Compatibility

The Go only implementations which use the bclass library is cross-platform compatible and has been tested on 64 bit Windows, Linux, and OS X. The MATLAB based implementations (distmlMatlab library) interface with MATLAB using the [MATLAB Engine API for C](https://www.mathworks.com/help/matlab/calling-matlab-engine-from-c-c-and-fortran-programs.html), and require [nonshared, single session, engine instances](https://www.mathworks.com/help/matlab/apiref/engopensingleuse.html) that are only supported on 64 bit Windows platforms up to MATLAB version R2016b. The distmlMatlab implementations have been tested on 64 bit Windows 7, 8.1, Server 2012 and MATLAB versions R2011b, R2014b, R2015a, and R2016b.

## Installation

To use distbayes you must have a correctly configured go development environment, see [How to write Go Code](https://golang.org/doc/code.html)

Once you set up your environment, distbayes can be installed with the go tool command:

```
go get github.com/4180122/distbayes
```

#### bclass Dependencies (Cross-Platform 64 bit)
The bclass polynomial classification library require the [gonum matrix library](https://github.com/gonum/matrix), which can also be installed with the go tool command:

```
go get github.com/gonum/matrix/mat64
```
#### distmlMatlab Dependencies (64 bit Windows Only)
The following steps are required to enable communication between Go and MATLAB via cgo and MATLAB Engine API.

 1. Install 64 bit MATLAB (R2011b or newer) into default directory
 2. Set MATLAB run-time library path (see [instructions](https://www.mathworks.com/help/matlab/matlab_external/building-and-running-engine-applications-on-windows-operating-systems.html))
 3. Install 64-bit gcc compiler for windows ([TDM-GCC-64 5.1.0](http://tdm-gcc.tdragon.net/download))
 4. Create the following symbolic links because the cgo linker does not play well with windows paths (must be done with Administrative privilages):

	```
	mklink /D <distbayes_root_directory>\include <matlab_install_directory>\extern\incldue
	
	mklink /D <distbayes_root_directory>\distmlMatlab\include <matlab_install_directory>\extern\incldue
	
	mklink /D <distbayes_root_directory>\lib <matlab_install_directory>\bin\win64
	
	mklink /D <distbayes_root_directory>\distmlMatlab\lib <matlab_install_directory>\bin\win64 
	```
 5. Ensure that the distmlMatlab\matlabfunctions\ folder has write privilages
 6. Update MATLAB_FUNCTION_PATH in distmlMatlab\matlabfun.c to the distmlMatlab\matlabfunctions folder
 6. Update batch files with apropriate paths for MATLAB run-time and gcc compiler

#### Other Dependencies
The following libraries are required for Raft replication and GoVector instrumentation:
```
go get github.com/arcaneiceman/GoVector/govec
go get github.com/coreos/etcd/raft
```


## Client-Side Commands

The implementation of the client prompts the user for the following commands.

* read  : Reads data from disk.
* push  : Pushes trained model to server.
* pull  : Request global model from server.
* train : Train local model from local data (reports error).
* valid : Validate global model with local data.
* test  : Test local model with test data.
* testg : Test global model with test data.
* who   : Print node name.

###Examples

Here are a list of command line arguments that need to be passed to each example program.

#### client
* name            : A string representing the unique name of the node in the system
* ip:port         : Address that the client uses to listen to the server
* ip:port         : Address of the server
* train_data.txt  : Name of the file containing the features of training data used to train the local model
* train_label.txt : Name of the file containing the labels of training data used to train the local model
* test_data.txt   : Name of the file containing the features of testing data used to test the local and global models
* test_label.txt  : Name of the file containing the labels of testing data used to test the local and global models
* id              : A string representing the name of the node for GoVec log

#### client_raft
* name            : A string representing the unique name of the node in the system
* ip:port         : Address that the client uses to listen to the server
* serverlist.txt  : Name of the file containing addresses of the servers (ip:port)
* train_data.txt  : Name of the file containing the features of training data used to train the local model
* train_label.txt : Name of the file containing the labels of training data used to train the local model
* test_data.txt   : Name of the file containing the features of testing data used to test the local and global models
* test_label.txt  : Name of the file containing the labels of testing data used to test the local and global 
* id              : A string representing the name of the node for GoVec log

#### server
* ip:port         : Address that the server uses to listen to the server
* id              : A string representing the name of the server for GoVec log

#### server_raft
* ip:port         : Address that the server uses to listen to the server
* raftlist.txt    : Name of the file containing addresses that the raft servers use to communicate to each other (ip:port)
* index           : Integer number mapping this server to an entry in the raftlist.txt
* id              : A string representing the name of the hospital for GoVec log

## Scripts
#### bclass (Cross-Platform 64 bit)
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

#### distmlMatlab (64 bit Windows Only)
To execute a normal run of the system, run:

```
run_matlab.bat
```

To execute a Raft replicated run of the system, run:

```
run_matlab_raft.bat
```

#### distmlMatlab Azure Scripts (64 bit Windows Only)
The following scripts are used to deploy an automated version of the client and server on Microsoft Azure Cloud:

 * Azure_client.bat
 * Azure_server.bat
 * Azure_raftserver.bat


#### Processing Log Files
The following scripts are used to process GoVector generated log files:

 * concat.bat
 * parse-log.bat
