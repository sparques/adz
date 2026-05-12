Add:
	- user start up script (~/.strap.adz)
	- working directory start up script ./strap.adz 


exec -input $| cat -r

<{} means use contents of {} as stdin
exec <{}

> means use stdout as output of command
exec >


Maybe a pipe command that temporarily redirects pipes?

pipe 