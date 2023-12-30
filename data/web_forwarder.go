package data

type RegisterWebForwarderFunc func(hostname string, dstIP string, dstHttpPort int, dstHttpsPort int)

type WebForwardTarget struct {
	Hostname     string
	DstIP        string
	DstHttpPort  int
	DstHttpsPort int
}
