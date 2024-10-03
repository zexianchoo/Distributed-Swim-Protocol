package main

import (
	pb "MP2/protos"
	"context"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/**
*	Helper function that determines if node being added is valid to be added
*
**/
func (s *Server) validNodeEntry(node *pb.NodeInfo) bool {

	elapsed_sec := timestamppb.Now().GetSeconds() - node.Timestamp.GetSeconds()
	elapsed_nano := timestamppb.Now().GetNanos() - node.Timestamp.GetNanos()
	elapsed_t := elapsed_sec*1e9 + int64(elapsed_nano)

	// if we have exceeded the timeodut, change pending to dead. Let dead propagate before removing
	// declare nodes that are suspected for very long to be dead. Let dead propagate before removing
	if node.State == "suspect" && elapsed_t > int64(SUSPECT_TIMEOUT)*1e9 {
		return false
	}
	// delete nodes that have been dead for more than 1 second
	if node.State == "dead" && elapsed_t > int64(PENDING_TIMEOUT)*1e9 {
		return false
	}
	return true
}

/**
*	NodeInfo struct which carries Membership information for an instance of a ip address
*	Fields:
*		 Hostname(string): e.g. "fa24-cs425...illinois.edu"
*		 Port(string): 5138 by default.
*		 State(string): ["alive", "suspect", "dead"]. Contains current known state of node
*		 Incarnation(int32): Version number. Incremented by Introducer when node leaves & rejoins
**/
type NodeInfo struct {
	Hostname    string
	Port        string
	State       string
	Incarnation int32
	Time        timestamppb.Timestamp
}

/**
*	Server struct which carries the Servers for both LogQuery and Membership Services
*	Fields:
*		 Memb_list(map[string]NodeInfo):
*			Map of ip_addr -> NodeInfo structs. See below for more details
**/
type Server struct {
	pb.UnimplementedLogQueryServiceServer
	Memb_list        map[string]*pb.NodeInfo
	Host             string
	Port             string
	Incarnation      int32
	Version          int32
	ToggleSusFlag    bool
	ToggleSusVersion int32

	Mu sync.Mutex
}

/**
*	Helper that returns a NodeInfo struct.
*
**/

func (s *Server) createNodeInfo(host string, port string, state string, incarnation int32) *pb.NodeInfo {
	NodeInfo := &pb.NodeInfo{
		Hostname:    host,
		Port:        port,
		State:       state,
		Incarnation: incarnation,
		Timestamp:   timestamppb.New(time.Now()),
	}

	return NodeInfo
}

/**
*	Helper that returns a MembershipPush struct.
*
**/

func (s *Server) createMembershipPush(header string, host string, port string, incarnation int32, servers map[string]*pb.NodeInfo) pb.MembershipPush {

	mempush_req := pb.MembershipPush{
		Header:           header,
		Servers:          servers,
		SourceHost:       host,
		SourcePort:       port,
		Incarnation:      incarnation,
		ToggleSusFlag:    s.ToggleSusFlag,
		ToggleSusVersion: s.ToggleSusVersion,
	}
	return mempush_req
}

/**
*	Helper that returns a MembershipAck struct.
*
**/

func (s *Server) createMembershipAck(header string, servers map[string]*pb.NodeInfo, host string, port string, incarnation int32) pb.MembershipAck {

	memack_req := pb.MembershipAck{
		Header:           header,
		Servers:          servers,
		Hostname:         host,
		Port:             port,
		Incarnation:      incarnation,
		ToggleSusFlag:    s.ToggleSusFlag,
		ToggleSusVersion: s.ToggleSusVersion,
	}

	return memack_req
}

// Implements the GetLogs rpc
// Executes the command in the GrepRequeset request field
func (s *Server) GetLogs(ctx context.Context, req *pb.GrepRequest) (*pb.GrepReply, error) {
	logrus.Infof("Command received: %s", req.GetReq())

	// Run shell Command
	cmd := exec.Command("bash", "-c", req.GetReq())
	output, err := cmd.CombinedOutput()

	// if there is an error, set is_success to false.
	if err != nil {
		logrus.Errorf("Command Failed with Error: %v", err)
		return &pb.GrepReply{Idx: req.GetIdx(), Err: err.Error(), GrepOutput: string(output)}, err
	}

	// Grab the number of matching lines
	split_out := strings.Split(string(output), "\n")
	match_count := int32(len(split_out) - 1)

	// return a GrepReply protobuf containing grep results
	return &pb.GrepReply{Idx: req.GetIdx(), GrepOutput: string(output), MatchCount: match_count}, nil
}

/**
*	Helper function which iterates through the current server's membership list,
*	and changes "pending" to "dead" if timeout > TIMEOUT_THRESHOLD
*
**/
func (s *Server) checkNodeTimeouts() {

	for k, v := range s.Memb_list {

		elapsed_sec := timestamppb.Now().GetSeconds() - v.Timestamp.GetSeconds()
		elapsed_nano := timestamppb.Now().GetNanos() - v.Timestamp.GetNanos()
		elapsed_t := elapsed_sec*1e9 + int64(elapsed_nano)

		// if we have exceeded the timeout, change pending to dead. Let dead propagate before removing
		if v.State == "pending" && elapsed_t > int64(TIMEOUT_THRESHOLD)*1e9 {
			v.State = "dead"
			v.Timestamp = timestamppb.Now()
		}
		// declare nodes that are suspected for very long to be dead. Let dead propagate before removing
		if v.State == "suspect" && elapsed_t > int64(SUSPECT_TIMEOUT)*1e9 {
			delete(s.Memb_list, k)
		}
		// delete nodes that have been dead for more than 1 second
		if v.State == "dead" && elapsed_t > int64(PENDING_TIMEOUT)*1e9 {
			delete(s.Memb_list, k)
		}
	}
}

/**
*	Unforuantely, golang doesnt support function overloads. This function is meant to take in fields
*	and try to update them as per the logic as mentioned in the slides and above,
*	instead of just the entire map
*
*	Params:
*		host(string): host that we are updating in the map
*		port(string): port we wish to update to
*		state(string): the state we wish to update to in the map
*		incarnation(string): the incarnation we wish to update to in the map
*
**/
func (s *Server) updateMembershipInstance(host string, port string, state string, incarnation int32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	// logrus.Printf("Updating membership instance for host: %s to have state: %s", host, state)
	_, ok := s.Memb_list[host]
	// if this node does not exist, we will create the instance in the server Memb_list
	if !ok {
		s.Memb_list[host] = s.createNodeInfo(host, port, state, incarnation)
	} else if incarnation < s.Memb_list[host].Incarnation {
		// throw away old incarnation info
		return
	} else if incarnation > s.Memb_list[host].Incarnation {

		// override everything if incarnation is larger.
		s.Memb_list[host] = s.createNodeInfo(host, port, state, incarnation)
		s.Memb_list[host].Timestamp = timestamppb.Now()
	} else if state == "alive" || state == "dead" {

		// We ignore pendings here, because they dont provide definite info
		// We also ignore suspects, since we have clean up node
		logrus.Debugf("Mark host %s as state %s\n", host, state)
		s.Memb_list[host].State = state

		// update timestamp
		s.Memb_list[host].Timestamp = timestamppb.Now()
	} else if state == "suspect" && s.Memb_list[host].State == "alive" {

		logrus.Debugf("Mark host %s as state %s\n", host, state)
		s.Memb_list[host].State = state

		// update timestamp
		s.Memb_list[host].Timestamp = timestamppb.Now()
	}
}

/**
*	Server sends a request to introducer to join the network
*
**/
func (s *Server) joinGroup() error {
	conn, err := net.Dial("tcp", INTRODUCER_HOST+":"+INTRODUCER_PORT)
	if err != nil {
		logrus.Warnf("Introducer not alive")
		return err
	}

	defer conn.Close()
	join_req := s.createMembershipPush("join_req", s.Host, s.Port, s.Incarnation, s.Memb_list)

	marshalSendProtoTCP(&join_req, conn)

	logrus.Debugf("Join request sent")

	// read from tcp
	buffer := make([]byte, maxMsgSize)
	n, err := conn.Read(buffer)
	if err != nil {
		logrus.Errorf("Failed to receive ack on join: %v", err)
		return err
	}

	// deserialize response
	var join_ack pb.MembershipAck
	err = proto.Unmarshal(buffer[:n], &join_ack)
	if err != nil {
		logrus.Errorf("Failed to unmarshal ack on join: %v", err)
		return err
	}

	// update membership list + suspicion
	s.Mu.Lock()
	s.Memb_list = join_ack.Servers
	s.Mu.Unlock()

	logrus.Debugf("Successfully joined the system. Current membership list:")
	for nodeID, nodeInfo := range s.Memb_list {
		logrus.Debugf("%s: %v", nodeID, nodeInfo)
	}

	return nil
}

/**
*	Server sends message to introducer node to leave the network
*
**/
func (s *Server) leaveGroup() error {

	conn, err := net.Dial("tcp", INTRODUCER_HOST+":"+INTRODUCER_PORT)
	if err != nil {
		logrus.Warnf("Introducer not alive")
		return err
	}

	defer conn.Close()

	leave_req := s.createMembershipPush("leave_req", s.Host, s.Port, s.Incarnation, s.Memb_list)

	marshalSendProtoTCP(&leave_req, conn)

	logrus.Printf("Leave request sent")
	return nil
}
