package vroute

const (
	DefaultTimeout         = 60
	DefaultWaitForInterval = 5
)

type IpMask uint

const (
	IpMask24 = IpMask(24)
	IpMask16 = IpMask(16)
	IpMask8  = IpMask(8)
)

const (
	RouteTypeVmHost = RouteType("vmhost")
)

type Response struct {
	RequestId string `json:"request_id"`
}

type RouteType string

type RouteInstanceType string

type VpcSetType struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Ip      string `json:"ip"`
	Mask    int    `json:"mask"`
	Deleted bool   `json:"deleted"`
}

type DescribeVpcResponse struct {
	Response
	Domain VpcSetType `json:"domain"`
}

// route type
type RouteSetType struct {
	Id           string            `json:"id"`
	DomainId     string            `json:"domain_id"`
	InstanceId   string            `json:"instance_id"`
	InstanceType RouteInstanceType `json:"instance_type"`
	Ip           string            `json:"ip"`
	Mask         IpMask            `json:"mask"`
	System       bool              `json:"system"`
	Type         RouteType         `json:"type"`
	Vnetid       string            `json:"vnet_id"`
	VnetName     string            `json:"vnet_name"`
}

type RouteArgs struct {
	DomainId     string `json:"domain_id"`
	InstanceId   string `json:"instance_id"`
	InstanceType string `json:"instance_type"`
	Ip           string `json:"ip"`
	Mask         uint   `json:"mask"`
}

type GetRoutesResponse struct {
	Response
	Routes []RouteSetType `json:"routes"`
}

type DescribeRouteResponse struct {
	Response
	Route RouteSetType `json:"route"`
}
