package driver

const (
	vniMin = 1
	vniMax = 16777215
)

const (
	bridgeDefaultMacLimit       = 16384
	bridgeDefaultMacLearn       = "enable"
	bridgeDefaultMacAlarm       = "disable"
	bridgeDefaultMcFlood        = "enable"
	bridgeDefaultBcFlood        = "enable"
	bridgeDefaultUnknownUcFlood = "enable"
	bridgeDefaultMacLimitAction = "forward"
)

const (
	bridgePortDefaultTagMode        = "untag"
	bridgePortDefaultMacLearn       = "enable"
	bridgePortDefaultMacLimit       = 16384
	bridgePortDefaultMacAlarm       = "disable"
	bridgePortDefaultMacLimitAction = "forward"
)

const (
	interfaceDefaultMtu         = 9600
	interfaceDefaultAdminStatus = "up"
)
