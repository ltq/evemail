#!/bin/bash

########################### CONFIG ########################################

ROOT_SRC=""
ROOT_PKG=""
PKG_NAME="evemail"
PKG_SRC_PATH="github.com/evalgo/evemail"
PLATTFORMS=( "linux_amd64" "darwin_amd64" )

check_var(){
    CHK_VAR=${1}
    SRC=""
    case $CHK_VAR in
	"GOPATH")
	    CHK_VAR=$GOPATH
	    SRC="/src"
	    ;;
	"GOROOT")
	    CHK_VAR=$GOROOT
	    SRC="/src/pkg"
	    ;;
	*)
	    echo "0"
	    exit
    esac
    if [ ! -z "${CHK_VAR}" ];then
	ROOT_SRC=${CHK_VAR}${SRC}
        # check for supported plattforms
	for PLT in "${PLATTFORMS[@]}";do
	    if [ -d "${GOPATH}/pkg/${PLT}" ];then
		ROOT_PKG=${CHK_VAR}"/pkg/${PLT}"
		echo "${ROOT_SRC} ${ROOT_PKG}"
		exit
	    fi
	done
	echo "0"
	exit
    else
	echo "0"
	exit
    fi
}

# check for GOPATH and GOROOT
GO_PATH=( $( check_var "GOPATH" ) )
if [ "${GO_PATH}" == "0" ];then
    GO_ROOT=( $( check_var "GOROOT" ) )
    if [ "${GO_ROOT}" == "0" ];then
	echo "no needed env variable GOPATH or GOROOT was found!"
	exit
    else
	ROOT_SRC=${GO_ROOT[0]}
	ROOT_PKG=${GO_ROOT[1]}
    fi
else
    ROOT_SRC=${GO_PATH[0]}
    ROOT_PKG=${GO_PATH[1]}
fi

PKG_SRC=${ROOT_SRC}/${PKG_SRC_PATH}
PKG_LIB=${ROOT_PKG}/${PKG_SRC_PATH}.a

############################################################################

clean(){
    rm -rfv $PKG_SRC
    rm -rfv  $PKG_LIB
}

copy_files(){
    mkdir -p $EVEPATH/config/github.com/evalgo/evemail
    cp ./files/* $EVEPATH/config/github.com/evalgo/evemail/
}

deploy(){
    clean
    if [ ! -d "${PKG_SRC}" ];then
	mkdir -p $PKG_SRC
    fi
    cp -rfv ./* $PKG_SRC/
    go install -v $PKG_SRC_PATH   
}

start(){
    SERVICE=${1}
    clean
    mkdir -p $PKG_SRC
    cp -rfv ./* $PKG_SRC/
    go install -v $PKG_SRC_PATH   
    case ${SERVICE} in
	"http")
	    deploy
	    go run cmd/${PKG_NAME}-${SERVICE}/main.go
	    ;;
	"rpc")
	    deploy
	    go run cmd/${PKG_NAME}-${SERVICE}/main.go
	    ;;
	*)
	    echo "the given service ${SERVICE} is not supported!"
	    exit
    esac
}

case $1 in
    "start")
	if [ -z "${2}" ];then
	    echo "please specify a service you want to run:"
	    echo "${0} start [http|rpc]"
	    exit
	fi
	copy_files
	start $2
	;;
    "clean")
	clean
	;;
    "deploy")
	deploy
	copy_files
	;;
    *)
	echo "usage:"
	echo "${0} start [http|rpc]"
	echo "${0} deploy"
	echo "${0} clean"
esac
