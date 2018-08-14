package helm

func Fetch(url, version, name, dest string) error {
	toFetch := fetchCmd{
		untar:    true,
		untardir: dest,
		destdir:  dest,
		repoURL:  url,
		version:  version,
		chartRef: name,
	}

	err := toFetch.run()
	return err
}
