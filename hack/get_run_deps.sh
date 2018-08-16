#!/bin/sh
# "15th competing standard" copied from the various dockerfiles
#
# used in circle
set -ex

HELM_VERSION="v2.9.1"
HELM_URL="https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-linux-amd64.tar.gz"
HELM_TGZ="helm-v2.9.1-linux-amd64.tar.gz"
HELM="linux-amd64/helm"
HELM_SHA256SUM="56ae2d5d08c68d6e7400d462d6ed10c929effac929fedce18d2636a9b4e166ba"
TERRAFORM_VERSION="0.11.7"
TERRAFORM_URL="https://releases.hashicorp.com/terraform/0.11.7/terraform_0.11.7_linux_amd64.zip"
TERRAFORM_ZIP="terraform_0.11.7_linux_amd64.zip"
TERRAFORM_SHA256SUM="6b8ce67647a59b2a3f70199c304abca0ddec0e49fd060944c26f666298e23418"
KUBECTL_VERSION=v1.11.1
KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c

mkdir -p /usr/local/bin

curl -fsSLO "${HELM_URL}" \
    && echo "${HELM_SHA256SUM}  ${HELM_TGZ}" | sha256sum -c - \
    && tar xvf "$HELM_TGZ" \
    && mv "$HELM" "/usr/local/bin/helm-${HELM_VERSION}" \
    && ln -s "/usr/local/bin/helm-${HELM_VERSION}" /usr/local/bin/helm \
    && rm "$HELM_TGZ"

curl -fsSLO "${TERRAFORM_URL}" \
	&& echo "${TERRAFORM_SHA256SUM}  ${TERRAFORM_ZIP}" | sha256sum -c - \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform \
	&& rm "$TERRAFORM_ZIP"

curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

