# periph_board modular board component

*periph_board* is a Viam modular board component that an periph.io based GPIO interface for single board computers.

## Prerequisites

On the single board computer that will be running viam server and the periph_board module, run these commands:

``` bash
sudo apt update && sudo apt upgrade -y
sudo apt-get install git
wget https://go.dev/dl/go1.20.5.linux-arm64.tar.gz
sudo  rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.20.5.linux-arm64.tar.gz
echo 'PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
git clone https://github.com/viam-labs/periph_board.git
cd periph_board
go build
```

Note that if you want to use a different version of Go, you will need to change the go archive paths, and if you are using another shell like zsh you will need to change ~/.bashrc to ~/.zshrc (or similar).

## API

The periph_board resource fulfills the [Viam board API](https://github.com/viamrobotics/api/blob/main/proto/viam/component/board/v1/board.proto).

## Viam Component Configuration

### Install module

This module should first be installed as follows:

### Configure component

After installing the module, configure the component as type *board*, model *viamlabs:board:periph*.
