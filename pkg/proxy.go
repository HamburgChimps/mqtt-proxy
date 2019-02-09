package mqtt

import (
	// "crypto/tls"
	"net"
	// "os"
	// "io"
	"strconv"
	// "sync"
	// "sync/atomic"
	"github.com/eclipse/paho.mqtt.golang/packets"
	uuid "github.com/satori/go.uuid"

	log "github.com/sirupsen/logrus"
)

type ConnectRequest struct {
	ID string
}

type ConnectionManager struct {
	clusters []*BrokerCluster
}

func (cm ConnectionManager) RegisterCluster(ID string, cluster *BrokerCluster) {

}

func (cm ConnectionManager) UnregisterCluster(ID string) {

}

func HandleConnection(clientSide net.Conn, cm *ClusterManager) {
	ID := uuid.NewV4().String()
	defer clientSide.Close()
	defer log.Infoln(ID, "Connection closed")

	cp, err := packets.ReadPacket(clientSide)
	if err != nil {
		return
	}

	p, ok := cp.(*packets.ConnectPacket)
	if ok {
		log.Infoln(ID, p)
		//
		cluster := cm.Get("/password")
		broker := cluster.Get(ID)
		if broker == nil {
			log.Errorln(ID, "No broker found")
			return
		}
		s := NewSession(ID)
		brokerSide, err := DialBroker(ID, broker)
		if err != nil {
			return
		}
		s.Start(p, clientSide, brokerSide)
	}
}

func HandleRequest() {

}

func DialBroker(ID string, broker *Broker) (net.Conn, error) {
	log.Infoln(broker)
	addr := broker.Address + ":" + strconv.Itoa(broker.Port)
	log.Println(ID, "Dialing", addr)
	brokerSide, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorln(ID, "Dial failed :", addr, err)
		return nil, err
	}
	return brokerSide, nil
}
