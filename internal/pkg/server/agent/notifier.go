package agent

import (
	"context"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/grpc-edge-inventory-proxy-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

// Notifier structure to send data back to the management cluster.
type Notifier struct {
	// Mutex for managing the internal structure.
	sync.Mutex
	// notifyPeriod contains the default period between notifications to the management cluster.
	notifyPeriod time.Duration
	// AssetAlive is a map of asset identifiers with the timestamp they sent the last check/ping.
	assetAlive map[string]int64
	// AssetIP is a map of the asset identifiers with its ip
	AssetIP map [string]string
	// AssetNewIP is a map of the assets identifiers whose ip has changed, this list will be sent to the system model to update the information
	AssetNewIP map [string]string
	// provider for the persistent operations.
	provider asset.Provider
	// mngtClient with the client that connects to the management cluster.
	mngtClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient
	// organizationID with the organization identifier
	organizationID string
	// edgeControllerID with de EIC identifier
	edgeControllerID string
}

func NewNotifier(notifyPeriod time.Duration, provider asset.Provider, mngtClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient,
	organizationID string, edgeControllerID string) *Notifier {
	return &Notifier{
		notifyPeriod: notifyPeriod,
		assetAlive: make(map[string]int64, 0),
		AssetIP: make (map[string]string, 0),
		AssetNewIP: make (map[string]string, 0),
		provider: provider,
		mngtClient: mngtClient,
		organizationID: organizationID,
		edgeControllerID: edgeControllerID,
	}
}

// AgentAlive registers that an agent is alive and its IP
func (n *Notifier) AgentAlive(assetID string, ip string) {
	n.Lock()
	defer n.Unlock()
	log.Debug().Str("assetID", assetID).Msg("asset is alive")
	n.assetAlive[assetID] = time.Now().Unix()

	// check IP
	oldIP, exists := n.AssetIP[assetID]
	if !exists {
		n.AssetNewIP[assetID] = ip
		n.AssetIP[assetID] = ip
	}else if  exists && ip != oldIP {
		n.AssetNewIP[assetID] = ip
	}


}

// LaunchNotifierLoop is intended to be launched as goroutine for periodically sending data back to the management cluster.
func (n *Notifier) LaunchNotifierLoop() {
	ticker := time.NewTicker(n.notifyPeriod)
	for range ticker.C {
		n.notifyManagementCluster()
	}
}

func (n * Notifier) sendAliveMessages() bool{
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	messages := &grpc_inventory_manager_go.AgentsAlive{
		OrganizationId: n.organizationID,
		EdgeControllerId: n.edgeControllerID,
		Agents: n.assetAlive,
		AgentsIp: n.AssetNewIP,
	}

	_, err := n.mngtClient.LogAgentAlive(ctx, messages)
	if err != nil{
		log.Warn().Str("trace", conversions.ToDerror(err).DebugReport()).Msg("cannot send alive messages to management cluster")
		return false
	}

	return true
}

// notifyManagementCluster compiles the list of notifications to be sent to the management cluster regarding assets
// being online.
func (n *Notifier) notifyManagementCluster() {
	n.Lock()
	defer n.Unlock()
	if len(n.assetAlive) > 0{
		// TODO Implement send
		log.Debug().Int("len", len(n.assetAlive)).Msg("sending agent alive notifications")
		if n.sendAliveMessages(){
			// If successfull, cleanup the map
			for k := range n.assetAlive{
				delete(n.assetAlive, k)
			}
			for ip := range  n.AssetNewIP {
				delete (n.AssetNewIP, ip)
			}
		}
	}
	// TODO Iteration over other elements
}

func (n * Notifier) NotifyAgentStart(start * grpc_inventory_manager_go.AgentStartInfo) derrors.Error{
	n.Lock()
	defer n.Unlock()
	err := n.provider.AddAgentStart(*entities.NewAgentStartInfoFromGRPC(start))
	if err != nil{
		return err
	}
	// TODO Implement send or queue
	return nil
}

func (n * Notifier) NotifyCallback(response * grpc_inventory_manager_go.AgentOpResponse) derrors.Error {
	n.Lock()
	defer n.Unlock()
	err := n.provider.AddOpResponse(*entities.NewAgentOpResponseFromGRPC(response))
	if err != nil{
		return nil
	}
	// TODO Implement send or queue
	return nil
}