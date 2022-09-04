#!/usr/bin/env bash

function get_os() {
  unameOut="$(uname -s)"
  case "${unameOut}" in
  Darwin*)
    echo -n "macos"
    ;;
  Linux*)
    echo -n "linux"
    ;;
  CYGWIN*)
    echo -n "cygwin"
    ;;
  *)
    echo "Cannot detect your operating system so have to bail. Bye!"
    exit 1
    ;;
  esac
}

function check_env() {
  which $1 >/dev/null
  if [ $? -ne 0 ]; then
    echo "**** required program to run build.sh missing: $1"
    exit 1
  fi
}

check_env 'which'
check_env 'realpath'
check_env 'dirname'
check_env 'find'
check_env 'touch'
check_env 'sed'
check_env 'find'
check_env 'xargs'


os="$(get_os)"
bashver="${BASH_VERSION:0:1}"
docker_out="docker_out"

if ! [[ "$bashver" =~ ^[4-9]$ ]]; then
  echo 'The version of `bash` you are running is outdated. Upgrade and try again.'
  echo "Current version is: ${BASH_VERSION}"
  echo "Bash version of 5 or higher is required"
  echo
  case "${os}" in
  linux)
    echo "Use your Linux package manager to upgrade bash."
    ;;
  macos)
    echo 'Use `brew` to install/upgrade bash.'
    ;;
  cygwin)
    echo "CygWin detected - google for how to upgrades."
    ;;
  *)
    echo "Unknown OS so can't help with instructions for how to upgrade"
    ;;
  esac
  exit 1
fi

script_dir=$(dirname "$(realpath -e "$0")")
cwd="$(echo "$(pwd)")"
function cleanup() {
  cd "$cwd"
}
trap cleanup EXIT
cd "$script_dir"

function usage() {
  echo "Usage: build.sh [-h|--help] [-c|--clean] [-C|--clean-all]"
  echo "                            [-b|--build] [-d|--distribution]"
  echo '    Build Jokk'
  echo
  echo "Arguments:"
  echo "  -h|--help               This help text"
  echo '  -c|--clean              Clean generated artifacts'
  echo "  -C|--clean-all          Clean all artifacts and the Go module cache"
  echo "  -b|--build              Build using local tooling"
  echo "  -d|--distribution       Create a tarball distribution"
}

clean=0
clean_all=0
build=0
build_distribution=0

while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
  -h | --help)
    usage
    exit 0
    ;;
  -c | --clean)
    clean=true
    shift
    ;;
  -C | --clean-all)
    clean_all=true
    shift
    ;;
  -b | --build)
    build=true
    shift    
    ;;
  -d | --distribution)
    build_distribution=true
    shift
    ;;
  *)
    echo "Unknown argument $1"
    echo
    usage
    exit 1
    ;;
  esac
done

uid=$(id -u "${USER}")
gid=$(id -g "${USER}")

dist="distribution"
dist_exec="${dist}/exec"
dist_out="distribution-out"

function go_cmd () {
  out="$1"
  envvar="$2"
  eval "$envvar go build -o '$out' ."
}

go_used="Building with local go: $(which go)"
build_cmd="go_cmd"

if [ "$clean_all" = true ]; then
  echo "Deep cleaning..."
  clean=true
  go clean --modcache
fi

if [ "$clean" = true ]; then
  echo "Regular cleaning..."
	rm -fr ./dist/exec/jokk-*
	rm -fr "./${dist_out}"
	rm -f jokk
	go clean .
fi

if [ "$build" = true ]; then
  echo "Building..."
  echo "$go_used"
	$build_cmd "jokk"
fi

if [ "$build_distribution" = true ]; then
  echo "Creating distribution..."
  echo "$go_used"
	echo "Compiling for MacOS"
	$build_cmd "${dist_exec}/jokk-macos-amd64" 'CGO_ENABLED=0 GOOS=darwin GOARCH=amd64'
	echo "Compiling for Linux"
	$build_cmd "${dist_exec}/jokk-linux-amd64" 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64'
	echo "Compiling for Windows"
	$build_cmd "${dist_exec}/jokk-windows-amd64.exe" 'CGO_ENABLED=0 GOOS=windows GOARCH=amd64'
  $build_cmd "${dist_exec}/jokk-windows-386.exe" 'CGO_ENABLED=0 GOOS=windows GOARCH=386'
	rm -fr "${dist_out}"
	mkdir -p "${dist_out}"
	cd "${dist_out}"; $tar_cmd
fi