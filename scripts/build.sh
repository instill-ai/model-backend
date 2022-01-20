#!/bin/bash
set -Eeuo pipefail

# Ensure we start in the parent directory of where this script is (the root folder).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$SCRIPT_DIR/.."
cd "$ROOT_DIR" || exit 1

GITHUB_PREFIX="github.com/instill-ai"
BACKEND_NAME=$(basename "$(git rev-parse --show-toplevel)")
PROJECT_NAME=$GITHUB_PREFIX/$BACKEND_NAME
BACKEND_TYPE="go-gin"

CODE_DIR=$ROOT_DIR
CODEGEN_DIR=$ROOT_DIR/autogen

SERVER_URL="https://api.instill.tech"

GENERATE_ONLY=false
COMPILE=false
TEST=false

USAGE=$(
    cat <<-END
Usage: $0 [-s <server_url>] [-g] [-c]
    -g: To only generate code stubs without customizing them
    -c: To compile the go server
    -t: To run tests
    -s: To specify the server URL that will appear in the OpenAPI spec,
        e.g. https://api.instill.tech or http://localhost:8080 . Default is
        https://api.instill.tech
END
)

while getopts "s:gcth" args; do
    case $args in
    s) SERVER_URL="$OPTARG" ;;
    g) GENERATE_ONLY=true ;;
    c) COMPILE=true ;;
    t) TEST=true ;;
    h)
        echo -e "$USAGE"
        exit 1
        ;;
    [?])
        echo -e "$USAGE"
        exit 1
        ;;
    esac
done

### Generate server stubs from OpenAPI specification
echo -e "Building server stubs..."

# Generate config file and API spec for OpenAPI generator
PUBLIC_FULL_VERSION=$(cat ./version.txt)
IFS=" " read -r -a array <<<"${PUBLIC_FULL_VERSION//./ }" # replace points, split into array
if [ "${#array[@]}" -lt "3" ]; then
    echo -e "Public version $PUBLIC_FULL_VERSION is not in semantic versioning format: X.Y.Z"
    exit 1
fi

PUBLIC_VERSION="${array[0]}.${array[1]}" # extract X.Y and fill into OpenAPI yaml file

VERSION="${PUBLIC_VERSION}" SERVER_URL="$SERVER_URL" mo templates/go-gin/openapi.yaml.tmpl > openapi.yaml

# Check backend folder
[ ! -d "$CODE_DIR" ] && {
    echo -e "Directory $CODE_DIR DOES NOT exist."
    mkdir "$CODE_DIR"
}

# Validate spec
openapi-generator validate -i openapi.yaml

# Generate API code
rm -rf "$CODEGEN_DIR"
if [ $BACKEND_TYPE == "go-gin" ]; then
    JAVA_OPTS='-Dorg.openapitools.codegen.utils.yaml.minimize.quotes=false' openapi-generator generate -i openapi.yaml -o "$CODEGEN_DIR" -g go-gin-server -c templates/go-gin/openapi-generator-config.yaml -t templates/go-gin/openapi-generator-templates
fi

#### Customize the auto-generated code
if [ $GENERATE_ONLY == false ]; then
    echo -e "\nCopy autogen codes to src folder..."

    if [ $BACKEND_TYPE == "go-gin" ]; then
        cd "$CODE_DIR"

        [ ! -f "$CODE_DIR"/go.mod ] && {
            echo -e "go.mod DOES NOT exist"
            go mod init $PROJECT_NAME
        }

        # Create folders
        [ ! -d "$CODE_DIR"/api ] && { mkdir "$CODE_DIR"/api; }
        [ ! -d "$CODE_DIR"/configs ] && { mkdir "$CODE_DIR"/configs; }
        [ ! -d "$CODE_DIR"/cmd/openapi ] && { mkdir -p "$CODE_DIR"/cmd/openapi; }
        [ ! -d "$CODE_DIR"/pkg/routers ] && { mkdir -p "$CODE_DIR"/pkg/routers; }
        [ ! -d "$CODE_DIR"/pkg/controllers ] && { mkdir -p "$CODE_DIR"/pkg/controllers; }
        [ ! -d "$CODE_DIR"/pkg/handlers ] && { mkdir -p "$CODE_DIR"/pkg/handlers; }
        [ ! -d "$CODE_DIR"/pkg/models/openapi ] && { mkdir -p "$CODE_DIR"/pkg/models/openapi; }

        # Copy always-overriten codes
        cp "$CODEGEN_DIR"/api/openapi.yaml "$CODE_DIR"/api
