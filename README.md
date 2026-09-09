# gshocksrv

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/gshocksrv)
![Build](https://github.com/gonejack/gshocksrv/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/gshocksrv.svg?color=blue)](LICENSE)

[English](#english) | [中文](#中文)

## English

`gshocksrv` is a headless Go service that synchronizes the host's local time to compatible Casio G-Shock watches over
Bluetooth Low Energy (BLE).

> **Project origin:** A Go reimplementation of [izivkov/GShockTimeServer](https://github.com/izivkov/GShockTimeServer),
> covering its headless workflow, Casio BLE protocol, and model-specific synchronization logic.

### Installation

Choose one of the following methods.

#### 1. Download prebuilt (recommended)

Download the matching archive from [Releases](https://github.com/gonejack/gshocksrv/releases), extract it, and place the
executable in your `PATH`.

On macOS, clear the quarantine attribute if the executable cannot be opened:

```bash
xattr -c /path/to/gshocksrv
```

#### 2. Install with `go install`

Requires Go 1.26+.

```bash
go install github.com/gonejack/gshocksrv@latest
```

Ensure the Go binary directory (usually `$HOME/go/bin`) is in your `PATH`.

#### 3. Build with `go build`

Requires Go 1.26+.

```bash
git clone https://github.com/gonejack/gshocksrv.git
cd gshocksrv
go build .
```

### Usage

#### 1. Install BlueZ (Linux / Raspberry Pi only)

Skip this step on macOS.

```bash
sudo apt install bluez
sudo systemctl enable --now bluetooth
```

#### 2. Start the service

```bash
gshocksrv
```

Keep the service running.

#### 3. Connect the watch

Depending on the model, short-press the lower-right button, long-press the lower-left button, or enable automatic time
adjustment. Once connected, the service writes the host's local time to the watch.

#### 4. Set optional parameters

```bash
# Add one second to the synchronized time
gshocksrv --fine-adjustment-secs 1

# Show detailed connection logs
gshocksrv --log-level DEBUG

# Use a custom state file and disable colored logs
gshocksrv --store-path /var/lib/gshocksrv/state.json --no-color
```

The default state file, `gshock_server_data.json` in the current directory, records the latest connection.
Always-connected models such as the ECB series and DW-H5600 accept at most one connection every six hours.

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

- **macOS:** On first use, allow the terminal or executable to use Bluetooth in **System Settings > Privacy & Security >
  Bluetooth**.
- **Linux / Raspberry Pi:** BlueZ must be running, and the user must be allowed to access its D-Bus service.
- The synchronized time is taken from the host, make sure the host's clock and time zone is correct.

### Supported Behavior

- Standard digital, MTG analog, and GW-BX5600 MIP synchronization protocols.
- Reads and writes back required DST and world-city settings before synchronization.
- Time adjustment from `-10` to `10` seconds.
- Continuous scanning for multiple compatible watches.

### Acknowledgements

Thanks to [GShockTimeServer](https://github.com/izivkov/GShockTimeServer), the primary reference for this
implementation.

---

## 中文

`gshocksrv` 是一个无界面的 Go 服务，通过低功耗蓝牙（BLE）将主机本地时间同步到兼容的 Casio G-Shock 手表。

> **项目来源：** 本项目以 Go 重写 [izivkov/GShockTimeServer](https://github.com/izivkov/GShockTimeServer)，涵盖无界面服务流程、Casio
> BLE 协议及型号适配校时逻辑。

### 安装

任选一种安装方式。

#### 1. 下载预编译版本（推荐）

从 [Releases](https://github.com/gonejack/gshocksrv/releases) 下载对应系统和架构的压缩包，解压后将可执行文件放入 `PATH`。

macOS 若无法打开可执行文件，请清除隔离属性：

```bash
xattr -c /path/to/gshocksrv
```

#### 2. 使用 `go install` 安装

需要 Go 1.26+。

```bash
go install github.com/gonejack/gshocksrv@latest
```

确保 Go 可执行文件目录（通常为 `$HOME/go/bin`）已加入 `PATH`。

#### 3. 使用 `go build` 构建

需要 Go 1.26+。

```bash
git clone https://github.com/gonejack/gshocksrv.git
cd gshocksrv
go build .
```

### 使用

#### 1. 安装 BlueZ（仅 Linux / Raspberry Pi）

macOS 可跳过此步骤。

```bash
sudo apt install bluez
sudo systemctl enable --now bluetooth
```

#### 2. 启动服务

```bash
gshocksrv
```

保持服务运行。

#### 3. 连接手表

根据型号短按右下键、长按左下键或开启自动校时，由手表发起连接。连接后，服务将主机本地时间写入手表。

#### 4. 设置可选参数

```bash
# 同步时间增加 1 秒
gshocksrv --fine-adjustment-secs 1

# 显示详细连接日志
gshocksrv --log-level DEBUG

# 指定状态文件并禁用彩色日志
gshocksrv --store-path /var/lib/gshocksrv/state.json --no-color
```

默认状态文件为当前目录下的 `gshock_server_data.json`，用于记录最近一次连接。ECB 系列、DW-H5600 等常连接型号每六小时最多接受一次连接。

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

- **macOS：** 首次使用时，在 **系统设置 > 隐私与安全性 > 蓝牙** 中允许终端或可执行文件访问蓝牙。
- **Linux / Raspberry Pi：** BlueZ 必须运行，且当前用户需有权访问其 D-Bus 服务。
- 同步时间取自运行机器的系统时间，请确认机器的系统时间和时区设置正确。

### 支持的功能

- 标准数字表、MTG 模拟表及 GW-BX5600 MIP 校时协议。
- 校时前读取并回写必要的 DST 和世界城市设置。
- `-10` 至 `10` 秒时间微调。
- 持续扫描多块兼容手表。

### 致谢

感谢 [GShockTimeServer](https://github.com/izivkov/GShockTimeServer) 为本项目提供主要实现参考。
