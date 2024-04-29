# This dockerfile acts as the development environment when starting
# up a new environment via GitPod. It allows us to manage environment
# in line with all the dependencies required for the project.
FROM gitpod/workspace-go

# The link to the go download. Get it from https://go.dev/dl/
ARG GO_DOWNLOAD=go1.22.2.linux-amd64.tar.gz

RUN rm -rf /home/gitpod/go && \
    wget https://go.dev/dl/${GO_DOWNLOAD} && \
    tar -C /home/gitpod -xzf "${GO_DOWNLOAD}" && \
    rm ${GO_DOWNLOAD}

# Normally `sudo` isn't required in a dockerfile, but the gitpod workspace requires it at build time.
# This also installs a pinned version of TF to match the version used in CI/CD, which is 1.4.0
RUN sudo apt-get update && sudo apt-get install -y gnupg software-properties-common && \
    wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list && \
    sudo apt update && \
    sudo apt-get install terraform=1.4.0
