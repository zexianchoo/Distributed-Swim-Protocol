package main

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"os"
	"strings"
	"fmt"
	"strconv"
)

/**
*	get min value
**/
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

/**
*	Reads the file given by the hostnames_filepath, and subsequently saves it to a string array.
*
 */
func getFileList(hostnames *[]string, hostnames_filepath *string) {
	dat, err := os.ReadFile(*hostnames_filepath)
	if err != nil {
		log.Fatalf("[GetFileList] error occurred: %v\n", err)
	}
	// save to ip array
	ip_string := string(dat)
	scanner := bufio.NewScanner(strings.NewReader(ip_string))
	for scanner.Scan() {
		// append port number:
		*hostnames = append(*hostnames, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Scanner error occurred: %v\n", err)
	}
}

/**
*	Marshal data and send over the protobuf.
*
**/
func marshalSendProtoTCP(msg proto.Message, conn net.Conn) error {

	// serialize proto
	data, err := proto.Marshal(msg)
	if err != nil {
		logrus.Errorf("Failed to marshal response: %v", err)
		return err
	}

	// send over tcp
	_, err = conn.Write(data)
	if err != nil {
		logrus.Errorf("Failed to send to %s: %v", conn.RemoteAddr().(*net.TCPAddr).String(), err)
		return err
	}
	return nil
}

/**
*	Gets version from local persistent storage.
*   This version number will be incremented as immediately after retrieval.
*
**/
func getVersion() int32 {

	// if path doesnt exist, initialize with 0
	_, err := os.Stat(VERSION_PATH)
	if err != nil {
		f, err := os.Create(VERSION_PATH)
		if err != nil {
			logrus.Fatalf("Couldnt Create version file!\n")
		}
		defer f.Close()
		_, err = f.WriteString("0")
		if err != nil {
			logrus.Fatalf("Couldnt Write to version file!\n")
		}
		return 0
	} else {
		f, err := os.Open(VERSION_PATH)
		if err != nil {
			logrus.Fatalf("Couldnt Open incarnation file!\n")
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Scan()
		version_num, _ := strconv.Atoi(scanner.Text())

		// increment for a new version
		version_num += 1

		// overwrite the file with the new version_num:
		f2, err := os.Create(VERSION_PATH)
		if err != nil {
			logrus.Fatalf("Couldnt Create version file!\n")
		}
		defer f2.Close()
		_, err = f2.WriteString(fmt.Sprintf("%d\n", version_num))
		if err != nil {
			logrus.Fatalf("Couldnt Write to version file!\n")
		}

		return int32(version_num)
	}
}
