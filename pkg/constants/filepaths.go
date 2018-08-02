package constants

// InstallerPrefixPath is the path prefix of installed assets
const InstallerPrefixPath = "installer"

// ShipPath is the default folder path of Ship configuration
const ShipPath = ".ship"

// OverlaysPrefixPath is the path prefix of overlays
const OverlaysPrefixPath = "overlays/ship"

// StatePath is the default state file path
const StatePath = ".ship/state.json"

// KustomizeHelmPath is the path used to store Helm chart contents
const KustomizeHelmPath = "chart"

// RenderedHelmPath is the path where rendered Helm charts are written to
const RenderedHelmPath = "base"

// TempHelmValuesPath is the folder path used to store the updated values.yaml
const TempHelmValuesPath = "chart/tmp"
