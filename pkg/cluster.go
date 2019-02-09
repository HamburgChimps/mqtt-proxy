package mqtt

import (
	"github.com/HuKeping/rbtree"
	_ "github.com/sirupsen/logrus"
	"sync"
)

type ClusterManager struct {
	sync.RWMutex
	clusters map[string]*BrokerCluster
}

func NewClusterManager() *ClusterManager {
	clusters := make(map[string]*BrokerCluster)
	clusters["/hivemq"], clusters["/mosqsub"] = NewBrokerClusterDemo()
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

// BrokerCluster is a round robin MQTT load balancer
type BrokerCluster struct {
	sync.RWMutex
	current    *rbtree.Item
	hosts      *rbtree.Rbtree
	Mountpoint string

	// TlsContext *envoyAuth.UpstreamTlsContext
}

// NewBrokerClusterDemo ...
func NewBrokerClusterDemo() (*BrokerCluster, *BrokerCluster) {
	hosts1 := rbtree.New()
	hosts1.Insert(&Broker{Nr: 0, Address: "broker.hivemq.com", Port: 1883})
	bc1 := &BrokerCluster{
		Mountpoint: "/hivemq",
		hosts:      hosts1,
	}

	hosts2 := rbtree.New()
	hosts2.Insert(&Broker{Nr: 1, Address: "iot.eclipse.org", Port: 1883})
	bc2 := &BrokerCluster{
		Mountpoint: "/mosqsub",
		hosts:      hosts2,
	}
	return bc1, bc2
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
