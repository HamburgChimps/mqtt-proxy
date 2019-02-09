package mqtt

import (
	// "crypto/tls"
	"net"
	// "os"
	// "io"
	"strconv"
	"sync"
	// "sync/atomic"
	"github.com/HuKeping/rbtree"
	"github.com/eclipse/paho.mqtt.golang/packets"

	uuid "github.com/satori/go.uuid"

	log "github.com/sirupsen/logrus"
)

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
		log.Infoln(ID, p)
		//
		route := r.Route(&cp)
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

type Matcher interface {
	Match(*packets.ControlPacket) bool
}

type MatcherFunc func(cp *packets.ControlPacket) bool

func (fn MatcherFunc) Match(cp *packets.ControlPacket) bool {
	return fn(cp)
}

type RouteConfig struct {
	Matcher
	Nr         uint
	Mountpoint string
}

func (r RouteConfig) Less(than rbtree.Item) bool {
	return r.Nr < than.(*RouteConfig).Nr
}

type Router struct {
	sync.RWMutex
	routes *rbtree.Rbtree
}

func (r *Router) Route(cp *packets.ControlPacket) (Mountpoint string) {
	r.RLock()
	defer r.RUnlock()
	mountpoint := "DENIED"
	r.routes.Ascend(r.routes.Min(), func(i rbtree.Item) bool {
		rc := i.(*RouteConfig)
		if rc.Match(cp) {
			mountpoint = rc.Mountpoint
			return false
		}
		return true
	})
	return mountpoint
}

func NewRouter() *Router {
	routes := rbtree.New()
	routes.Insert(&RouteConfig{
		Nr:         0,
		Mountpoint: "/password",
		Matcher:    &ClientIDMatch{},
	})
	return &Router{
		routes: routes,
	}
}

type ClientIDMatch struct {
	clientID string
}

func (c *ClientIDMatch) Match(cp *packets.ControlPacket) bool {
	return true
}
