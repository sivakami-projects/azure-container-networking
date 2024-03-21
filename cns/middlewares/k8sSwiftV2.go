package middlewares

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/configuration"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/middlewares/utils"
	"github.com/Azure/azure-container-networking/cns/types"
	"github.com/Azure/azure-container-networking/crd/multitenancy/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errMTPNCNotReady            = errors.New("mtpnc is not ready")
	errInvalidSWIFTv2NICType    = errors.New("invalid NIC type for SWIFT v2 scenario")
	errInvalidMTPNCPrefixLength = errors.New("invalid prefix length for MTPNC primaryIP, must be 32")
)

const (
	prefixLength     = 32
	overlayGatewayv4 = "169.254.1.1"
	virtualGW        = "169.254.2.1"
	overlayGatewayV6 = "fe80::1234:5678:9abc"
)

type K8sSWIFTv2Middleware struct {
	Cli client.Client
}

// Verify interface compliance at compile time
var _ cns.IPConfigsHandlerMiddleware = (*K8sSWIFTv2Middleware)(nil)

// IPConfigsRequestHandlerWrapper is the middleware function for handling SWIFT v2 IP configs requests for AKS-SWIFT. This function wrapped the default SWIFT request
// and release IP configs handlers.
func (m *K8sSWIFTv2Middleware) IPConfigsRequestHandlerWrapper(defaultHandler, failureHandler cns.IPConfigsHandlerFunc) cns.IPConfigsHandlerFunc {
	return func(ctx context.Context, req cns.IPConfigsRequest) (*cns.IPConfigsResponse, error) {
		podInfo, respCode, message := m.validateIPConfigsRequest(ctx, &req)

		if respCode != types.Success {
			return &cns.IPConfigsResponse{
				Response: cns.Response{
					ReturnCode: respCode,
					Message:    message,
				},
			}, errors.New("failed to validate ip configs request")
		}
		ipConfigsResp, err := defaultHandler(ctx, req)
		// If the pod is not v2, return the response from the handler
		if !req.SecondaryInterfacesExist {
			return ipConfigsResp, err
		}
		// If the pod is v2, get the infra IP configs from the handler first and then add the SWIFTv2 IP config
		defer func() {
			// Release the default IP config if there is an error
			if err != nil {
				_, err = failureHandler(ctx, req)
				if err != nil {
					logger.Errorf("failed to release default IP config : %v", err)
				}
			}
		}()
		if err != nil {
			return ipConfigsResp, err
		}
		SWIFTv2PodIPInfo, err := m.getIPConfig(ctx, podInfo)
		if err != nil {
			return &cns.IPConfigsResponse{
				Response: cns.Response{
					ReturnCode: types.FailedToAllocateIPConfig,
					Message:    fmt.Sprintf("AllocateIPConfig failed: %v, IP config request is %v", err, req),
				},
				PodIPInfo: []cns.PodIpInfo{},
			}, errors.Wrapf(err, "failed to get SWIFTv2 IP config : %v", req)
		}
		ipConfigsResp.PodIPInfo = append(ipConfigsResp.PodIPInfo, SWIFTv2PodIPInfo)
		// Set routes for the pod
		for i := range ipConfigsResp.PodIPInfo {
			ipInfo := &ipConfigsResp.PodIPInfo[i]
			err = m.setRoutes(ipInfo)
			if err != nil {
				return &cns.IPConfigsResponse{
					Response: cns.Response{
						ReturnCode: types.FailedToAllocateIPConfig,
						Message:    fmt.Sprintf("AllocateIPConfig failed: %v, IP config request is %v", err, req),
					},
					PodIPInfo: []cns.PodIpInfo{},
				}, errors.Wrapf(err, "failed to set routes for pod %s", podInfo.Name())
			}
		}
		return ipConfigsResp, nil
	}
}

