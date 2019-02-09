package mqtt

import (
	// "contect"
	"github.com/HuKeping/rbtree"
	log "github.com/sirupsen/logrus"
	"sync"
	// "time"
	// envoyAuth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	// envoyCore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

// const timeout = 1000 * time.Millisecond

type ClusterManager struct {
	sync.RWMutex
	clusters map[string]*BrokerCluster
}

func NewClusterManager() *ClusterManager {
	clusters := make(map[string]*BrokerCluster)
	clusters["/password"] = NewBrokerCluster("password")
	cm := &ClusterManager{
		clusters: clusters,
	}
	return cm
}

func (cm *ClusterManager) Get(mp string) *BrokerCluster {
	cm.RLock()
	defer cm.RUnlock()
	if cluster, ok := cm.clusters[mp]; ok {
		return cluster
	}
	return nil
}

type ClusterRequest struct {
	Mountpoint string
	Out        chan *BrokerCluster
}

// BrokerCluster is a round robin MQTT load balancer
type BrokerCluster struct {
	sync.RWMutex
	current    *rbtree.Item
	hosts      *rbtree.Rbtree
	Mountpoint string

	// TlsContext *envoyAuth.UpstreamTlsContext
}

// NewBrokerCluster ...
func NewBrokerCluster(mp string) *BrokerCluster {
	hosts := rbtree.New()
	hosts.Insert(&Broker{Nr: 0, Address: "0.0.0.0", Port: 1883})
	hosts.Insert(&Broker{Nr: 1, Address: "iot.eclipse.org", Port: 1883})
	// current := hosts.Min()
	r := &BrokerCluster{
		Mountpoint: mp,
		hosts:      hosts,
	}
	log.Infoln("Cluster", r)
	// go r.loop()
	return r
}

func (bc *BrokerCluster) Balance() *Broker {
	bc.RLock()
	defer bc.RUnlock()

	if bc.hosts.Len() == 0 {
		return nil
	}
	if bc.current == nil {
		next := bc.hosts.Min()
		bc.current = &next
	}

	current := *bc.current
	mBroker := bc.hosts.Max().(*Broker)
	cBroker := current.(*Broker)

	if cBroker.Nr == mBroker.Nr {
		next := bc.hosts.Min()
		bc.current = &next
	} else {
		bc.hosts.Ascend(current, func(i rbtree.Item) bool {
			if i.(*Broker).Nr > cBroker.Nr {
				bc.current = &i
				return false
			}
			return true
		})
	}

	return cBroker
}

func (bc *BrokerCluster) Get(nr uint) *Broker {
	bc.RLock()
	defer bc.RUnlock()
	found := bc.hosts.Get(&Broker{Nr: nr})
	if found != nil {
		return found.(*Broker)
	}
	return nil
}

func (bc *BrokerCluster) Add(broker *Broker) {
	bc.Lock()
	defer bc.Unlock()
	bc.hosts.Insert(broker)
}

func (bc *BrokerCluster) Delete(nr uint) {
	bc.Lock()
	defer bc.Unlock()
	bc.hosts.Delete(&Broker{Nr: nr})
}

type Broker struct {
	Nr      uint
	Address string
	Port    int
}

func (b Broker) Less(than rbtree.Item) bool {
	return b.Nr < than.(*Broker).Nr
}
