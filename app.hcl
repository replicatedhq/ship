asset "inline" "install_script" {

  dest = "install.sh"
  mode = 755
  contents = <<EOF
#!/bin/bash

echo "This script will install superbigtool"
sleep 5
echo "Whoops, something went wrong"
exit 1

EOF
}

asset "github" {
  dest = "manifests/"
  mode = 755

  repo = "github.com/replicatedhq/superbigtool-enterprise"
  ref = "aef8fe5afe5af67fe6789fa678fe876fa867fbc"
  path = "**/*.yml"
  depends = [
    "${docker_image.api}"
  ]
}

asset "docker_image" "api" {
  dest = "manifests/"
  mode = 755

  private = true
  image = "quay.io/retracedhq/api:21023910"
}

config "Kubernetes Cluster Info" {

  item {
    name = "namespace"
    required = "true"
    default = "default"
  }

  item {
    name = "num_workers"
    required = "true"
    default = "2"
  }
}

lifecycle "message" {
  contents = "generating assets..."
}

lifecycle "render" {}

lifecycle "message" {
  contents = "Done!"
}

