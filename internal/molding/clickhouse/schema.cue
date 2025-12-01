// internal/molding/clickhouse/schema.cue
package clickhouse

#Config: {	
	// File: config.yaml (main ClickHouse configuration)
	serverConfig?: {
		[string]: _ 
	}
	
	// File: users.yaml (users, profiles, quotas)
	usersConfig?: {
		[string]: _
	}
	
    customFunctionConfig? :{
        [string]: _
    }

	//Additional config files in config.d/
	config_d?: {
		[string]: {  // filename -> content
			[string]: _ 
		}
	}
	
	// Optional: users.d/ directory
	users_d?: {
		[string]: {  // filename -> content
			[string]: _ 
		}
	}
}