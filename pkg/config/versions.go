package config

// Binaries holds the paths to required binaries
type Binaries struct {
	Wine       string
	Wineboot   string
	Msidb      string
	Winecfg    string
	Wineserver string
}

// Versions holds all version strings and paths for the installer
type Versions struct {
	Workdir       string
	ProtonVer     string
	ProtonBaseURL string
	WineVer       string
	WinetricksVer string
	DXVKVer       string
	VKD3DVer      string
	FSRPath       string
	Binaries      Binaries
}

// DefaultVersions contains the version configuration
// Wine Binaries are set later after installing to `~/.local/share/bellum/`
var DefaultVersions = Versions{
	Workdir:       ".",
	ProtonVer:     "proton-cachyos-10.0-20260424-slr-x86_64",
	ProtonBaseURL: "https://github.com/CachyOS/proton-cachyos/releases/download",
	WineVer:       "bellum-wine-11.8",
	WinetricksVer: "20250102-modified",
	DXVKVer:       "2.7.1-3-521-low-latency",
	VKD3DVer:      "2.14",
	FSRPath:       "packages/fsr4",
	Binaries: Binaries{
		Wine:       "",
		Wineboot:   "",
		Msidb:      "",
		Winecfg:    "",
		Wineserver: "",
	},
}
