package main

import (
	pb "MP2/protos"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/**
*	Updates the membership list with the logic given by cs425 lecture.
*	1. Check incarnation number. Higher incarnation always overrides lower incarnation
*   2. Suspect > Alive
*	3. Need a mechanism to check that the node is dead
*
*	Doesn't change anything when receives a "pending" status, unless its a new node
*	(relatively undefined beh)
*
*	Params:
*		incoming_server: membership list from the incoming server map[string]*pb.NodeInfo
**/
func (s *Server) updateMembershipList(incoming_server map[string]*pb.NodeInfo) {

	for k, v := range incoming_server {

		_, ok := s.Memb_list[k]

		// if we have not discovered this node before, just copy the incoming server, set to pending?
		if !ok {
			logrus.Debugf("Incoming membership list has a new row. Only add this row if it is valid.")

			if !s.validNodeEntry(v) {
				continue
			}
			s.Memb_list[k] = v
		}

		logrus.Debugf("%s has state: %s", v.Hostname, v.State)

		// if the update is of our own host and we are being suspected, we will increment incarnation
		// and propagte that we are alive
		if v.Hostname == s.Host && v.State == "suspect" {
			logrus.Infof("This host was suspected to be dead. Incrementing s.Incarnation...")
			s.Incarnation = max(v.Incarnation+1, s.Incarnation)
			s.Memb_list[s.Host].Incarnation = s.Incarnation
			s.Memb_list[s.Host].State = "alive"
		}

		// if we have, update using logic and move
		incoming_incarnation := v.Incarnation
		curr_incarnation := s.Memb_list[k].Incarnation

		// if we are receiving old incarnation info or old timestamp info, throw away the update
		if incoming_incarnation < curr_incarnation || s.Memb_list[k].Timestamp.AsTime().After(v.Timestamp.AsTime()) {
			continue
		} else if v.State == "suspect" {
			// if incoming state is suspect, and if we are already dead, do not update to suspect again.
			if s.Memb_list[k].State == "dead" {
				continue
			}

			s.Memb_list[k] = v

			// Take the earliest suspicion as the time to set if incumbent is sus
			if s.Memb_list[k].State == "suspect" {
				if s.Memb_list[k].Timestamp.AsTime().After(v.Timestamp.AsTime()) {
					s.Memb_list[k].Timestamp = v.Timestamp
				}
			}

		} else if v.State == "dead" {
			logrus.Debugf("%s declared %s \n", v.State, v.Hostname)
			s.Memb_list[k] = v

			if s.Memb_list[k].State == "dead" {
				if s.Memb_list[k].Timestamp.AsTime().After(v.Timestamp.AsTime()) {
					s.Memb_list[k].Timestamp = v.Timestamp
				}
			} else {
				// update time
				s.Memb_list[k].Timestamp = timestamppb.Now()
			}
		}
	}

}

/**
*	Implements the handshake protocol
*	This will receive a membership push, and return a membership ack.
*	The membership ack will contain:
*		Hostname
*		Port
*		Incarnation
*		MembershipList of reporting node
*
**/

func (s *Server) Handshake(conn *net.UDPConn, addr *net.UDPAddr, msg *pb.MembershipPush) error {

	// update the rest of the nodes from the membership list that is incoming, and also affirm this is alive
	s.Mu.Lock()
	s.updateMembershipList(msg.Servers)
	s.Mu.Unlock()

	// update sus if there is any change
	if msg.ToggleSusVersion > s.ToggleSusVersion {
		s.ToggleSusFlag = msg.ToggleSusFlag
		s.ToggleSusVersion = msg.ToggleSusVersion
	}

	// Immediately send a pong back.
	membership_ack := s.createMembershipAck("pong", s.Memb_list, s.Host, s.Port, s.Incarnation)

	// cant use helper, for udp unfortunately.
	data, err := proto.Marshal(&membership_ack)
	if err != nil {
		logrus.Errorf("Failed to marshal proto with err %v", err)
		return err
	}

	// send via udp
	_, err = conn.WriteToUDP(data, addr)
	if err != nil {
		logrus.Errorf("Failed to send Pong to %s:%s: %v", msg.SourceHost, msg.SourcePort, err)
		return err
	}

	return nil
}

func (s *Server) startUDPServer() {
	addr, err := net.ResolveUDPAddr("udp", ":"+s.Port)
	if err != nil {
		logrus.Errorf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logrus.Errorf("Failed to listen on UDP port: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, maxMsgSize)
	logrus.Infof("UDP server listening on %s:%s", s.Host, s.Port)

	for {
		// listen for incoming UDP messages
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			logrus.Errorf("Error reading from UDP: %v", err)
			continue
		}

		// umarshal the received message
		var msg pb.MembershipPush
		err = proto.Unmarshal(buffer[:n], &msg)
		if err != nil {
			logrus.Errorf("Failed to unmarshal message: %v", err)
			continue
		}

		// handle the received ping
		s.Handshake(conn, clientAddr, &msg)
	}
}

/** Distributes, streams and randomly sends pings to at least 4 other nodes using SWIM protocol in
*   parallel, which shall contain piggyback of suspicion.
*	Calls MembershipPush
*	This func is called by server because the server will routinely send out these
*	SWIM pings.
*	hostnames (string[]): pointer to an array of hostnames
 */
func (s *Server) StreamMembershipPing() (err error) {

	// define time ticker for streaming
	ticker := time.NewTicker(time.Duration(PING_PERIOD) * time.Second)
	defer ticker.Stop()

	// Handshake/Gossip -> ping/push
	for {
		select {
		case <-ticker.C:
			s.Mu.Lock()
			MembershipPing := s.createMembershipPush("ping", s.Host, s.Port, s.Incarnation, s.Memb_list)

			// clean out the pendings to dead if beyond timeout and remove dead nodes
			s.checkNodeTimeouts()
			s.Mu.Unlock()

			// Send requests to all servers concurrently
			var wg sync.WaitGroup

			// write membership list to array for indexing
			s.Mu.Lock()
			var membership_list []string
			for hostname := range s.Memb_list {
				membership_list = append(membership_list, hostname)
			}
			s.Mu.Unlock()

			// randomly choose host without replacement
			permutation := rand.Perm(len(s.Memb_list))[:min(NUM_RANDOM_NODES, len(membership_list))]

			for i := 0; i < min(NUM_RANDOM_NODES, len(membership_list)); i += 1 {

				var selected_host = membership_list[permutation[i]]

				wg.Add(1)
				go func(destination_host string) {
					defer wg.Done()

					// send pint-ack via UDP
					addr, err := net.ResolveUDPAddr("udp", destination_host+":"+NETWORK_PORT)
					if err != nil {
						panic(err)
					}
					conn, err := net.DialUDP("udp", nil, addr)
					if err != nil {
						logrus.Errorf("Failed to create UDP connection: %v", err)
						return
					}
					defer conn.Close()

					// cant use helper, for udp unfortunately.
					// marshal the MembershipPing (basically just serialize it)
					data, err := proto.Marshal(&MembershipPing)
					if err != nil {
						logrus.Errorf("Failed to marshal MembershipPing: %v", err)
						return
					}
					// send ping
					_, err = conn.Write(data)
					if err != nil {
						logrus.Errorf("Failed to send MembershipPing to %s: %v", destination_host, err)
						return
					}

					s.updateMembershipInstance(destination_host, NETWORK_PORT, "pending", s.Memb_list[destination_host].Incarnation)

					buffer := make([]byte, maxMsgSize)
					conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(CONN_TIMEOUT)))

					n, addr, err := conn.ReadFromUDP(buffer)
					if err != nil {
						logrus.Errorf("Timeout or error receiving Pong from %s: %v", destination_host, err)
						if s.ToggleSusFlag {
							// immediately print when a node becomes suspected
							logrus.Infof("Susepcting node %s to be dead", destination_host)
							s.updateMembershipInstance(destination_host, NETWORK_PORT, "suspect", s.Memb_list[destination_host].Incarnation)
						} else {
							s.updateMembershipInstance(destination_host, NETWORK_PORT, "dead", s.Memb_list[destination_host].Incarnation)
						}
						return
					}

					// unmarshal the response
					var ack pb.MembershipAck
					err = proto.Unmarshal(buffer[:n], &ack)
					if err != nil {
						logrus.Errorf("Failed to unmarshal MembershipAck: %v", err)
						if s.ToggleSusFlag {
							s.updateMembershipInstance(destination_host, NETWORK_PORT, "suspect", s.Memb_list[destination_host].Incarnation)
						} else {
							s.updateMembershipInstance(destination_host, NETWORK_PORT, "dead", s.Memb_list[destination_host].Incarnation)
						}
						return
					}

					// update membership list
					logrus.Debugf("Received %s from %s", ack.GetHeader(), ack.GetHostname())
					s.updateMembershipInstance(ack.GetHostname(), ack.GetPort(), "alive", ack.GetIncarnation())

				}(selected_host)

			}

			// await barrier
			wg.Wait()

			logrus.Debugf("===============CURR STATE====================\n")
			names := make([]string, 0, len(s.Memb_list))
			for k := range s.Memb_list {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				logrus.Debugf("Host: %s, State: %s, Incarnation: %d\n", k, s.Memb_list[k].GetState(), s.Memb_list[k].GetIncarnation())
			}
			logrus.Debugf("=============================================\n")
		}
	}
}
