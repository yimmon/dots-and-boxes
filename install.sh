#! /bin/bash

CURDIR=`pwd`
OLDGOPATH=$GOPATH
export GOPATH=$CURDIR
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

echo "Installing algorithm/board"
go install algorithm/board
echo "Installing algorithm/qboard"
go install algorithm/qboard
echo "Installing algorithm/uct"
go install algorithm/uct
echo "Installing algorithm/uctann"
go install algorithm/uctann
echo "Installing algorithm/quct"
go install algorithm/quct
echo "Installing algorithm/quctann"
go install algorithm/quctann
echo "Installing algorithm/alphabeta"
go install algorithm/alphabeta
echo "Installing server"
go install server
echo "Installing ann"
go install ann
echo "Installing ann/rcds"
go install ann/rcds
echo "Installing ann/ds"
go install ann/ds
echo "Installing ann/tr"
go install ann/tr
echo "Installing ann/test-models"
go install ann/test-models
echo "Installing ann/eval"
go install ann/eval
echo "Installing misc/battle"
go install misc/battle
echo "Installing misc/merger"
go install misc/merger
echo "Installing misc/examer"
go install misc/examer
echo "Installing misc/balance"
go install misc/balance
echo "Installing ann/test-single"
go install misc/test-single
echo "Installing ann/test-info"
go install misc/test-info
echo "Installing misc/test-whowillwin"
go install misc/test-whowillwin
echo "Installing misc/test-qboard"
go install misc/test-qboard
echo "Installing misc/test-sim"
go install misc/test-sim

export GOPATH=$OLDGOPATH
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
echo "Finished"

