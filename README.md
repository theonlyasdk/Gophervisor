# Gophervisor

Gophervisor is a graphical frontend for the QEMU machine emulator and virtualization software, written in Go. It provides an interface to configure and launch virtual machines using `qemu-system` and manage virtual hard disk images using `qemu-img`.

## Screenshots

![Main Dashboard](docs/screenshots/main-window.png)

![QEMU Image Creation](docs/screenshots/qemu-img-creation.png)

## Installation

QEMU must be installed on your host system to utilize the full capabilities of Gophervisor.

To build Gophervisor from source, ensure you have the Go toolchain installed, and run:

```sh
go build -o gophervisor .
```

To start the application:

```sh
./gophervisor
```

## Libraries Used

- [Fyne](https://fyne.io/): A cross-platform UI toolkit and application API written in Go.

## License

This project is licensed under the Mozilla Public License 2.0. See [LICENSE](LICENSE) for more information.