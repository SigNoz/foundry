package zookeeper

// Allowed duration-like ints for ZK configs
_durationInt: int & >=0

// Server entry definition
#ServerSpec: {
    id:          int & >0
    host:        string
    peerPort:    int & >0
    electionPort: int & >0
}

// Full configuration schema
#ConfigSpec: {
    tickTime:          int & >0
    dataDir:           string
    clientPort:        int & >0
    initLimit:         _durationInt
    syncLimit:         _durationInt
    maxClientCnxns:    int & >=0

    autopurge: {
        snapRetainCount: int & >=1
        purgeInterval:   int & >=0
    }

    // Array of server definitions
    servers: [...#ServerSpec]

    // Extensions allowed
    ...
}