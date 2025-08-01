package types

type MIGDevice struct {
	DeviceID    int
	InstanceID  int
	GPU         string
	InUse       bool
	Memory      uint64
	ProfileName string
	start       int32
	size        int32
}

type GPU struct {
}

type Profile struct {
	GID  int   // GI profile id
	CID  int32 // CI profile id
	Size uint32
}
