package postgres

#ConfigSpec: {
	serverConfig?: {
		[string]: _
	}

	hbaConfig?: {
		[string]: _
	}

	auth: {
		[string]: _
	}
	// Allow user to extend anything
	...
}