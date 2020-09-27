## Transforming TCP sockets to MQTT with Go

In the last post we've created one proxy to upgrade one legacy application that sends raw TCP sockets to a HTTP server without changing the original application.

Now we're going to do the same but instead sending HTTP request we're going to connect to a MQTT broker. Probably try to change the legacy application to connect to a MQTT broker can be a nightmare but with with this approach is pretty straightforward.

The idea is the same. We're going to send our TCP sockets to localhost. Then we're going to build a go client that reads the TCP sockets and send the information to the MQTT broker.

We're going to use Mosquitto as MQTT broker. We can set up easily with docker:

```yaml
version: '2'

services:
  mosquitto:
    image: eclipse-mosquitto
    hostname: mosquitto
    container_name: mosquitto
    build:
      context: .docker/mosquitto
      dockerfile: Dockerfile
    expose:
      - "1883"
      - "9001"
    ports:
      - "1883:1883"
      - "9001:9001"
```

We can also set up our Mosquitto server with user and password with mosquitto.conf and users.txt. For this example we're going to use the credentials: username:password

```
username:$6$6jOr4vVqaKxisTls$4KVYh8NBZdP+z4S/YbuoSHKlJ+5F1DxiE7XtWWXVHQ+7PlCI+b6LhqSbj8lL45HnGlo4D5t0AVFYrYGjb5lTxg==
```

Our Go program is very similar than the http version:

```go
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	port, closeConnection, topic, broker := parseFlags()
	openSocket(*port, *closeConnection, *topic, *broker, onMessage)
}

func openSocket(port string, closeConnection bool, topic string, broker string, onMessage func(url string, topic string, buffer string)) {
	PORT := "localhost:" + port
	l, err := net.Listen("tcp4", PORT)
	log.Printf("Serving %s\n", l.Addr().String())
	if err != nil {
		log.Fatalln(err)
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handleConnection(c, closeConnection, topic, broker, onMessage)
	}
}

func createClientOptions(url string) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetUsername(os.Getenv("MQTT_USERNAME"))
	opts.SetPassword(os.Getenv("MQTT_PASSWORD"))
	return opts
}

func connect(url string) mqtt.Client {
	opts := createClientOptions(url)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func onMessage(url string, topic string, buffer string) {
	client := connect(url)
	client.Publish(topic, 0, false, buffer)
}

func parseFlags() (*string, *bool, *string, *string) {
	port := flag.String("port", "7777", "port number")
	closeConnection := flag.Bool("close", true, "Close connection")
	topic := flag.String("topic", "topic", "mqtt topic")
	broker := flag.String("broker", "tcp://localhost:1883", "mqtt topic")
	flag.Parse()

	return port, closeConnection, topic, broker
}

func handleConnection(c net.Conn, closeConnection bool, topic string, broker string, onMessage func(url string, topic string, buffer string)) {
	log.Printf("Accepted connection from %s\n", c.RemoteAddr().String())
	for {
		ip, port, err := net.SplitHostPort(c.RemoteAddr().String())
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			log.Println(err)
		}

		message := map[string]interface{}{
			"body":   strings.TrimSpace(netData),
			"ipFrom": ip,
			"port":   port,
		}

		log.Printf("sending to topic %s message:%s\n", topic, message)
		bytesRepresentation, err := json.Marshal(message)
		if err != nil {
			log.Println(err)
		} else {
			onMessage(broker, topic, string(bytesRepresentation))
		}

		if closeConnection {
			c.Close()
			return
		}
	}
	c.Close()
}
```

And that's all. Our legacy application can now speak MQTT without problems

