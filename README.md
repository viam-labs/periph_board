# periph_board modular board component

*periph_board* is a Viam modular board component that an periph.io based GPIO interface for single board computers.

## Build

On the single board computer that will be running viam-server and the periph_board module, run these commands to install prerequisites and build:

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

This will create an executable named `periph_board`.

Note that if you want to use a different version of Go, you will need to change the go archive paths.
If you are using another shell like zsh you will need to change ~/.bashrc to ~/.zshrc (or similar).

Also see [examples](#viam-server-board-setup-examples) of installing viam-server on various SBCs.

## API

The periph_board resource fulfills the [Viam board API](https://github.com/viamrobotics/api/blob/main/proto/viam/component/board/v1/board.proto).

## Viam configuration

### Configure module

This module should first be configured as follows:

In the Viam app, go to Config->Modules and add a new module.
Choose any `Name` you'd like, and for `Executable path` add the path to the executable you generated in the [Build step](#build) (for example, */home/username/periph_board/periph_board*).

Make sure to `Save config` at the bottom of the screen.

If you prefer to edit raw configuration JSON, make sure inside the main block your `modules` config looks something like this:

``` json
{
  "modules": [
    {
      "executable_path": "/home/username/periph_board/periph_board",
      "name": "periph"
    }
  ],
  "components": []
}
```

### Configure component

After configuring the module, configure the component as type *board*, model *viamlabs:board:periph*.
You may choose any `Name` you want.

If you prefer to edit raw configuration JSON, make sure inside the main block your `components` config looks something like this:

``` json
{
  "modules": [
    {
      "executable_path": "/home/username/periph_board/periph_board",
      "name": "periph"
    }
  ],
  "components": [
    {
      "name": "local",
      "type": "board",
      "model": "viamlabs:board:periph",
      "attributes": {},
      "depends_on": []
    }
  ]
}
```

## viam-server board setup examples

### Orange Pi Zero 2

The [Orange Pi Zero 2](http://www.orangepi.org/html/hardWare/computerAndMicrocontrollers/details/Orange-Pi-Zero-2.html) is an open-source SBC with dual-band wireless WiFi and Bluetooth 5.0 .It is highly compact with a dimension of 53X60mm. It can run Android10, Ubuntu, Debian.

Start by downloading the latest Linux distribution from OrangePi.
We tested and had success with the [Debian bullseye distribution](https://drive.google.com/drive/folders/1Xk7b1jOMg-rftowFLExynLg0CyuQ7kCM).

Flash this image to an SD card with [Balena Etcher](https://etcher.balena.io/)

Then insert the flashed SD card to the Orange Pi, plug the Pi into a monitor with an mini-HDMI cable, and plug in the Pi to with a 5v power supply.
Log in with the credentials root/orangepi.

Secure the Pi:

- Change the root password by issuing the `passwd` command.
- Change the password for the `orangepi` user account by issuing the commmand `passwd orangepi`

Update the software:

``` bash
apt update
apt upgrade
```

Enable wifi:

```bash
nmcli dev wifi
nmcli dev wifi connect <wifi_name> password <wifi_passwd>
```

Enable the host being announced on the network by running `orangepi-config` and going to System->Avahi and *Enable system announce in the network*.

While still in `orangepi-config`, change the hostname to your preference by going to Personal->Hostname.

Enable i2c and spi by editing `/boot/orangepiEnv.txt`, adding the line:

``` bash
overlays=i2c3 spi-spidev
param_spidev_spi_bus=1
param_spidev_spi_cs=1
```

Now install the Orange Pi gpio tool:

``` bash
git clone https://github.com/orangepi-xunlong/wiringOP
cd wiringOP
sudo ./build clean
sudo ./build
gpio readall
```

At this point, you can unplug the HDMI cable and restart.
You should now be able to connect to the Pi headless via `ssh <user>@<hostname>.local` and [install viam-server](https://docs.viam.com/installation/#install-viam-server)

### Rock Pi S

The [Rock Pi S](https://wiki.radxa.com/RockpiS) is a Rockchip RK3308 based SBC(Single Board Computer) by Radxa.
It equips a 64bits quad core processor, USB, ethernet, wireless connectivity and voice detection engine at the size of 1.7 inch, makes it perfect for IoT and voice applications.

Start by downloading the latest Armbian distribution for the RockPi S.
We tested and had success with the [Bookwork distribution](https://www.armbian.com/rockpi-s/).

Flash this image to an SD card with [Balena Etcher](https://etcher.balena.io/)

Then insert the flashed SD card to the Rock Pi, plug the Pi into a router with an ethernet cable, and plug in the Pi to with a 5v power supply.
Find the IP address (we looked at devices attached to router), and ssh to that IP as the user `root`.
Log in with the password *1234*.

Secure the Pi:

- Change the root password by issuing the `passwd` command.
- Create a user account by issuing `sudo useradd -m <username>` and change the password: `passwd <username>`

Update the software:

``` bash
apt update
apt upgrade
```

Enable wifi:

```bash
nmcli dev wifi
nmcli dev wifi connect <wifi_name> password <wifi_passwd>
```

Enable the host being announced on the network by running `rockpi-config` and going to System->Avahi and *Enable system announce in the network*.

While still in `rockpi-config`, change the hostname to your preference by going to Personal->Hostname.

At this point, you can unplug the network cable and restart.
You should now be able to connect headless to the Pi via `ssh <user>@<hostname>.local` and [install viam-server](https://docs.viam.com/installation/#install-viam-server)
