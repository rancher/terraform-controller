FROM alpine
RUN apk add --no-cache bash git
COPY terraform-operator /usr/bin/
ENTRYPOINT ["/usr/bin/terraform-operator"]
