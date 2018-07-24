package api

// Spec is the top level Ship document that defines an application
type Spec struct {
	Assets    Assets    `json:"assets" yaml:"assets" hcl:"asset"`
	Lifecycle Lifecycle `json:"lifecycle" yaml:"lifecycle" hcl:"lifecycle"`
	Config    Config    `json:"config" yaml:"config" hcl:"config"`
}

// Image
type Image struct {
	URL      string `json:"url" yaml:"url" hcl:"url" meta:"url"`
	Source   string `json:"source" yaml:"source" hcl:"source" meta:"source"`
	AppSlug  string `json:"appSlug" yaml:"appSlug" hcl:"appSlug" meta:"appSlug"`
	ImageKey string `json:"imageKey" yaml:"imageKey" hcl:"imageKey" meta:"imageKey"`
}

type GithubContent struct {
	Repo  string       `json:"repo" yaml:"repo" hcl:"repo" meta:"repo"`
	Path  string       `json:"path" yaml:"path" hcl:"path" meta:"path"`
	Ref   string       `json:"ref" yaml:"ref" hcl:"ref" meta:"ref"`
	Files []GithubFile `json:"files" yaml:"files" hcl:"files" meta:"files"`
}

// GithubFile
type GithubFile struct {
	Name string `json:"name" yaml:"name" hcl:"name" meta:"name"`
	Path string `json:"path" yaml:"path" hcl:"path" meta:"path"`
	Sha  string `json:"sha" yaml:"sha" hcl:"sha" meta:"sha"`
	Size int64  `json:"size" yaml:"size" hcl:"size" meta:"size"`
	Data string `json:"data" yaml:"data" hcl:"data" meta:"data"`
}

type HelmChartMetadata struct {
	Description string `json:"description" yaml:"description" hcl:"description" meta:"description"`
	Version     string `json:"version" yaml:"version" hcl:"version" meta:"version"`
	Icon        string `json:"icon" yaml:"icon" hcl:"icon" meta:"icon"`
	Release     string `json:"release" yaml:"release" hcl:"release" meta:"release"`
}

// ReleaseMetadata
type ReleaseMetadata struct {
	ReleaseID         string            `json:"releaseId" yaml:"releaseId" hcl:"releaseId" meta:"release-id"`
	CustomerID        string            `json:"customerId" yaml:"customerId" hcl:"customerId" meta:"customer-id"`
	ChannelID         string            `json:"channelId" yaml:"channelId" hcl:"channelId" meta:"channel-id"`
	ChannelName       string            `json:"channelName" yaml:"channelName" hcl:"channelName" meta:"channel-name"`
	ChannelIcon       string            `json:"channelIcon" yaml:"channelIcon" hcl:"channelIcon" meta:"channel-icon"`
	Semver            string            `json:"semver" yaml:"semver" hcl:"semver" meta:"release-version"`
	ReleaseNotes      string            `json:"releaseNotes" yaml:"releaseNotes" hcl:"releaseNotes" meta:"release-notes"`
	Created           string            `json:"created" yaml:"created" hcl:"created" meta:"release-date"`
	RegistrySecret    string            `json:"registrySecret" yaml:"registrySecret" hcl:"registrySecret" meta:"registry-secret"`
	Images            []Image           `json:"images" yaml:"images" hcl:"images" meta:"images"`
	GithubContents    []GithubContent   `json:"githubContents" yaml:"githubContents" hcl:"githubContents" meta:"githubContents"`
	HelmChartMetadata HelmChartMetadata `json:"helmChartMetadata" yaml:"helmChartMetadata" hcl:"helmChartMetadata" meta:"helmChartMetadata"`
}

// Release
type Release struct {
	Metadata ReleaseMetadata
	Spec     Spec
}
