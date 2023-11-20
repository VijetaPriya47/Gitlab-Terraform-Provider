# This dockerfile acts as the development environment when starting
# up a new environment via GitPod. It allows us to manage environment
# in line with all the dependencies required for the project.
FROM gitpod/workspace-go

# The link to the go download. Get it from https://go.dev/dl/
ARG GO_DOWNLOAD=go1.21.3.linux-amd64.tar.gz

RUN rm -rf /home/gitpod/go && \
    wget https://go.dev/dl/${GO_DOWNLOAD} && \
    tar -C /home/gitpod -xzf "${GO_DOWNLOAD}" && \
    rm ${GO_DOWNLOAD}
    