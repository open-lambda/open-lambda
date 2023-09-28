package loadbalancer

const (
	Random = 0
	KMeans = 1
)

const (
	NumGroup = 5
)

var Lb *LoadBalancer

type LoadBalancer struct {
	LbType int
}

func InitLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		LbType: KMeans,
	}
}
