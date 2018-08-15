package constants

// InstallerPrefixPath is the path prefix of installed assets
const InstallerPrefixPath = "installer"

// ShipPath is the default folder path of Ship configuration
const ShipPath = ".ship"

// OverlaysPrefixPath is the path prefix of overlays
const OverlaysPrefixPath = "overlays/ship"

// StatePath is the default state file path
const StatePath = ".ship/state.json"

// ReleasePath is the default place to write a pulled release to the filesystem
const ReleasePath = ".ship/release.yml"

// KustomizeHelmPath is the path used to store Helm chart contents
const KustomizeHelmPath = "chart"

// RenderedHelmTempPath is the path where the `helm template` command writes to
const RenderedHelmTempPath = ".ship/tmp-rendered"

// RenderedHelmPath is the path where rendered Helm charts are written to
const RenderedHelmPath = "base"

// TempHelmValuesPath is the folder path used to store the updated values.yaml
const TempHelmValuesPath = "chart/tmp"

// TempApplyOverlayPath is the folder path used to apply patch
const TempApplyOverlayPath = "overlays/tmp-apply"
