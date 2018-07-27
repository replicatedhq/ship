package constants

const InstallerPrefix = "installer"

// BasePath is the default file path
const BasePath = ".ship"

// StatePath is the default state file path
const StatePath = ".ship/state.json"

// KustomizeHelmPath is the path used to store helm chart contents
const KustomizeHelmPath = ".ship/kustomize/chart"

// TempHelmValuesPath is the path used to store the updated values.yaml
const TempHelmValuesPath = ".ship/kustomize/tmp"
