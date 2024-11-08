# Use the docker container proxy so we don't hit docker hub limits
ARG DOCKER_REGISTRY_IMAGE_PREFIX=""
ARG GO_VERSION="1.23.3"
FROM ${DOCKER_REGISTRY_IMAGE_PREFIX}golang:${GO_VERSION}

# Install docker and compose, as we need that for our CI/CD and image downloads
RUN   apt update -qq && \
      apt install -qq --yes --no-install-recommends ca-certificates curl gnupg lsb-release unzip && \
      mkdir -p /etc/apt/keyrings && \
      curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
      echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
        $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
      apt update -qq && \
      apt install -qq --yes --no-install-recommends docker-ce docker-ce-cli containerd.io docker-compose-plugin && \
      echo 'docker compose "$@"' > /bin/docker-compose && \
      chmod +x /bin/docker-compose

# Install Terraform for the builds, pinned to a specific version
# This version is chosen as the minimum version required for the provider because without
# https://github.com/hashicorp/terraform/pull/32551 , the provider produces inconsistent 
# plan operations for many complex objects
ARG TERRAFORM_VERSION="1.4.0"
RUN  curl -o terraform.zip https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
     unzip ./terraform.zip && \
     mkdir -p /usr/bin && \
     mv ./terraform /usr/bin/terraform && \
     rm ./terraform.zip


