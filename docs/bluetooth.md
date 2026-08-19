# G-Shock BLE 调用流程

本文说明本项目如何通过 `tinygo.org/x/bluetooth` 发现、连接并操作 Casio G-Shock 手表。重点对应 [internal/gshock/client.go](../internal/gshock/client.go) 和 [internal/gshock/watch.go](../internal/gshock/watch.go) 的实现。

## 总体调用链

```text
main.go
  -> gshock.NewClient(bluetooth.DefaultAdapter, ...)
  -> adapter.Enable()
  -> server.Run()
  -> Client.ScanAndConnect()
       -> Adapter.Scan()
       -> Adapter.Connect()
       -> Device.DiscoverServices()
       -> DeviceService.DiscoverCharacteristics()
       -> DeviceCharacteristic.EnableNotifications()
  -> Watch.PressedButton() / Watch.SetTime()
       -> WriteWithoutResponse() 或 Write()
       -> notification 回调
  -> Device.Disconnect()
```

`main.go` 只负责取得默认适配器并创建客户端：[main.go:61](../main.go#L61)。实际的 BLE 操作由 `Client` 和 `Watch` 完成。

## 1. 选择并启用本机适配器

```go
client, err := gshock.NewClient(
    bluetooth.DefaultAdapter,
    logger,
    a.RequestTimeout,
)
```

`bluetooth.DefaultAdapter` 是当前系统的默认蓝牙适配器，表示本机的 BLE Central，而不是某一块手表。

`NewClient` 会完成两件事：[client.go:151-162](../internal/gshock/client.go#L151-L162)

1. 解析 Casio 服务 UUID。
2. 调用 `adapter.Enable()` 启用本机蓝牙适配器。

`tinygo.org/x/bluetooth` 根据编译平台选择底层实现：

| 平台 | 底层实现 |
| --- | --- |
| macOS | CoreBluetooth |
| Linux | BlueZ，通过 D-Bus |
| Windows | WinRT |
| 部分嵌入式平台 | HCI 或 SoftDevice |

应用代码因此不需要直接调用 CoreBluetooth、BlueZ D-Bus 或 HCI API。

## 2. 扫描 BLE 广播

服务器每轮创建一个带超时的上下文，然后调用 `ScanAndConnect`：[internal/server/server.go:46-50](../internal/server/server.go#L46-L50)

```go
scanCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
watch, err := s.c.ScanAndConnect(scanCtx, s.allowWatch)
```

`ScanAndConnect` 调用适配器扫描：[client.go:34-64](../internal/gshock/client.go#L34-L64)

```go
err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
    if found != nil || !result.HasServiceUUID(c.serviceUUID) {
        return
    }

    name := result.LocalName()
    if allow != nil && !allow(name) {
        return
    }

    found = &candidate{address: result.Address, name: name}
    _ = adapter.StopScan()
})
```

扫描过程是 BLE 广播监听：

1. `Scan` 开始监听附近设备的 advertisement。
2. 每发现一个设备，库调用一次回调函数并传入 `ScanResult`。
3. `HasServiceUUID` 过滤出广播 Casio 服务 UUID 的设备。
4. `allow` 过滤设备名称和连接频率限制。
5. 保存设备地址后立即调用 `StopScan`。

扫描上下文超时时，`context.AfterFunc` 也会调用 `StopScan`：[client.go:40-42](../internal/gshock/client.go#L40-L42)。因此 `ScanTimeout` 同时限制“找设备”的最长时间。

## 3. 连接 BLE 外设

扫描找到设备后，代码使用广播中得到的蓝牙地址连接：[client.go:66-75](../internal/gshock/client.go#L66-L75)

```go
device, err := c.adapter.Connect(
    found.address,
    bluetooth.ConnectionParams{},
)
```

返回的 `bluetooth.Device` 是已经建立连接的 BLE GATT 外设。这里使用的是 BLE GATT 连接，不是传统蓝牙串口（RFCOMM）连接。

如果后续服务或特征发现失败，代码会主动断开这个设备，避免留下半连接状态。

## 4. 发现 GATT 服务和特征

连接成功后，客户端发现设备提供的所有 GATT 服务和特征：[client.go:77-93](../internal/gshock/client.go#L77-L93)

```go
services, err := device.DiscoverServices(nil)

for _, service := range services {
    discovered, err := service.DiscoverCharacteristics(nil)
    // 将特征按小写 UUID 保存
}
```

传入 `nil` 表示不指定过滤条件，发现全部服务或全部特征。GATT 层级为：

```text
Device
  └── Service
        └── Characteristic
```

客户端按 UUID 找到 Casio 协议需要的特征：[client.go:95-102](../internal/gshock/client.go#L95-L102)

| UUID 常量 | 用途 |
| --- | --- |
| `readRequestUUID` | 请求读取手表功能数据 |
| `allFeaturesUUID` | 写入功能数据和当前时间 |
| `spRequestUUID` | MIP 协议请求，部分型号使用 |
| `spDataUUID` | MIP 协议数据，部分型号使用 |

缺少 `readRequestUUID` 或 `allFeaturesUUID` 时，客户端会报告“手表缺少必要特征”，不会继续操作。

## 5. 订阅通知

客户端遍历发现的特征并尝试启用通知：[client.go:124-147](../internal/gshock/client.go#L124-L147)

```go
err := characteristic.EnableNotifications(func(data []byte) {
    w.enqueue(w.notifications, data)
})
```

通知是手表到本机的异步数据通道：

1. `EnableNotifications` 在 GATT 层订阅 characteristic 的 notification 或 indication。
2. 手表产生响应后，底层蓝牙库调用回调函数。
3. `enqueue` 复制数据并放入带缓冲的 Go channel。
4. 请求函数从 channel 中等待自己需要的响应。

MIP 型号的 `spDataUUID` 使用独立的 `spNotifications` channel，避免与普通协议数据混在一起。

如果没有任何特征支持通知，客户端会拒绝这次连接。

## 6. 读取按键状态

服务器连接成功后调用 `watch.PressedButton`：[server.go:86-94](../internal/server/server.go#L86-L94)。该方法向 `readRequest` 特征写入 `featureBLE` 请求，然后等待通知：[watch.go:40-45](../internal/gshock/watch.go#L40-L45)。

请求的核心流程在 [watch.go:66-85](../internal/gshock/watch.go#L66-L85)：

```text
清空旧通知
  -> WriteWithoutResponse(readRequest, request)
  -> 等待 notifications channel
  -> 检查响应 key 是否匹配
  -> 返回解包后的数据
```

写入封装在 [watch.go:87-94](../internal/gshock/watch.go#L87-L94)：

- `WriteWithoutResponse`：BLE 写命令，不等待远端写入确认，适合发送读取请求。
- `Write`：BLE 写请求，带协议层确认，适合写入最终功能数据。

每次等待都有 `requestTimeout` 限制，防止手表不响应时永久阻塞。

## 7. 设置时间

服务器确认按键有效后调用 `watch.SetTime`：[server.go:95-102](../internal/server/server.go#L95-L102)。

普通协议在 [watch.standard.go](../internal/gshock/watch.standard.go) 中执行：

1. 读取并回写 DST 状态。
2. 读取并回写世界城市或本地时间配置。
3. 将当前时间编码成 Casio 协议数据。
4. 使用 `Write(allFeatures, data)` 写入手表。

普通协议的请求-响应-回写操作由 `roundTrip` 完成：[watch.go:56-64](../internal/gshock/watch.go#L56-L64)。

MIP 型号使用 `spRequest`/`spData` 特征执行多步数据交换，最后仍通过 `allFeatures` 写入时间，见 [watch.mip.go](../internal/gshock/watch.mip.go)。

## 8. 断开连接

处理完成后，普通型号由服务器延迟调用：

```go
watch.Disconnect()
```

最终对应 `w.device.Disconnect()`：[watch.go:53-55](../internal/gshock/watch.go#L53-L55)。标记为 `AlwaysConnected` 的型号不会在每次处理后主动断开。

## 运行环境要求

- macOS：需要在“系统设置 -> 隐私与安全性 -> 蓝牙”允许终端或应用访问蓝牙。
- Linux：需要安装并运行 BlueZ，并确保运行用户有访问 BlueZ D-Bus 的权限。
- 所有平台：手表必须以 BLE 外设身份广播，并提供项目使用的 Casio GATT 特征。

因此，`tinygo.org/x/bluetooth` 在本项目中的职责是提供跨平台的 BLE Adapter、扫描、连接和 GATT API；Casio 特征 UUID、数据包格式、请求顺序和时间编码则由 `internal/gshock` 自己实现。