// validateIPConfigsRequest validates if pod is multitenant by checking the pod labels, used in SWIFT V2 AKS scenario.
// nolint
func (m *K8sSWIFTv2Middleware) validateIPConfigsRequest(ctx context.Context, req *cns.IPConfigsRequest) (podInfo cns.PodInfo, respCode types.ResponseCode, message string) {
	// Retrieve the pod from the cluster
	podInfo, err := cns.UnmarshalPodInfo(req.OrchestratorContext)
	if err != nil {
		errBuf := errors.Wrapf(err, "failed to unmarshalling pod info from ipconfigs request %+v", req)
		return nil, types.UnexpectedError, errBuf.Error()
	}
	logger.Printf("[SWIFTv2Middleware] validate ipconfigs request for pod %s", podInfo.Name())
	podNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
	pod := v1.Pod{}
	if err := m.Cli.Get(ctx, podNamespacedName, &pod); err != nil {
		errBuf := errors.Wrapf(err, "failed to get pod %+v", podNamespacedName)
		return nil, types.UnexpectedError, errBuf.Error()
	}

	// check the pod labels for Swift V2, set the request's SecondaryInterfaceSet flag to true and check if its MTPNC CRD is ready
	if _, ok := pod.Labels[configuration.LabelPodSwiftV2]; ok {
		req.SecondaryInterfacesExist = true
		// Check if the MTPNC CRD exists for the pod, if not, return error
		mtpnc := v1alpha1.MultitenantPodNetworkConfig{}
		mtpncNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
		if err := m.Cli.Get(ctx, mtpncNamespacedName, &mtpnc); err != nil {
			return nil, types.UnexpectedError, fmt.Errorf("failed to get pod's mtpnc from cache : %w", err).Error()
		}
		// Check if the MTPNC CRD is ready. If one of the fields is empty, return error
		if mtpnc.Status.PrimaryIP == "" || mtpnc.Status.MacAddress == "" || mtpnc.Status.NCID == "" || mtpnc.Status.GatewayIP == "" {
			return nil, types.UnexpectedError, errMTPNCNotReady.Error()
		}
	}
	logger.Printf("[SWIFTv2Middleware] pod %s has secondary interface : %v", podInfo.Name(), req.SecondaryInterfacesExist)
	// retrieve podinfo from orchestrator context
	return podInfo, types.Success, ""
}

// getIPConfig returns the pod's SWIFT V2 IP configuration.
func (m *K8sSWIFTv2Middleware) getIPConfig(ctx context.Context, podInfo cns.PodInfo) (cns.PodIpInfo, error) {
	// Check if the MTPNC CRD exists for the pod, if not, return error
	mtpnc := v1alpha1.MultitenantPodNetworkConfig{}
	mtpncNamespacedName := k8stypes.NamespacedName{Namespace: podInfo.Namespace(), Name: podInfo.Name()}
	if err := m.Cli.Get(ctx, mtpncNamespacedName, &mtpnc); err != nil {
		return cns.PodIpInfo{}, errors.Wrapf(err, "failed to get pod's mtpnc from cache")
	}

	// Check if the MTPNC CRD is ready. If one of the fields is empty, return error
	if mtpnc.Status.PrimaryIP == "" || mtpnc.Status.MacAddress == "" || mtpnc.Status.NCID == "" || mtpnc.Status.GatewayIP == "" {
		return cns.PodIpInfo{}, errMTPNCNotReady
	}
	logger.Printf("[SWIFTv2Middleware] mtpnc for pod %s is : %+v", podInfo.Name(), mtpnc)

	// Parse MTPNC primaryIP to get the IP address and prefix length
	p, err := netip.ParsePrefix(mtpnc.Status.PrimaryIP)
	if err != nil {
		return cns.PodIpInfo{}, errors.Wrapf(err, "failed to parse mtpnc primaryIP %s", mtpnc.Status.PrimaryIP)
	}
	// Get the IP address and prefix length
	ip := p.Addr()
	prefixSize := p.Bits()
	if prefixSize != prefixLength {
		return cns.PodIpInfo{}, errors.Wrapf(errInvalidMTPNCPrefixLength, "mtpnc primaryIP prefix length is %d", prefixSize)
	}
	podIPInfo := cns.PodIpInfo{
		PodIPConfig: cns.IPSubnet{
			IPAddress:    ip.String(),
			PrefixLength: uint8(prefixSize),
		},
		MacAddress:        mtpnc.Status.MacAddress,
		NICType:           cns.DelegatedVMNIC,
		SkipDefaultRoutes: false,
		// InterfaceName is empty for DelegatedVMNIC
	}

	return podIPInfo, nil
}

