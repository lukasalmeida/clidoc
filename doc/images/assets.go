package images

import "embed"

// FS contains the static image assets served by the preview UI.
//
//go:embed icon.png icon_transp.png
var FS embed.FS
