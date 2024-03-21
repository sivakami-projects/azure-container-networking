package network

import (
	"github.com/Azure/azure-container-networking/common"
)

// MockNetworkManager is a mock structure for Network Manager
type MockNetworkManager struct {
	TestNetworkInfoMap  map[string]*NetworkInfo
	TestEndpointInfoMap map[string]*EndpointInfo
	TestEndpointClient  *MockEndpointClient
}

// NewMockNetworkmanager returns a new mock
func NewMockNetworkmanager(mockEndpointclient *MockEndpointClient) *MockNetworkManager {
	return &MockNetworkManager{
		TestNetworkInfoMap:  make(map[string]*NetworkInfo),
		TestEndpointInfoMap: make(map[string]*EndpointInfo),
		TestEndpointClient:  mockEndpointclient,
	}
}

// Initialize mock
func (nm *MockNetworkManager) Initialize(config *common.PluginConfig, isRehydrationRequired bool) error {
	return nil
}

// Uninitialize mock
func (nm *MockNetworkManager) Uninitialize() {}

// AddExternalInterface mock
func (nm *MockNetworkManager) AddExternalInterface(ifName string, subnet string) error {
	return nil
}

// CreateNetwork mock
func (nm *MockNetworkManager) CreateNetwork(nwInfo *NetworkInfo) error {
	nm.TestNetworkInfoMap[nwInfo.Id] = nwInfo
	return nil
}

// DeleteNetwork mock
func (nm *MockNetworkManager) DeleteNetwork(networkID string) error {
	return nil
}

// GetNetworkInfo mock
func (nm *MockNetworkManager) GetNetworkInfo(networkID string) (NetworkInfo, error) {
	if info, exists := nm.TestNetworkInfoMap[networkID]; exists {
		return *info, nil
	}
	return NetworkInfo{}, errNetworkNotFound
}

// CreateEndpoint mock
func (nm *MockNetworkManager) CreateEndpoint(_ apipaClient, _ string, epInfos []*EndpointInfo) error {
	for _, epInfo := range epInfos {
		if err := nm.TestEndpointClient.AddEndpoints(epInfo); err != nil {
			return err
		}
	}

	nm.TestEndpointInfoMap[epInfos[0].Id] = epInfos[0]
	return nil
}

// DeleteEndpoint mock
func (nm *MockNetworkManager) DeleteEndpoint(_, endpointID string, _ *EndpointInfo) error {
	delete(nm.TestEndpointInfoMap, endpointID)
	return nil
}

// SetStatelessCNIMode enable the statelessCNI falg and inititlizes a CNSClient
func (nm *MockNetworkManager) SetStatelessCNIMode() error {
	return nil
}

// IsStatelessCNIMode checks if the Stateless CNI mode has been enabled or not
func (nm *MockNetworkManager) IsStatelessCNIMode() bool {
	return false
}

// GetEndpointID returns the ContainerID value
func (nm *MockNetworkManager) GetEndpointID(containerID, ifName string) string {
	if nm.IsStatelessCNIMode() {
		return containerID
	}
	if len(containerID) > ContainerIDLength {
		containerID = containerID[:ContainerIDLength]
	} else {
		return ""
	}
	return containerID + "-" + ifName
}

func (nm *MockNetworkManager) GetAllEndpoints(networkID string) (map[string]*EndpointInfo, error) {
	return nm.TestEndpointInfoMap, nil
}

// GetEndpointInfo mock
func (nm *MockNetworkManager) GetEndpointInfo(_, endpointID string) (*EndpointInfo, error) {
	if info, exists := nm.TestEndpointInfoMap[endpointID]; exists {
		return info, nil
	}
	return nil, errEndpointNotFound
}

// GetEndpointInfoBasedOnPODDetails mock
func (nm *MockNetworkManager) GetEndpointInfoBasedOnPODDetails(networkID string, podName string, podNameSpace string, doExactMatchForPodName bool) (*EndpointInfo, error) {
	return &EndpointInfo{}, nil
}

// AttachEndpoint mock
func (nm *MockNetworkManager) AttachEndpoint(networkID string, endpointID string, sandboxKey string) (*endpoint, error) {
	return &endpoint{}, nil
}

// DetachEndpoint mock
func (nm *MockNetworkManager) DetachEndpoint(networkID string, endpointID string) error {
	return nil
}

// UpdateEndpoint mock
func (nm *MockNetworkManager) UpdateEndpoint(networkID string, existingEpInfo *EndpointInfo, targetEpInfo *EndpointInfo) error {
	return nil
}

// GetNumberOfEndpoints mock
func (nm *MockNetworkManager) GetNumberOfEndpoints(ifName string, networkID string) int {
	return 0
}

func (nm *MockNetworkManager) FindNetworkIDFromNetNs(netNs string) (string, error) {
	// based on the GetAllEndpoints func above, it seems that this mock is only intended to be used with
	// one network, so just return the network here if it exists
	for network := range nm.TestNetworkInfoMap {
		return network, nil
	}

	return "", errNetworkNotFound
}

// GetNumEndpointsByContainerID mock
func (nm *MockNetworkManager) GetNumEndpointsByContainerID(_ string) int {
	// based on the GetAllEndpoints func above, it seems that this mock is only intended to be used with
	// one network, so just return the number of endpoints if network exists
	numEndpoints := 0

	for _, network := range nm.TestNetworkInfoMap {
		if _, err := nm.GetAllEndpoints(network.Id); err == nil {
			numEndpoints++
		}
	}

	return numEndpoints
}
