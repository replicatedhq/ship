All files in this directory besides `exports.go` are copied from [Helm v2.9.1](https://github.com/helm/helm/tree/v2.9.1/cmd/helm)
except for `exports.go` which exports the commands that Ship uses elsewhere. The copied files have three changes - `package helm` is replaced with `package helm` with this command:
```
sed -i 's/package main/package helm/g' `ag -l 'package main' .`
```
the `root()` function within `helm.go` has been commented out, and `canonical paths` (`// import "k8s.io/helm/cmd/helm/installer"` and `// import "k8s.io/helm/cmd/helm"`) have been removed.

Further, several changes have been made to the test files:
1. `TestInitCmd_tlsOptions` has had the test directory changed from `../../testdata` to `./helmtestdata`.
2. `TestTemplateCmd` has had the test directory changed from `./../../pkg/chartutil/testdata/subpop/charts/subchart1` to `./subchart1`.
3. `tlsTestFile` within `installer` has been changed from `../../../testdata` to `../helmtestdata`.
4. `TestLoadPlugins` has been disabled due to breaking Circle.

To support this, the files from the relevant helm paths have been added within `pkg/helm`.