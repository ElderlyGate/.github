# test.yml builds this through the image action, without pushing: both platforms build, the target is picked, and the version argument arrives.
FROM alpine:3 AS other
RUN false

FROM alpine:3 AS run
ARG VERSION=unset
RUN test "$VERSION" = v1.2.3
