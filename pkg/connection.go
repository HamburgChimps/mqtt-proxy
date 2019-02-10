package mqtt

import (
	"github.com/HuKeping/rbtree"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"net"
	"strconv"
	"strings"
	"sync"

	uuid "github.com/satori/go.uuid"

	log "github.com/sirupsen/logrus"
)

type Router struct {
	sync.RWMutex
	routes *rbtree.Rbtree
}

func (r *Router) Route(cp packets.ControlPacket) (ClusterID string) {
	r.RLock()
	defer r.RUnlock()
	res := "DENIED"
	r.routes.Ascend(r.routes.Min(), func(i rbtree.Item) bool {
		rc := i.(*RouteConfig)
		if rc.Match(cp) {
			res = rc.ClusterID
			return false
		}
		return true
	})
	return res
}

type Matcher interface {
	Match(packets.ControlPacket) bool
}

type RouteConfig struct {
	Matcher
	ID        uint
	ClusterID string
}

func (r RouteConfig) Less(than rbtree.Item) bool {
	return r.ID < than.(*RouteConfig).ID
}

type ClientIDMatch struct {
	clientID string
}

func (c *ClientIDMatch) Match(cp packets.ControlPacket) bool {
	p, ok := cp.(*packets.ConnectPacket)
	if ok {
		return strings.HasPrefix(p.ClientIdentifier, c.clientID)
	}
	return false
}

func HandleConnection(clientSide net.Conn, r *Router, cm *ClusterManager) {
	ID := uuid.NewV4().String()
	defer clientSide.Close()
	defer log.Infoln(ID, "Connection closed")

	cp, err := packets.ReadPacket(clientSide)
	if err != nil {
		return
	}

	p, ok := cp.(*packets.ConnectPacket)
	if ok {
		log.Infoln(ID, "Handler", "INGRESS", p)
		//
		route := r.Route(cp)
		cluster := cm.Get(route)
		if cluster == nil {
			log.Errorln(ID, "No cluster found")
			return
		}
		broker := cluster.Balance()
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

func DialBroker(ID string, broker *Broker) (net.Conn, error) {
	addr := broker.Address + ":" + strconv.Itoa(broker.Port)
	log.Println(ID, "Dialing", addr)
	brokerSide, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorln(ID, "Dial failed :", addr, err)
		return nil, err
	}
	return brokerSide, nil
}

func NewRouterDemo() *Router {
	routes := rbtree.New()
	routes.Insert(&RouteConfig{
		ID:        0,
		ClusterID: "/mosqsub",
		Matcher: &ClientIDMatch{
			clientID: "mosqsub",
		},
	})
	routes.Insert(&RouteConfig{
		ID:        1,
		ClusterID: "/hivemq",
		Matcher: &ClientIDMatch{
			clientID: "hivemq",
		},
	})
	return &Router{
		routes: routes,
	}
}
