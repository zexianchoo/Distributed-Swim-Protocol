package main

import (
	pb "MP2/protos"
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	is_introducer       = flag.Bool("introducer", false, "Set this flag if this server is the introducer")
	is_introducer_short = flag.Bool("i", false, "alias for --introducer")
	introducer_instance *Introducer
	server_instance     *Server
)

func main() {
	flag.Parse()

	// Set log level to Info
	logrus.SetLevel(LOG_LEVEL)

	// Check if either introducer flag is set
	introducer_flag := *is_introducer || *is_introducer_short

	// grpc instance
	s := grpc.NewServer(grpc.MaxRecvMsgSize(maxMsgSize), grpc.MaxSendMsgSize(maxMsgSize))

	// define server
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	server_instance = &Server{
		Memb_list:   make(map[string]*pb.NodeInfo),
		Incarnation: 0, // always begins with 0 and scales up
		Host:        hostname,
		Port:        NETWORK_PORT,
		Version:     getVersion(),

		// default suspicion to initial const
		ToggleSusFlag:    USE_SUSPECT_FLAG,
		ToggleSusVersion: 0,
	}

	// if introducer, create introducer instance
	if introducer_flag {
		introducer_instance = &Introducer{
			Server: *server_instance,
		}
		//  start a tcp server for joins/leaves
		// add self to list
		introducer_instance.updateMembershipInstance(hostname, NETWORK_PORT, "alive", 0)
		go introducer_instance.startTCPServer()
	} else {
		// introducer doesn't need to join
		go server_instance.joinGroup()
	}

	// register log query
	pb.RegisterLogQueryServiceServer(s, server_instance)

	// start the udp
	go server_instance.startUDPServer()

	// stream membership pinging
	go server_instance.StreamMembershipPing()

	fmt.Printf("Server is running on hostname: %s:%s\n", hostname, NETWORK_PORT)
	fmt.Println("=============================================================================")
	logrus.Infof("Current incarnation number: %d\n", server_instance.Incarnation)

	reader := bufio.NewScanner(os.Stdin)

	// await stdin input
	for reader.Scan(){
		// Read input from stdin until newline
		input := reader.Text()
		// if err != nil {
		// 	logrus.Errorf("Error reading input: %v", err)
		// 	return
		// }
		
		// // trim newline
		// input = input[:len(input)-1]

		// handle inputs:
		switch input {
		case "exit":
			return
		case "enable_sus":
			USE_SUSPECT_FLAG = true
			server_instance.Mu.Lock()
			server_instance.ToggleSusFlag = true
			server_instance.ToggleSusVersion += 1
			server_instance.Mu.Unlock()

		case "disable_sus":
			USE_SUSPECT_FLAG = false
			server_instance.Mu.Lock()
			server_instance.ToggleSusFlag = false
			server_instance.ToggleSusVersion += 1
			server_instance.Mu.Unlock()

		case "status_sus":
			fmt.Printf("Suspicion On Status: %t\n", server_instance.ToggleSusFlag)
			fmt.Printf("Suspicion On Version: %d\n", server_instance.ToggleSusVersion)
		case "list_mem":
			// Print the hostnames in sorted order for easy viewing
			names := make([]string, 0, len(server_instance.Memb_list))
			for k := range server_instance.Memb_list {
				names = append(names, k)
			}
			sort.Strings(names)
			fmt.Printf("Current Membership List\n")
			for _, k := range names {
				fmt.Printf("Host: %s, State: %s, Incarnation: %d\n", k, server_instance.Memb_list[k].GetState(), server_instance.Memb_list[k].GetIncarnation())
			}
		case "list_self":
			fmt.Printf("Current hostname: %s", hostname)
		case "leave":
			server_instance.leaveGroup()
			return
		default:
			fmt.Printf("Invalid command!\n\n")
		}
	}
}
