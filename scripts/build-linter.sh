#!/usr/bin/env bash

VERSION="v$(grep "golangci-lint" .tool-versions | awk '{print $2}')"
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin "$VERSION"
