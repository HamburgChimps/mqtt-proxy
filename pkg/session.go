package mqtt

import (
	// "crypto/tls"
	"net"
	// "os"
	"io"
	// "strconv"
	"sync"
	// "sync/atomic"
	"github.com/eclipse/paho.mqtt.golang/packets"

	log "github.com/sirupsen/logrus"
)

type Session struct {
	ID string
	wg sync.WaitGroup
	//deviceName string
	Username         string
	ClientIdentifier string
	clientSide       net.Conn
	brokerSide       net.Conn
	closed           bool
}

func NewSession(ID string) *Session {
	var session Session
	session.ID = ID
	return &session
}

func (session *Session) Start(cp *packets.ConnectPacket, clientSide net.Conn, brokerSide net.Conn) {
	session.clientSide = clientSide
	session.brokerSide = brokerSide
	defer session.clientSide.Close()
	defer session.brokerSide.Close()
	session.wg.Add(2)

	err := cp.Write(session.brokerSide)
	if err != nil {
		log.Errorln(err)
		return
	}
	go session.forwardHalf("EGRESS", session.brokerSide, session.clientSide)
	go session.forwardHalf("INGRESS", session.clientSide, session.brokerSide)
	session.wg.Wait()

	// atomic.AddInt32(&globalSessionCount, -1)
	log.Println(session.ID, "Session", "Closed", clientSide.LocalAddr().String())
}

func (session *Session) ForwardMQTTPacket(way string, r net.Conn, w net.Conn) error {
	cp, err := packets.ReadPacket(r)
	if err != nil {
		if !session.closed {
			if err != io.EOF {
				log.Errorln(session.ID, "Session", way, "Error reading MQTT packet", err)
			}
		}
		return err
	}
	log.Println(session.ID, "Session", way, "-", cp.String())

	if err != nil {
		log.Println("Session", session.ID, way, "Forward MQTT packet", err)
		return err
	}

	err = cp.Write(w)
	if err != nil {
		log.Errorln(session.ID, way, "Session", "Error writing MQTT packet", err)
		return err
	}
	return nil
}

func (session *Session) forwardHalf(way string, c1 net.Conn, c2 net.Conn) {
	defer c1.Close()
	defer c2.Close()
	defer session.wg.Done()
	//io.Copy(c1, c2)

	for {
		// log.Println("Session", session.ID, way, "- Wait Packet", c1.RemoteAddr().String(), c2.RemoteAddr().String())
		err := session.ForwardMQTTPacket(way, c1, c2)
		if err != nil {
			session.closed = true
			break
		}
	}
}
