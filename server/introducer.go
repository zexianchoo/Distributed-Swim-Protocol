package main

import (
	pb "MP2/protos"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// extend server with introducer
type Introducer struct {
	Server
}

/**
*	handles join requests from servers.
*
**/
func (i *Introducer) handleJoin(conn net.Conn, request *pb.MembershipPush) {
	logrus.Debugf("Received join request from %s:%s", request.SourceHost, request.SourcePort)
	logrus.Debugf("Current membership list %v", i.Memb_list)

	// update incarnation's membership list with the new node that joined,
	i.updateMembershipInstance(request.SourceHost, request.SourcePort, "alive", request.GetIncarnation())

	join_ack := i.createMembershipAck("join_ack", i.Memb_list, i.Host, i.Port, i.Incarnation)

	marshalSendProtoTCP(&join_ack, conn)

}

/**
*	handles leave requests from servers.
*
**/
func (i *Introducer) handleLeave(request *pb.MembershipPush) {
	logrus.Infof("Received leave request from %s:%s", request.SourceHost, request.SourcePort)
	i.Mu.Lock()
	delete(i.Memb_list, request.SourceHost)
	i.Mu.Unlock()
}

/**
*	recieves join requests from TCP and passes to handleJoin
*
**/
func (i *Introducer) receiveJoinRequests(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, maxMsgSize)
	nbytes, err := conn.Read(buffer)

	if err != nil {
		logrus.Errorf("[receiveJoinRequests] Error reading tcp, %v\n", err)
	}

	var request pb.MembershipPush
	err = proto.Unmarshal(buffer[:nbytes], &request)
	if err != nil {
		logrus.Errorf("[receiveJoinRequests] Error proto Unmarshal, %v\n", err)
	}

	// only handle joins when request header == join
	if request.Header == "join_req" {
		i.handleJoin(conn, &request)
	} else if request.Header == "leave_req" {
		i.handleLeave(&request)
	}
}

func (i *Introducer) startTCPServer() {
	listener, err := net.Listen("tcp", ":"+INTRODUCER_PORT)
	if err != nil {
		logrus.Errorf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	logrus.Infof("TCP server listening on %s:%s", i.Host, INTRODUCER_PORT)

	// awaits tcp connections and handles recieve and join requests
	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("Failed to accept TCP connection: %v", err)
			continue
		}

		// Handle the connection in a separate goroutine
		go i.receiveJoinRequests(conn)
	}
}
