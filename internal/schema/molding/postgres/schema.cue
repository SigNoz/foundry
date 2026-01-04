package postgres

#ConfigSpec: {
	serverConfig?: {
		[string]: _
	}

	hbaConfig?: {
		[string]: _
	}

	authConfig : {
		[string]: _
	}
	// Allow user to extend anything
	...
}