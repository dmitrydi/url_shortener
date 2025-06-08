package gl

import "log"

var Log *log.Logger

func init() {
	Log = log.Default()
}
