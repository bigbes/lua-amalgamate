# Dockerfile for lua-amalgamate (multi‑platform)
# GoReleaser will place the built binary in $TARGETPLATFORM/ directory

FROM scratch

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/lua-amalgamate /usr/local/bin/lua-amalgamate

ENTRYPOINT ["lua-amalgamate"]