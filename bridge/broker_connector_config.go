package bridge

type Stomp int
type Rabbit int

type BrokerConnectorConfig struct {
    Username        string
    Password        string
    ServerAddr      string
    WSPath          string
    UseWS           string
    BrokerType      int
    HostHeader      string
}