// setRoutes sets the routes for podIPInfo used in SWIFT V2 scenario.
func (m *K8sSWIFTv2Middleware) setRoutes(podIPInfo *cns.PodIpInfo) error {
	logger.Printf("[SWIFTv2Middleware] set routes for pod with nic type : %s", podIPInfo.NICType)
	podIPInfo.Routes = []cns.Route{}
	switch podIPInfo.NICType {
	case cns.DelegatedVMNIC:
		virtualGWRoute := cns.Route{
			IPAddress: fmt.Sprintf("%s/%d", virtualGW, prefixLength),
		}
		// default route via SWIFT v2 interface
		route := cns.Route{
			IPAddress:        "0.0.0.0/0",
			GatewayIPAddress: virtualGW,
		}
		podIPInfo.Routes = []cns.Route{virtualGWRoute, route}
	case cns.InfraNIC:
		// Get and parse infraVNETCIDRs from env
		infraVNETCIDRs, err := configuration.InfraVNETCIDRs()
		if err != nil {
			return errors.Wrapf(err, "failed to get infraVNETCIDRs from env")
		}
		infraVNETCIDRsv4, infraVNETCIDRsv6, err := utils.ParseCIDRs(infraVNETCIDRs)
		if err != nil {
			return errors.Wrapf(err, "failed to parse infraVNETCIDRs")
		}

		// Get and parse podCIDRs from env
		podCIDRs, err := configuration.PodCIDRs()
		if err != nil {
			return errors.Wrapf(err, "failed to get podCIDRs from env")
		}
		podCIDRsV4, podCIDRv6, err := utils.ParseCIDRs(podCIDRs)
		if err != nil {
			return errors.Wrapf(err, "failed to parse podCIDRs")
		}

		// Get and parse serviceCIDRs from env
		serviceCIDRs, err := configuration.ServiceCIDRs()
		if err != nil {
			return errors.Wrapf(err, "failed to get serviceCIDRs from env")
		}
		serviceCIDRsV4, serviceCIDRsV6, err := utils.ParseCIDRs(serviceCIDRs)
		if err != nil {
			return errors.Wrapf(err, "failed to parse serviceCIDRs")
		}
		// Check if the podIPInfo is IPv4 or IPv6
		ip, err := netip.ParseAddr(podIPInfo.PodIPConfig.IPAddress)
		if err != nil {
			return errors.Wrapf(err, "failed to parse podIPConfig IP address %s", podIPInfo.PodIPConfig.IPAddress)
		}
		if ip.Is4() {
			// routes for IPv4 podCIDR traffic
			for _, podCIDRv4 := range podCIDRsV4 {
				podCIDRv4Route := cns.Route{
					IPAddress:        podCIDRv4,
					GatewayIPAddress: overlayGatewayv4,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, podCIDRv4Route)
			}
			// route for IPv4 serviceCIDR traffic
			for _, serviceCIDRv4 := range serviceCIDRsV4 {
				serviceCIDRv4Route := cns.Route{
					IPAddress:        serviceCIDRv4,
					GatewayIPAddress: overlayGatewayv4,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, serviceCIDRv4Route)
			}
			// route for IPv4 infraVNETCIDR traffic
			for _, infraVNETCIDRv4 := range infraVNETCIDRsv4 {
				infraVNETCIDRv4Route := cns.Route{
					IPAddress:        infraVNETCIDRv4,
					GatewayIPAddress: overlayGatewayv4,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, infraVNETCIDRv4Route)
			}
		} else {
			// routes for IPv6 podCIDR traffic
			for _, podCIDRv6 := range podCIDRv6 {
				podCIDRv6Route := cns.Route{
					IPAddress:        podCIDRv6,
					GatewayIPAddress: overlayGatewayV6,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, podCIDRv6Route)
			}
			// route for IPv6 serviceCIDR traffic
			for _, serviceCIDRv6 := range serviceCIDRsV6 {
				serviceCIDRv6Route := cns.Route{
					IPAddress:        serviceCIDRv6,
					GatewayIPAddress: overlayGatewayV6,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, serviceCIDRv6Route)
			}
			// route for IPv6 infraVNETCIDR traffic
			for _, infraVNETCIDRv6 := range infraVNETCIDRsv6 {
				infraVNETCIDRv6Route := cns.Route{
					IPAddress:        infraVNETCIDRv6,
					GatewayIPAddress: overlayGatewayV6,
				}
				podIPInfo.Routes = append(podIPInfo.Routes, infraVNETCIDRv6Route)
			}
		}
		podIPInfo.SkipDefaultRoutes = true
	case cns.BackendNIC:
	default:
		return errInvalidSWIFTv2NICType
	}
	return nil
}
