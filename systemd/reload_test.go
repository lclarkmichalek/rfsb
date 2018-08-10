package systemd

import "github.com/lclarkmichalek/rfsb"

var (
	_ rfsb.Resource = &DaemonReload{}
	_ rfsb.Resource = &StartUnit{}
)
