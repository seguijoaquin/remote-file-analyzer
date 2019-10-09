# Remote File Analyzer

Distributed app responsible for scanning remote hosts via FTP protocol and listing their directories and filesizes, inside a Docker Compose network.

### How to run

To run the app and one FTP server, run:
```
$ docker-compose up
```

To run a command-line interface and send tasks to the app, run:
```
$ go run launcher/launcher.go
```

### FTP Server image

The server image used was taken from [here](https://github.com/stilliard/docker-pure-ftpd)
