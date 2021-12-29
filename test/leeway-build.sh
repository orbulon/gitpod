#!/bin/bash
# Copyright (c) 2021 Gitpod GmbH. All rights reserved.
# Licensed under the GNU Affero General Public License (AGPL).
# See License-AGPL.txt in the project root for license information.

export CGO_ENABLED=0

# Compile integration tests

mkdir -p bin/integration

for AGENT in pkg/agent/*; do
    echo building agent "$AGENT"
    base=$(basename "$AGENT")
    go build -trimpath -ldflags="-buildid= -w -s" -o bin/integration/gitpod-integration-test-"${base%_agent}"-agent ./"$AGENT"
done

for COMPONENT in tests/components/*; do
    echo building test "$COMPONENT"
    OUTPUT=$(basename "$COMPONENT")
    go test -trimpath -ldflags="-buildid= -w -s" -c -o bin/integration/"$OUTPUT".test ./"$COMPONENT"
done

go test -trimpath -ldflags="-buildid= -w -s" -o bin/integration/workspace.test -c ./tests/workspace

# Ide integration test run separately so compile them in a different folder

mkdir -p bin/ide-integration

cp bin/integration/gitpod-integration-test-*-agent bin/ide-integration/

go test -trimpath -ldflags="-buildid= -w -s" -o bin/ide-integration/ide.test -c ./tests/ide
