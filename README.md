# gshocksrv

`gshocksrv` 是 [GShockTimeServer](https://github.com/izivkov/GShockTimeServer) 无显示版本的 Go 实现。它持续扫描通过 BLE 发起连接的 Casio 手表，读取连接按钮并写入当前系统时间。

## 功能

- 保留 Python 版 `--fine-adjustment-secs` 的 `-10..10` 秒微调。
- 支持标准数字表协议、MTG 模拟表协议，以及 GW-BX5600 的 MIP 校时流程。
- 在校时前读取并回写 DST、世界城市等必要配置。
- ECB、DW-H5600 等常连接型号每 6 小时最多接受一次连接。
- 将最近一次连接写入 `gshock_server_data.json`。

## 运行

需要 Go 1.24 或更高版本。Linux/Raspberry Pi 还需要 BlueZ：

```bash
sudo apt install bluez
go run .
```

手表短按右下键、长按左下键，或启用自动校时后会主动连接。时间以运行机器的本地时区为准。

常用参数：

```bash
go run . --fine-adjustment-secs 1 --log-level DEBUG
go run . --no-color
go run . --help
```

构建与测试：

```bash
go test ./...
go build -o gshocksrv .
```

macOS 首次运行需在“系统设置 -> 隐私与安全性 -> 蓝牙”中允许终端访问蓝牙。Linux 服务账号需要有访问 BlueZ D-Bus 的权限。
