#!/bin/sh

set -ex

test -z "$(go fmt $(glide novendor) | tee /dev/stderr)"
test -z "$(for package in $(glide novendor); do golint $package; done | tee /dev/stderr)"
test -z "$(go vet $(glide novendor) 2>&1 | tee /dev/stderr)"
