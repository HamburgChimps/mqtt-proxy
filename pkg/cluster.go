package mqtt

import (
	// "contect"
	log "github.com/sirupsen/logrus"
	// "sync"
	// "time"
	// envoyAuth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	// envoyCore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

// const timeout = 1000 * time.Millisecond

type ClusterManager struct {
	clusters   map[string]*BrokerCluster
	clusterReq chan interface{}
}

func NewClusterManager() *ClusterManager {
	clusters := make(map[string]*BrokerCluster)
	clusters["/password"] = NewBrokerCluster("password")
	cm := &ClusterManager{
		clusters:   clusters,
		clusterReq: make(chan interface{}),
	}
	go cm.loop()
	return cm
}

func (cm *ClusterManager) Get(mp string) *BrokerCluster {
	out := make(chan *BrokerCluster)
	cm.clusterReq <- &ClusterRequest{Mountpoint: mp, Out: out}
	resp := <-out
	return resp
}

func (cm *ClusterManager) loop() {
	for {
		select {
		case req := <-cm.clusterReq:
			switch req := req.(type) {
			case *ClusterRequest:
				if cluster, ok := cm.clusters[req.Mountpoint]; ok {
					req.Out <- cluster
				} else {
					req.Out <- nil
				}
			case string:
				delete(cm.clusters, req)
			}

		}
	}
}

type ClusterRequest struct {
	Mountpoint string
	Out        chan *BrokerCluster
}

// BrokerCluster is a round robin MQTT load balancer
type BrokerCluster struct {
	current int

	hosts      map[int]*Broker
	Mountpoint string
	brokerReq  chan interface{}

	// TlsContext *envoyAuth.UpstreamTlsContext
}

// NewBrokerCluster ...
func NewBrokerCluster(mp string) *BrokerCluster {
	hosts := make(map[int]*Broker)
	hosts[0] = &Broker{Address: "0.0.0.0", Port: 1883}
	hosts[1] = &Broker{Address: "iot.eclipse.org", Port: 1883}
	r := &BrokerCluster{
		Mountpoint: mp,
		current:    0,
		hosts:      hosts,
		brokerReq:  make(chan interface{}),
	}
	log.Infoln("Cluster", r)
	go r.loop()
	return r
}

type ReadRequest struct {
	ID       string
	brokerID string
	Out      chan *Broker
}

func (bc *BrokerCluster) Get(requestID string) *Broker {
	out := make(chan *Broker, 1)
	bc.brokerReq <- &ReadRequest{ID: requestID, Out: out}
	resp := <-out
	return resp
}

type AddRequest struct {
	ID       *string
	brokerID int
	Broker   *Broker
	Out      chan bool
}

func (bc *BrokerCluster) Add(requestID *string, broker *Broker) {

}

func (bc *BrokerCluster) loop() {
	for {
		select {
		case req := <-bc.brokerReq:
			switch req := req.(type) {
			case *ReadRequest:
				if len(bc.hosts) == 0 {
					req.Out <- nil
				}
				if bc.current >= len(bc.hosts) {
					bc.current = 0
				}
				res := bc.hosts[bc.current]
				bc.current++
				req.Out <- res
				log.Infoln(req.ID, "Found broker", res)
			case *AddRequest:
				bc.hosts[req.brokerID] = req.Broker
			case int:
				delete(bc.hosts, req)
			}
		}
	}
}

type Broker struct {
	Address string
	Port    int
}
