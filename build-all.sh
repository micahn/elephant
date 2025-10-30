#!/usr/bin/env bash





PREFIX="$HOME/.local/"
BINDIR="$PREFIX/bin"

PROVIDER_PATH="$HOME/.config/elephant/providers"
# Build configuration
GO_BUILD_FLAGS=(
  -buildvcs=false
  -trimpath
)
exclude=(nirisessions)


install=false
uninstall=false
clean=false

for arg in "$@"; do
  case "$arg" in
    -i|--install) install=true ;;
    -u|--uninstall) uninstall=true ;;
    -c|--clean) clean=true ;;
    *) echo "Unknown argument: $arg"; exit 1 ;;
  esac
done

direxcluded(){
  local -n dirglob=$1
  local -n excludedirs=$2
  local -n result=$3
  result=()
  declare -A excludemap
  for item in "${excludedirs[@]}"; do
    excludemap["$item"]=1
  done

  for item in "${dirglob[@]}"; do
    name=$(basename "$item")
    [[ -z "${excludemap["$name"]}" ]] && result+=("$item")
  done
  unset excludemap
}

# Build elephant
(
  cd "cmd/elephant" || exit 1
  if [ $clean == true ]; then
    go clean
    echo "Cleaned Elephant"
  else
    go build "${GO_BUILD_FLAGS[@]}" -o elephant
    $install && mv elephant "$BINDIR"
  fi
)


echo "$PROVIDER_PATH"
[ -d "$PROVIDER_PATH" ] || mkdir -p "$PROVIDER_PATH"

# Build Providers
GO_BUILD_FLAGS+=(-buildmode=plugin)
declare -a filtered_dirs
providers=(internal/providers/*)
direxcluded providers exclude filtered_dirs
for dir in "${filtered_dirs[@]}"; do
  if [ -d "$dir" ]; then
    (

      name="$(basename "$dir").so"
      cd "$dir" || exit 1
      if [ $clean == true ]; then
        go clean
        echo "Cleaned: $name"
      else
      printf "Building: %s" "$name"
      go build "${GO_BUILD_FLAGS[@]}" || exit 1
      printf " - done"
      $install && mv "$name" "$PROVIDER_PATH" && printf " - installed"
      echo
      fi
    )
  fi
done
unset filtered_dirs

