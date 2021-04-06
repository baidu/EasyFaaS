#!/bin/bash
pushd . > /dev/null
CWD="${BASH_SOURCE[0]}";
while([ -L "${CWD}" ]); do
    cd "`dirname "${CWD}"`"
    CWD="$(readlink "$(basename "${CWD}")")";
done
BUILD_HOME="$(dirname "${CWD}")"

usage () {
    echo "Usage: $0 [COMMAND]"
    echo "Commands:"
    echo "  start               Create and start service"
    echo "  stop                Stop and clean service"
    echo "  restart             Restart service"
    echo "  stats               List Containers"
    exit
}

fstart () {

    source ${BUILD_HOME}/.env
    if [ ! -d "$FUNCTION_DATA_DIR" ]; then
      mkdir -p "$FUNCTION_DATA_DIR"
      chmod 777 "$FUNCTION_DATA_DIR"
    fi

    DOCKER_CONFIG=${BUILD_HOME}/.docker docker-compose --project-directory ${BUILD_HOME} -f ${BUILD_HOME}/docker-compose.yaml -p openless up --force-recreate -d
}

fstop () {
    docker-compose --project-directory ${BUILD_HOME} -f ${BUILD_HOME}/docker-compose.yaml -p openless down --volumes --remove-orphans
}

frestart () {
    fstop
    fstart
}

fstats () {
  echo ${BUILD_HOME}
   docker-compose --project-directory ${BUILD_HOME} -f ${BUILD_HOME}/docker-compose.yaml -p openless ps
}

ACTION=$1

case "X$ACTION" in
    Xstart)
        fstart
        ;;
    Xstop)
        fstop
        ;;
    Xrestart)
        frestart
        ;;
    Xstats)
        fstats
        ;;
    X)
        usage
        ;;
esac
