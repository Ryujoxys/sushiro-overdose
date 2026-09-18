package app

import (
	"github.com/Ryujoxys/sushiro-overdose/internal/notify"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

// TestMain replaces both delivery boundaries before any test runs. Production
// keeps the real implementations, including the explicit notification test UI.
var (
	sendDesktopNotification = platform.DesktopNotification
	configuredNotifiers     = notify.BuildNotifierFromConfig
)
