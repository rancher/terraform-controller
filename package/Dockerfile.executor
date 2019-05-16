FROM alpine

# Need to grab terraform binary

RUN apk add --no-cache curl git openssh unzip
# This is the real url we will eventually need to pull the zip from
# https://releases.hashicorp.com/terraform/0.11.11/terraform_0.11.11_linux_amd64.zip
RUN curl -sLf https://github.com/dramich/terraform/releases/download/testing/linux_amd64.zip -o terraform_0.11.11_linux_amd64.zip && \
    unzip terraform_0.11.11_linux_amd64.zip -d /usr/bin && \
    chmod +x /usr/bin/terraform && \
    rm terraform_0.11.11_linux_amd64.zip

COPY terraform-executor /usr/bin/

RUN mkdir -p /root/module
WORKDIR /root/module

CMD ["terraform-executor"]
