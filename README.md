# gshocksrv

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/gshocksrv)
![Build](https://github.com/gonejack/gshocksrv/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/gshocksrv.svg?color=blue)](LICENSE)

[English](#english) | [中文](#中文)

## English

`gshocksrv` is a headless Go service that synchronizes the current system time to compatible Casio G-Shock watches over Bluetooth Low Energy (BLE).

> **Project origin:** This project was developed with reference to [izivkov/GShockTimeServer](https://github.com/izivkov/GShockTimeServer). It reimplements the headless time-server workflow in Go, including the Casio BLE protocol handling and watch-specific synchronization behavior.

### Installation

Go 1.26 or later is required.

```bash
go install github.com/gonejack/gshocksrv@latest
```

Make sure the Go binary directory (usually `$HOME/go/bin`) is included in your `PATH`.

On Linux and Raspberry Pi, install and start BlueZ first:

```bash
sudo apt install bluez
sudo systemctl enable --now bluetooth
```

Alternatively, build the executable from source:

```bash
git clone https://github.com/gonejack/gshocksrv.git
cd gshocksrv
go build -o gshocksrv .
```

### Usage

Start the server and leave it running:

```bash
gshocksrv
```

Then trigger a Bluetooth connection from the watch. Depending on the model, short-press the lower-right button, long-press the lower-left button, or enable automatic time adjustment. Once connected, `gshocksrv` writes the host machine's current local time to the watch.

Common examples:

```bash
# Add one second to the time written to the watch
gshocksrv --fine-adjustment-secs 1

# Show detailed connection logs
gshocksrv --log-level DEBUG

# Store connection state at a custom path
gshocksrv --store-path /var/lib/gshocksrv/state.json --no-color
```

The default state file is `gshock_server_data.json` in the current working directory and records the most recent connection. While the server is running, always-connected models such as the ECB series and DW-H5600 are limited to one accepted connection every six hours.

### Options

```text
Flags:
  -h, --help                      Show context-sensitive help.
      --fine-adjustment-secs=0    Seconds added to watch time (-10..10).
      --scan-timeout=1m           Maximum time for each BLE scan.
      --request-timeout=5s        Maximum time to wait for a watch response.
      --store-path="gshock_server_data.json"
                                  Path to the state file.
      --log-level="INFO"          Log level: DEBUG, INFO, WARN, or ERROR.
      --no-color                  Disable colored log output.
```

### Platform Notes

- **macOS:** On first use, allow your terminal or the installed executable to access Bluetooth in **System Settings > Privacy & Security > Bluetooth**.
- **Linux / Raspberry Pi:** BlueZ must be running, and the user running `gshocksrv` must have permission to access the BlueZ D-Bus service.
- The synchronized time is taken from the host's local time zone. Configure the host's clock and time zone correctly before running the server.

### Supported Behavior

- Standard digital-watch protocol, MTG analog-watch protocol, and the GW-BX5600 MIP synchronization flow.
- Required DST and world-city settings are read and written back before time synchronization.
- Optional time adjustment from `-10` to `10` seconds.
- Continuous scanning and support for multiple compatible watches.

### Acknowledgements

Special thanks to [GShockTimeServer](https://github.com/izivkov/GShockTimeServer), which served as the primary reference for this Go implementation and its Casio watch synchronization behavior.

---

## 中文

`gshocksrv` 是一个无显示界面的 Go 服务，通过低功耗蓝牙（BLE）将当前系统时间同步到兼容的 Casio G-Shock 手表。

> **项目来源：** 本项目参考 [izivkov/GShockTimeServer](https://github.com/izivkov/GShockTimeServer) 开发，以 Go 重新实现了其无显示时间服务器的工作流程，包括 Casio BLE 协议处理和不同手表的校时逻辑。

### 安装

需要 Go 1.26 或更高版本。

```bash
go install github.com/gonejack/gshocksrv@latest
```

请确保 Go 的可执行文件目录（通常为 `$HOME/go/bin`）已经加入 `PATH`。

在 Linux 和 Raspberry Pi 上，需要先安装并启动 BlueZ：

```bash
sudo apt install bluez
sudo systemctl enable --now bluetooth
```

也可以从源码构建可执行文件：

```bash
git clone https://github.com/gonejack/gshocksrv.git
cd gshocksrv
go build -o gshocksrv .
```

### 使用

启动服务并保持运行：

```bash
gshocksrv
```

然后在手表上触发蓝牙连接。具体操作因型号而异，可以短按右下键、长按左下键，或开启自动校时。连接成功后，`gshocksrv` 会把运行机器的当前本地时间写入手表。

常用示例：

```bash
# 在写入手表的时间上增加 1 秒
gshocksrv --fine-adjustment-secs 1

# 显示详细的连接日志
gshocksrv --log-level DEBUG

# 将连接状态保存到指定位置，并关闭彩色日志
gshocksrv --store-path /var/lib/gshocksrv/state.json --no-color
```

默认状态文件为当前工作目录下的 `gshock_server_data.json`，用于记录最近一次连接。服务运行期间，ECB 系列、DW-H5600 等常连接型号每六小时最多接受一次连接。

### 参数

```text
Flags:
  -h, --help                      Show context-sensitive help.
      --fine-adjustment-secs=0    Seconds added to watch time (-10..10).
      --scan-timeout=1m           Maximum time for each BLE scan.
      --request-timeout=5s        Maximum time to wait for a watch response.
      --store-path="gshock_server_data.json"
                                  Path to the state file.
      --log-level="INFO"          Log level: DEBUG, INFO, WARN, or ERROR.
      --no-color                  Disable colored log output.
```

### 平台说明

- **macOS：** 首次使用时，需要在 **系统设置 > 隐私与安全性 > 蓝牙** 中允许终端或已安装的可执行文件访问蓝牙。
- **Linux / Raspberry Pi：** BlueZ 必须保持运行，并且执行 `gshocksrv` 的用户需要有权限访问 BlueZ D-Bus 服务。
- 同步时间取自运行机器的本地时区。启动服务前，请确认机器的系统时间和时区设置正确。

### 支持的功能

- 标准数字表协议、MTG 模拟表协议，以及 GW-BX5600 的 MIP 校时流程。
- 校时前读取并回写必要的 DST 和世界城市设置。
- 支持 `-10` 到 `10` 秒的时间微调。
- 持续扫描并支持多块兼容手表。

### 致谢

特别感谢 [GShockTimeServer](https://github.com/izivkov/GShockTimeServer)。本项目的 Go 实现及 Casio 手表校时逻辑以该项目为主要参考。
