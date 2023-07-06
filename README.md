# periph_board modular board component

*periph_board* is a Viam modular board component that an periph.io based GPIO interface for single board computers.

## Prerequisites

``` bash
sudo apt update && sudo apt upgrade -y
sudo apt-get install python3
sudo apt install python3-pip
```

## API

The periph_board resource fulfills the [Viam board API](https://github.com/viamrobotics/api/blob/main/proto/viam/component/board/v1/board.proto).

## Viam Component Configuration

### Install module

This module should first be installed as follows:

### Configure component

After installing the module, configure the component as type *board*, model *viamlabs:board:periph*.
