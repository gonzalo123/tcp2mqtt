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
