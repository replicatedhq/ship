All files in this directory besides `exports.go` are copied from [Helm v2.9.1](https://github.com/helm/helm/tree/v2.9.1/cmd/helm)
except for `exports.go` which exports the commands that Ship uses elsewhere. The copied files have three changes - `package helm` is replaced with `package helm` with this command:
```
sed -i 's/package main/package helm/g' `ag -l 'package main' .`
```
the `root()` function within `helm.go` has been commented out, and `canonical paths` (`// import "k8s.io/helm/cmd/helm/installer"` and `// import "k8s.io/helm/cmd/helm"`) have been removed.

Further, several changes have been made to the test files due to missing files:
1. `TestInitCmd_tlsOptions` has been disabled.
2. `TestInitCmd_output` has been disabled.
3. `TestTemplateCmd` has been disabled.
4. `TestSecretManifest` within `installer` has been disabled.
5. `TestInstall_WithTLS` within `installer` has been disabled.