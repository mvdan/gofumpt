#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '2' ]
then
	env
	set -x
elif [ "$verbose" -gt '1' ]
then
	set -x
fi

set -e -f -u

log() {
	if [ "$verbose" -gt '0' ]
	then
		# Don't use quotes to get word splitting.
		echo "$1" 1>&2
	fi
}

log 'starting to build gofumpt release'

version="${VERSION:-}"
readonly version

log "version '$version'"

dist="${DIST_DIR:-build}"
readonly dist

out="${OUT:-gofumpt}"

log "checking tools"

for tool in tar zip
do
	if ! command -v "$tool" > /dev/null
	then
		log "tool '$tool' not found"

		exit 1
	fi
done

# Data section.  Arrange data into space-separated tables for read -r to read.
# Use 0 for missing values.

#    os  arch      arm mips
platforms="\
darwin   amd64     0   0
darwin   arm64     0   0
freebsd  386       0   0
freebsd  amd64     0   0
freebsd  arm       5   0
freebsd  arm       6   0
freebsd  arm       7   0
freebsd  arm64     0   0
linux    386       0   0
linux    amd64     0   0
linux    arm       5   0
linux    arm       6   0
linux    arm       7   0
linux    arm64     0   0
linux    mips      0   softfloat
linux    mips64    0   softfloat
linux    mips64le  0   softfloat
linux    mipsle    0   softfloat
linux    ppc64le   0   0
openbsd  amd64     0   0
openbsd  arm64     0   0
windows  386       0   0
windows  amd64     0   0
windows  arm64     0   0"
readonly platforms

build() {
	# Get the arguments.  Here and below, use the "build_" prefix for all
	# variables local to function build.
	build_dir="${dist}/${1}"\
		build_name="$1"\
		build_os="$2"\
		build_arch="$3"\
		build_arm="$4"\
		build_mips="$5"\
		;

	# Use the ".exe" filename extension if we build a Windows release.
	if [ "$build_os" = 'windows' ]
	then
		build_output="./${build_dir}/${out}.exe"
	else
		build_output="./${build_dir}/${out}"
	fi

	mkdir -p "./${build_dir}"

	# Build the binary.
	#
	# Set GOARM and GOMIPS to an empty string if $build_arm and $build_mips
	# are zero by removing the zero as if it's a prefix.
	#
	# Don't use quotes with $build_par because we want an empty space if
	# parallelism wasn't set.
	env\
		GOARCH="$build_arch"\
		GOARM="${build_arm#0}"\
		GOMIPS="${build_mips#0}"\
		GOOS="$os"\
		VERBOSE="$(( verbose - 1 ))"\
		VERSION="$version"\
		OUT="$build_output"\
		sh ./scripts/go-build.sh\
		;

	log "$build_output"

	# Prepare the build directory for archiving.
	cp ./LICENSE ./LICENSE.google ./README.md "$build_dir"

	# Make archives.  Windows prefers ZIP archives; the rest, gzipped tarballs.
	case "$build_os"
	in
	('windows')
		build_archive="./${dist}/${out}-${build_name}-${version}.zip"
		# TODO(a.garipov): Find an option similar to the -C option of tar for
		# zip.
		( cd "${dist}" && zip -9 -q -r "../${build_archive}" "./${build_name}" )
		;;
	(*)
		build_archive="./${dist}/${out}-${build_name}-${version}.tar.gz"
		tar -C "./${dist}" -c -f - "./${build_name}" | gzip -9 - > "$build_archive"
		;;
	esac

	log "$build_archive"
}

log "starting builds"

# Go over all platforms defined in the space-separated table above, tweak the
# values where necessary, and feed to build.
echo "$platforms" | while read -r os arch arm mips
do
	case "$arch"
	in
	(arm)
		name="${os}-${arch}${arm}"
		;;
	(*)
		name="${os}-${arch}"
		;;
	esac

	build "$name" "$os" "$arch" "$arm" "$mips"
done

log "finished"
