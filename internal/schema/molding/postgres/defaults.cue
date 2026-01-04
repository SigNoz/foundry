package postgres

#BaseConfig: #ConfigSpec & {
	serverConfig: {
		port:            *5432 | int
		max_connections: *100 | int
		...
	}

	hbaConfig: {
		...
	}

	auth: {
		postgres_password: string & =~"[a-z]+" & =~"[A-Z]+" & =~"[0-9]+" & =~"[!@#$%^&*]+"
		postgres_db: *"signoz" | string
		postgres_user: *"signoz" | string
	}
	...
}

#BaseConfig