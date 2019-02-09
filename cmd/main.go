package main

import (
	// "context"
	// "github.com/HamburgChimps/disqtt/internal/router"
	// "github.com/alsm/paho.mqtt.golang/paho"
	"github.com/HamburgChimps/mqtt-proxy/pkg"
	log "github.com/sirupsen/logrus"

	"net"
	"os"
	// "os/signal"
	// "syscall"
)

func main() {

	log.Printf("Connected to hello world\n")
	mqttListen()

}

func mqttAccept(l net.Listener) {
	// cluster := mqtt.NewBrokerCluster()
	cm := mqtt.NewClusterManager()
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		log.Infoln(cm)
		go mqtt.HandleConnection(conn, cm)
	}
}

func mqttListen() {
	// Listen for incoming connections.
	addr := "0.0.0.0:8989"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("mqtt: Error listening mqtt://"+addr, err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Println("mqtt: listening on mqtt://" + addr)

	mqttAccept(l)
}