#        cp "$CODEGEN_DIR"/main.go "$CODE_DIR"/cmd/openapi/main.go
        cp "$CODEGEN_DIR"/routers.go "$CODE_DIR"/pkg/routers
        cp "$CODEGEN_DIR"/middlewares.go "$CODE_DIR"/pkg/routers
        cp "$CODEGEN_DIR"/init.go "$CODE_DIR"/pkg/handlers

        # Copy health code
        if [ ! -f "$CODE_DIR"/pkg/routers/health.go ]; then
            cp "$CODEGEN_DIR"/health.go "$CODE_DIR"/pkg/routers/health.go
        else
            echo
            echo -e "The health.go file $CODE_DIR/cmd/openapi/main.go already exists, which cannot be overwritten!"
        fi

        uname_out="$(uname -s)"
        case "${uname_out}" in
        Linux*)
            sed -i "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/cmd/openapi/main.go
            sed -i "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/pkg/routers/routers.go
            sed -i "s#__MAGIC_TRICK_BACKEND_NAME__#$BACKEND_NAME#" "$CODE_DIR"/pkg/routers/middlewares.go
            sed -i "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/pkg/handlers/init.go
            ;;
        Darwin*)
            sed -i '' "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/cmd/openapi/main.go
            sed -i '' "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/pkg/routers/routers.go
            sed -i '' "s#__MAGIC_TRICK_BACKEND_NAME__#$BACKEND_NAME#" "$CODE_DIR"/pkg/routers/middlewares.go
            sed -i '' "s#__MAGIC_TRICK_PROJECT_NAME__#$PROJECT_NAME#" "$CODE_DIR"/pkg/handlers/init.go
            ;;
        esac

        # Copy controller-api  and controller-api-handler auto-gen codes
        for path in "$CODEGEN_DIR"/go/*api_*; do
            filename=${path##*/}
            filename=${filename/api_/}
            if [[ $filename == *"handler"* ]]; then
                # Don't overwrite the implementation of API handlers
                if [ ! -f "$CODE_DIR"/pkg/handlers/"${filename/%_handler.go/.go}" ]; then
                    cp "$path" "$CODE_DIR"/pkg/handlers/"${filename/%_handler.go/.go}"
                else
                    echo
                    echo -e "The API handler file $CODE_DIR/pkg/handlers/${filename/%_handler.go/.go} already exists, which cannot be overwritten!"
                fi
            else
                # Always overwrite the API controllers
                cp "$path" "$CODE_DIR"/pkg/controllers/"$filename"
            fi
        done

        # Copy auto-gen data model codes
        for path in "$CODEGEN_DIR"/go/model_*; do
            filename=${path##*/}
            filename=${filename/model_/}
            # Always overwrite the data models
            cp "$path" "$CODE_DIR"/pkg/models/openapi/"$filename"
        done

        # Copy configuration codes
        if [ ! -f "$CODE_DIR/configs/configs.go" ]; then
            cp "$CODEGEN_DIR"/configs.go "$CODE_DIR/configs/configs.go"
        else
            echo
            echo -e "The congis.go file $CODE_DIR/configs/configs.go already exists, which cannot be overwritten!"
        fi

        if [ ! -f "$CODE_DIR/configs/config.yaml" ]; then
            cp "$CODEGEN_DIR"/config.yaml "$CODE_DIR/configs/config.yaml"
        else
            echo
            echo -e "The config.yaml file $CODE_DIR/configs/config.yaml already exists, which cannot be overwritten!"
        fi

        go mod tidy

        # Cleanup
        cd "$ROOT_DIR"
        rm -rf "$CODEGEN_DIR"
    fi
fi

#### Build go server
if [ "$COMPILE" == true ]; then
    echo -e "\nBuilding go binary..."
    cd "$CODE_DIR"
    go get -d -v ./...
    go build -a -o "$ROOT_DIR/openapi" "$ROOT_DIR/cmd/openapi"
    echo -e "Go binary built at $ROOT_DIR/openapi"
fi

#### Run test
if [ "$TEST" == true ]; then
    cd "$CODE_DIR"
    echo -e "\nRunning go test..."
    go test -v ./...
fi

# Cleanup
rm openapi.yaml

echo -e "\nDone"
