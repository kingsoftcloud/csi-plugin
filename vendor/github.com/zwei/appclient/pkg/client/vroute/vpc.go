package vroute

import (
	"github.com/zwei/appclient/config"
	appHttp "github.com/zwei/appclient/pkg/http"
	"fmt"
	"github.com/golang/glog"
	"encoding/json"
	"time"
	"github.com/zwei/appclient/config/types"
	"github.com/zwei/appclient/pkg/client/aksk"
	"strconv"
	"net/url"
)

const (
	defaultVersion    = "2016-03-04"
	defautlServerName = "vpc"
)

type VpcClient struct {
	conf       *config.Config
	client     *appHttp.AppDataClient
	aksk       *types.AKSK
	tenantID   string
	region     string
	debug      bool
	headers    map[string]string
	kubeconfig string
}

func NewVpcClient(conf *config.Config) (*VpcClient, error) {
	if len(conf.NetworkEndpoint) == 0 {
		conf.NetworkEndpoint = config.NetworkEndpoint
	}
	dataClient := appHttp.NewAppDataClient()
	dataClient.SetEndpoint(conf.NetworkEndpoint)
	if len(conf.Token) == 0 {
		conf.Token = fmt.Sprintf("%s:%s", conf.UserID, conf.TenantID)
	}

	headers := make(map[string]string)
	headers["X-Auth-Project-Id"] = conf.TenantID
	headers["X-Auth-Token"] = conf.Token
	headers["User-Agent"] = "app-agent"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"
	// headers["X-Auth-User-Tag"] = "docker"

	vpcClient := &VpcClient{
		conf:       conf,
		headers:    headers,
		debug:      true,
		client:     dataClient,
		tenantID:   conf.TenantID,
		region:     conf.Region,
		kubeconfig: conf.Kubeconfig,
		aksk:       &types.AKSK{},
	}

	return vpcClient, nil
}

// http://neutron:9696/v2.0/vpc/domains/domains_id
func (c *VpcClient) DescribeVpc(id string) (vpc *VpcSetType, err error) {
	if err := aksk.GetAKSK(c.kubeconfig, c.aksk); err != nil {
		return nil, err
	}
	action := url.Values{
		"Action":  []string{"DescribeVpcs"},
		"Version": []string{defaultVersion},
		"VpcId.1": []string{id},
	}
	glog.V(9).Infof("get neutron vpc: %s", c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(appHttp.GET)
	c.client.SetSigner(defautlServerName, c.region, c.aksk.AK, c.aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	response := new(DescribeVpcResponse)
	err = json.Unmarshal([]byte(data), response)
	if err != nil {
		return nil, err
	}
	return &response.Domain, nil
}

func (c *VpcClient) CreateRoute(args *RouteArgs) (route *RouteSetType, err error) {
	// http://{{neutron_host}}:9696/v2.0/vpc/routes
	if err := aksk.GetAKSK(c.kubeconfig, c.aksk); err != nil {
		return nil, err
	}
	action := url.Values{
		"Action":     []string{"CreateRoute"},
		"Version":    []string{defaultVersion},
		"VpcId":      []string{args.DomainId},
		"RouteType":  []string{args.InstanceType},
		"InstanceId": []string{args.InstanceId},
		"Ip":         []string{args.Ip},
		"Mask":       []string{strconv.Itoa(int(args.Mask))},
	}
	glog.V(9).Infof("create neutron route : %s", c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetBody(args)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(appHttp.POST)
	c.client.SetSigner(defautlServerName, c.region, c.aksk.AK, c.aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(data), route)
	if err != nil {
		return nil, err
	}
	return route, nil
}

func (c *VpcClient) DeleteRoute(id string) error {
	// http://{{neutron_host}}:9696/v2.0/vpc/routes
	if err := aksk.GetAKSK(c.kubeconfig, c.aksk); err != nil {
		return err
	}
	action := url.Values{
		"Action":  []string{"DeleteRoute"},
		"Version": []string{defaultVersion},
		"RouteId": []string{id},
	}
	glog.V(9).Infof("create neutron route : %s", c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(appHttp.DELETE)
	c.client.SetSigner(defautlServerName, c.region, c.aksk.AK, c.aksk.SK)
	if _, err := c.client.Go(); err != nil {
		return err
	}
	return nil
}

// GetRouteEntry get routes entry
func (c *VpcClient) GetRoutes(args *RouteArgs) ([]RouteSetType, error) {
	// http://{{neutron_host}}:9696/v2.0/vpc/routes
	if err := aksk.GetAKSK(c.kubeconfig, c.aksk); err != nil {
		return nil, err
	}
	action := url.Values{
		"Action":         []string{"DeleteRoute"},
		"Version":        []string{defaultVersion},
		"Filter.1.Name":  []string{"vpc-id"},
		"Filter.1.Value": []string{args.DomainId},
		"Filter.2.Name":  []string{"route-type"},
		"Filter.2.Value": []string{args.InstanceType},
		"Filter.3.Name":  []string{"ip"},
		"Filter.3.Value": []string{args.Ip},
		"Filter.4.Name":  []string{"mask"},
		"Filter.4.Value": []string{strconv.Itoa(int(args.Mask))},
	}
	glog.V(9).Infof("get neutron route : %s", c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(appHttp.GET)
	c.client.SetSigner(defautlServerName, c.region, c.aksk.AK, c.aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}

	response := new(GetRoutesResponse)
	err = json.Unmarshal([]byte(data), response)
	if err != nil {
		return nil, err
	}
	return response.Routes, nil
}

// GetRouteEntry get route entry
func (c *VpcClient) DescribeRoute(id string) (*RouteSetType, error) {
	// http://{{neutron_host}}:9696/v2.0/vpc/routes/<route_id>
	if err := aksk.GetAKSK(c.kubeconfig, c.aksk); err != nil {
		return nil, err
	}
	action := url.Values{
		"Action":  []string{"DeleteRoute"},
		"Version": []string{defaultVersion},
		"RouteId": []string{id},
	}
	glog.V(9).Infof("create neutron route : %s", c.conf.NetworkEndpoint)
	c.client.SetHeader(c.headers)
	c.client.SetUrlQuery("", action)
	c.client.SetMethod(appHttp.GET)
	c.client.SetSigner(defautlServerName, c.region, c.aksk.AK, c.aksk.SK)
	data, err := c.client.Go()
	if err != nil {
		return nil, err
	}
	response := new(DescribeRouteResponse)
	err = json.Unmarshal([]byte(data), response)
	if err != nil {
		return nil, err
	}
	return &response.Route, nil
}

// WaitForAllRouteEntriesAvailable waits for all route entries to Available status
func (c *VpcClient) WaitForAllRouteEntriesAvailable(vrouterId string, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	for {
		success := true
		route, err := c.DescribeRoute(vrouterId)
		if err != nil || len(route.Id) == 0 {
			success = false
		}

		if success {
			break
		} else {
			timeout = timeout - DefaultWaitForInterval
			if timeout <= 0 {
				return GetClientErrorFromString("Timeout", "")
			}
			time.Sleep(DefaultWaitForInterval * time.Second)
		}
	}
	return nil
}
