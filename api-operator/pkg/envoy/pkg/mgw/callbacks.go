package mgw

import (
	"context"
	"log"
	"sync"

	mgwconfig "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/configs"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type Callbacks struct {
	Signal   chan struct{}
	Debug    bool
	Fetches  int
	Requests int
	mu       sync.Mutex
}

func (cb *Callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.Printf("server callbacks fetches=%d requests=%d\n", cb.Fetches, cb.Requests)
}
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	if cb.Debug {
		log.Printf("stream %d open for %s\n", id, typ)
	}
	return nil
}
func (cb *Callbacks) OnStreamClosed(id int64) {
	if cb.Debug {
		log.Printf("stream %d closed\n", id)
	}
}
func (cb *Callbacks) OnStreamRequest(id int64, drq *discovery.DiscoveryRequest) error {
	log.Println("stream request received")
	// s, _ := json.MarshalIndent(drq, "", "\t")
	// log.Println("request : " + string(s))
	// log.Println("stream request received ---- ")

	nodeId := drq.Node.GetId()
	conf, _ := mgwconfig.ReadConfigs()
	updateEnvoyForSpecificNode(conf.Apis.Location, nodeId)

	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Requests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}
func (cb *Callbacks) OnStreamResponse(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	log.Println("stream response received")

}
func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	log.Println("fetch request received")
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Fetches++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}
func (cb *Callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	log.Println("fetch response received")
}
