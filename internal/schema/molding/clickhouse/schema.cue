package clickhouse

// Commons helpers
#LoggerLevel:          "none" | "fatal" | "critical" | "error" | "warning" | "notice" | "information" | "debug" | "trace"
#LoggerFormattingType: "json" | "pattern" | "console"
#LoadBalancing:        "random" |
	"nearest_hostname" |
	"hostname_levenshtein_distance" |
	"in_order" |
	"first_or_random"

#ConfigSpec: {
	// Whether to use standalone ClickHouse Keeper
	// If true: keeper runs as separate instance, zookeeper config is added to clickhouse-server
	// If false: keeper runs embedded in clickhouse-server, no zookeeper config
	keeper?: {
		enabled: *false | bool
		if enabled == true{
			replicas: int
			version?: string
			config?: {
				[string]: _
			}
		}
	}
	// File: config.yaml (main ClickHouse configuration)
	serverConfig?: {
		[string]: _
	}

	// File: users.yaml (users, profiles, quotas)
	usersConfig?: {
		[string]: _
	}

	customFunctionConfig?: {
		[string]: _
	}

	//Additional config files in config.d/
	config_d?: {
		[string]: {// filename -> content
			[string]: _
		}
	}

	// Optional: users.d/ directory
	users_d?: {
		[string]: {// filename -> content
			[string]: _
		}
	}
}

#KeeperConfigSpec: {
	[string]: _
}