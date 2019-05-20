ARG source_tag=slim
FROM replicated/ship:${source_tag}

ENV TERRAFORM_VERSION=0.11.14
ENV TERRAFORM_URL="https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_ZIP="terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
ENV TERRAFORM_SHA256SUM=9b9a4492738c69077b079e595f5b2a9ef1bc4e8fb5596610f69a6f322a8af8dd

RUN curl -fsSLO "$TERRAFORM_URL" \
	&& echo "${TERRAFORM_SHA256SUM}  ${TERRAFORM_ZIP}" | sha256sum -c - \
	&& unzip "$TERRAFORM_ZIP" \
	&& mv "terraform" "/usr/local/bin/terraform-${TERRAFORM_VERSION}" \
	&& ln -s "/usr/local/bin/terraform-${TERRAFORM_VERSION}" /usr/local/bin/terraform \
	&& rm $TERRAFORM_ZIP

ENV AWS_IAM_AUTHENTICATOR_VERSION=1.11.5
ENV AWS_IAM_AUTHENTICATOR_URL=https://amazon-eks.s3-us-west-2.amazonaws.com/1.11.5/2018-12-06/bin/linux/amd64/aws-iam-authenticator
ENV AWS_IAM_AUTHENTICATOR_SHA256SUM=a46c66eb14ad08204f2f588b32dc50b10e9a8a0cc48ddf0966596d3c07abe059
RUN curl -o aws-iam-authenticator -fsSLO  "$AWS_IAM_AUTHENTICATOR_URL" \
	&& echo "${AWS_IAM_AUTHENTICATOR_SHA256SUM}  aws-iam-authenticator" | sha256sum -c - \
	&& chmod +x aws-iam-authenticator \
	&& mv "aws-iam-authenticator" "/usr/local/bin/aws-iam-authenticator"


ENV KUBECTL_VERSION=v1.11.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c

RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

