package zookeeper

#BaseConfig: #ConfigSpec & {
	tickTime:       *2000 | int
	dataDir:        *"/var/lib/zookeeper/data" | int
	clientPort:     *2181 | int
	initLimit:      *10 | int
	syncLimit:      *5 | int
	maxClientCnxns: *60 | int
	autopurge: {
		snapRetainCount: *3 | int
		purgeInterval:   *1 | int
	}

	// Server list (1 node default)
	servers: [
		{
			id:           *1 | int
			host:         *"localhost" | int
			peerPort:     *2888 | int
			electionPort: *3888 | int
		},
	]

	// Allow user to extend anything
	...
}

#BaseConfig