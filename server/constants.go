package main

import (
	"github.com/sirupsen/logrus"
	"sync"
)

var (

	// timeouts
	NUM_RANDOM_NODES  = 4
	CONN_TIMEOUT      = 2
	PING_PERIOD       = 2
	SUSPECT_TIMEOUT   = 10
	PENDING_TIMEOUT   = 1
	TIMEOUT_THRESHOLD = 15

	// toggle suspicion
	USE_SUSPECT_FLAG = true

	// logging
	LOG_LEVEL = logrus.InfoLevel

	// networking
	maxMsgSize      = 1024 * 1024 * 100
	INTRODUCER_HOST = "fa24-cs425-2501.cs.illinois.edu" // change the introducer host here
	NETWORK_PORT    = "5138"
	INTRODUCER_PORT = "5139"

	// misc
	Wg           sync.WaitGroup
	VERSION_PATH = "./version_num"
)
