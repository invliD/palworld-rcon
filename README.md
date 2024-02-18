# palworld-rcon
[![GitHub Build](https://github.com/invliD/palworld-rcon/workflows/build/badge.svg)](https://github.com/invliD/palworld-rcon/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/invliD/palworld-rcon)](https://goreportcard.com/report/github.com/invliD/palworld-rcon)

## Install
```shell
go get github.com/invliD/palworld-rcon
```

## Usage
```go
package main

import (
	"fmt"
	"log"

	palworldrcon "github.com/invliD/palworld-rcon"
)

func main() {
	client := palworldrcon.NewClient("127.0.0.1:25575", "password")

	info, err := client.Info()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Connected to server '%s' running version %s!\n", info.ServerName, info.Version)
}
```

## License
MIT License, see [LICENSE](LICENSE)
