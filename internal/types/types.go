package types

type MIGDevice struct {
	DeviceID    int
	InstanceID  int
	GPU         string
	InUse       bool
	Memory      uint64
	ProfileName string
}
