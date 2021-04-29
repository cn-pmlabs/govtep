package govtep

// RemoteMcastfdb ...
type RemoteMcastfdb struct {
	UUID           string
	Bridge         string //Bridge uuid
	IsolationGroup string //水平分割组
	Mac            string
	LocatorSet     []string //Locator set uuid
}

// LocalMcastfdb ...
type LocalMcastfdb struct {
	UUID           string
	Bridge         string //Bridge uuid
	IsolationGroup string //水平分割组
	Mac            string
	L2portSet      []string //L2port set uuid
}
