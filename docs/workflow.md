# G-Shock BLE Workflow

[English](#english) | [中文](#中文)

## English

This document explains how the project uses `tinygo.org/x/bluetooth` to discover, connect to, and operate Casio G-Shock watches. The main implementations are in [gshock/client.go](../gshock/client.go) and [gshock/watch.go](../gshock/watch.go).

### Overall Call Flow

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
       -> WriteWithoutResponse() or Write()
       -> notification callback
  -> Device.Disconnect()
```

`main.go` only obtains the default adapter and creates the client: [main.go:61](../main.go#L61). The `Client` and `Watch` types perform the actual BLE operations.

### 1. Select and Enable the Local Adapter

```go
client, err := gshock.NewClient(
    bluetooth.DefaultAdapter,
    logger,
    a.RequestTimeout,
)
```

`bluetooth.DefaultAdapter` is the host system's default Bluetooth adapter. It represents the local BLE Central, not a specific watch.

`NewClient` performs two tasks: [client.go:154-165](../gshock/client.go#L154-L165)

1. Parses the Casio service UUID.
2. Calls `adapter.Enable()` to enable the local Bluetooth adapter.

`tinygo.org/x/bluetooth` selects its underlying implementation for the target platform:

| Platform | Implementation |
| --- | --- |
| macOS | CoreBluetooth |
| Linux | BlueZ over D-Bus |
| Windows | WinRT |
| Some embedded platforms | HCI or SoftDevice |

The application therefore does not need to call the CoreBluetooth, BlueZ D-Bus, or HCI APIs directly.

### 2. Scan BLE Advertisements

For each scan, the server creates a context with a timeout and calls `ScanAndConnect`: [server/server.go:48-50](../server/server.go#L48-L50)

```go
scanCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
watch, err := s.c.ScanAndConnect(scanCtx, s.acceptWatch)
```

`ScanAndConnect` starts a scan on the adapter: [client.go:34-78](../gshock/client.go#L34-L78)

```go
err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
    if found != nil || !result.HasServiceUUID(c.serviceUUID) {
        return
    }

    name := result.LocalName()
    if accept != nil && !accept(name) {
        return
    }

    found = &candidate{address: result.Address, name: name}
    _ = adapter.StopScan()
})
```

The scan listens for BLE advertisements:

1. `Scan` starts listening for nearby advertisements.
2. The library invokes the callback with a `ScanResult` for each discovered device.
3. `HasServiceUUID` filters for devices advertising the Casio service UUID.
4. `accept` filters device names and enforces connection-rate limits.
5. After saving the device address, the callback immediately calls `StopScan`.

When the scan context expires, `context.AfterFunc` also calls `StopScan`: [client.go:40-42](../gshock/client.go#L40-L42). `ScanTimeout` therefore limits how long the server waits to find a watch.

### 3. Connect to the BLE Peripheral

After discovering a watch, the client connects using the Bluetooth address from its advertisement: [client.go:69-76](../gshock/client.go#L69-L76)

```go
device, err := c.adapter.Connect(
    found.address,
    bluetooth.ConnectionParams{},
)
```

The returned `bluetooth.Device` represents a connected BLE GATT peripheral. This is a BLE GATT connection, not a classic Bluetooth serial (RFCOMM) connection.

If service or characteristic discovery subsequently fails, the client disconnects the device to avoid leaving a partial connection open.

### 4. Discover GATT Services and Characteristics

After connecting, the client discovers all GATT services and characteristics exposed by the watch: [client.go:81-96](../gshock/client.go#L81-L96)

```go
services, err := device.DiscoverServices(nil)

for _, service := range services {
    discovered, err := service.DiscoverCharacteristics(nil)
    // Store each characteristic by its lowercase UUID.
}
```

Passing `nil` applies no filter, so all services or characteristics are discovered. The GATT hierarchy is:

```text
Device
  -> Service
       -> Characteristic
```

The client locates the characteristics required by the Casio protocol using their UUIDs: [client.go:98-125](../gshock/client.go#L98-L125)

| UUID constant | Purpose |
| --- | --- |
| `readRequestUUID` | Requests watch feature data |
| `allFeaturesUUID` | Writes feature data and the current time |
| `spRequestUUID` | Sends MIP protocol requests for supported models |
| `spDataUUID` | Exchanges MIP protocol data for supported models |

If `readRequestUUID` or `allFeaturesUUID` is absent, the client reports that the watch lacks a required characteristic and stops processing it.

### 5. Subscribe to Notifications

The client iterates over the discovered characteristics and attempts to enable notifications: [client.go:127-150](../gshock/client.go#L127-L150)

```go
err := characteristic.EnableNotifications(func(data []byte) {
    w.enqueue(w.notifications, data)
})
```

Notifications provide an asynchronous data channel from the watch to the host:

1. `EnableNotifications` subscribes to GATT notifications or indications on a characteristic.
2. When the watch sends a response, the Bluetooth library invokes the callback.
3. `enqueue` copies the data into a buffered Go channel.
4. The requesting function waits on that channel for the expected response.

For MIP models, `spDataUUID` uses a separate `spNotifications` channel so that MIP data is not mixed with standard protocol data.

The client rejects the connection if none of the characteristics supports notifications.

### 6. Read the Pressed Button

After connecting, the server calls `watch.PressedButton`: [server.go:86-94](../server/server.go#L86-L94). This method writes a `featureBLE` request to the `readRequest` characteristic and waits for a notification: [watch.go:40-45](../gshock/watch.go#L40-L45).

The request flow is implemented in [watch.go:66-85](../gshock/watch.go#L66-L85):

```text
drain stale notifications
  -> WriteWithoutResponse(readRequest, request)
  -> wait on the notifications channel
  -> verify that the response key matches
  -> return the unwrapped data
```

Writes are wrapped by [watch.go:87-94](../gshock/watch.go#L87-L94):

- `WriteWithoutResponse` sends a BLE write command without waiting for a remote write acknowledgement and is used for read requests.
- `Write` sends a BLE write request with protocol-level acknowledgement and is used for final feature data.

Every wait uses `requestTimeout`, preventing an unresponsive watch from blocking the server indefinitely.

### 7. Set the Time

After confirming that the pressed button is supported, the server calls `watch.SetTime`: [server.go:95-102](../server/server.go#L95-L102).

The standard protocol is implemented in [watch.standard.go](../gshock/watch.standard.go):

1. Read and write back the DST state.
2. Read and write back the world-city or home-time configuration.
3. Encode the current time in the Casio protocol format.
4. Write it with `Write(allFeatures, data)`.

For the standard protocol, `roundTrip` handles each request-response-write-back operation: [watch.go:56-64](../gshock/watch.go#L56-L64).

MIP models exchange data in several steps through the `spRequest` and `spData` characteristics, then write the time through `allFeatures`; see [watch.mip.go](../gshock/watch.mip.go).

### 8. Disconnect

After processing a standard model, the server defers this call:

```go
watch.Disconnect()
```

This ultimately calls `w.device.Disconnect()`: [watch.go:53-55](../gshock/watch.go#L53-L55). Models marked `AlwaysConnected` are not actively disconnected after each operation.

### Runtime Requirements

- **macOS:** Allow the terminal or application to access Bluetooth in **System Settings > Privacy & Security > Bluetooth**.
- **Linux:** Install and run BlueZ, and ensure the user running the server can access the BlueZ D-Bus service.
- **All platforms:** The watch must advertise itself as a BLE peripheral and expose the Casio GATT characteristics used by this project.

In this project, `tinygo.org/x/bluetooth` provides the cross-platform BLE adapter, scanning, connection, and GATT APIs. The Casio characteristic UUIDs, packet formats, request sequence, and time encoding are implemented by `gshock`.

---

## 中文

本文说明本项目如何通过 `tinygo.org/x/bluetooth` 发现、连接并操作 Casio G-Shock 手表。重点对应 [gshock/client.go](../gshock/client.go) 和 [gshock/watch.go](../gshock/watch.go) 的实现。

### 总体调用链

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

### 1. 选择并启用本机适配器

```go
client, err := gshock.NewClient(
    bluetooth.DefaultAdapter,
    logger,
    a.RequestTimeout,
)
```

`bluetooth.DefaultAdapter` 是当前系统的默认蓝牙适配器，表示本机的 BLE Central，而不是某一块手表。

`NewClient` 会完成两件事：[client.go:154-165](../gshock/client.go#L154-L165)

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

### 2. 扫描 BLE 广播

服务器每轮创建一个带超时的上下文，然后调用 `ScanAndConnect`：[server/server.go:48-50](../server/server.go#L48-L50)

```go
scanCtx, cancel := context.WithTimeout(ctx, s.cfg.ScanTimeout)
watch, err := s.c.ScanAndConnect(scanCtx, s.acceptWatch)
```

`ScanAndConnect` 调用适配器扫描：[client.go:34-78](../gshock/client.go#L34-L78)

```go
err := c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
    if found != nil || !result.HasServiceUUID(c.serviceUUID) {
        return
    }

    name := result.LocalName()
    if accept != nil && !accept(name) {
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
4. `accept` 过滤设备名称和连接频率限制。
5. 保存设备地址后立即调用 `StopScan`。

扫描上下文超时时，`context.AfterFunc` 也会调用 `StopScan`：[client.go:40-42](../gshock/client.go#L40-L42)。因此 `ScanTimeout` 同时限制“找设备”的最长时间。

### 3. 连接 BLE 外设

扫描找到设备后，代码使用广播中得到的蓝牙地址连接：[client.go:69-76](../gshock/client.go#L69-L76)

```go
device, err := c.adapter.Connect(
    found.address,
    bluetooth.ConnectionParams{},
)
```

返回的 `bluetooth.Device` 是已经建立连接的 BLE GATT 外设。这里使用的是 BLE GATT 连接，不是传统蓝牙串口（RFCOMM）连接。

如果后续服务或特征发现失败，代码会主动断开这个设备，避免留下半连接状态。

### 4. 发现 GATT 服务和特征

连接成功后，客户端发现设备提供的所有 GATT 服务和特征：[client.go:81-96](../gshock/client.go#L81-L96)

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

客户端按 UUID 找到 Casio 协议需要的特征：[client.go:98-125](../gshock/client.go#L98-L125)

| UUID 常量 | 用途 |
| --- | --- |
| `readRequestUUID` | 请求读取手表功能数据 |
| `allFeaturesUUID` | 写入功能数据和当前时间 |
| `spRequestUUID` | MIP 协议请求，部分型号使用 |
| `spDataUUID` | MIP 协议数据，部分型号使用 |

缺少 `readRequestUUID` 或 `allFeaturesUUID` 时，客户端会报告“手表缺少必要特征”，不会继续操作。

### 5. 订阅通知

客户端遍历发现的特征并尝试启用通知：[client.go:127-150](../gshock/client.go#L127-L150)

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

### 6. 读取按键状态

服务器连接成功后调用 `watch.PressedButton`：[server.go:86-94](../server/server.go#L86-L94)。该方法向 `readRequest` 特征写入 `featureBLE` 请求，然后等待通知：[watch.go:40-45](../gshock/watch.go#L40-L45)。

请求的核心流程在 [watch.go:66-85](../gshock/watch.go#L66-L85)：

```text
清空旧通知
  -> WriteWithoutResponse(readRequest, request)
  -> 等待 notifications channel
  -> 检查响应 key 是否匹配
  -> 返回解包后的数据
```

写入封装在 [watch.go:87-94](../gshock/watch.go#L87-L94)：

- `WriteWithoutResponse`：BLE 写命令，不等待远端写入确认，适合发送读取请求。
- `Write`：BLE 写请求，带协议层确认，适合写入最终功能数据。

每次等待都有 `requestTimeout` 限制，防止手表不响应时永久阻塞。

### 7. 设置时间

服务器确认按键有效后调用 `watch.SetTime`：[server.go:95-102](../server/server.go#L95-L102)。

普通协议在 [watch.standard.go](../gshock/watch.standard.go) 中执行：

1. 读取并回写 DST 状态。
2. 读取并回写世界城市或本地时间配置。
3. 将当前时间编码成 Casio 协议数据。
4. 使用 `Write(allFeatures, data)` 写入手表。

普通协议的请求-响应-回写操作由 `roundTrip` 完成：[watch.go:56-64](../gshock/watch.go#L56-L64)。

MIP 型号使用 `spRequest`/`spData` 特征执行多步数据交换，最后仍通过 `allFeatures` 写入时间，见 [watch.mip.go](../gshock/watch.mip.go)。

### 8. 断开连接

处理完成后，普通型号由服务器延迟调用：

```go
watch.Disconnect()
```

最终对应 `w.device.Disconnect()`：[watch.go:53-55](../gshock/watch.go#L53-L55)。标记为 `AlwaysConnected` 的型号不会在每次处理后主动断开。

### 运行环境要求

- macOS：需要在“系统设置 -> 隐私与安全性 -> 蓝牙”允许终端或应用访问蓝牙。
- Linux：需要安装并运行 BlueZ，并确保运行用户有访问 BlueZ D-Bus 的权限。
- 所有平台：手表必须以 BLE 外设身份广播，并提供项目使用的 Casio GATT 特征。

因此，`tinygo.org/x/bluetooth` 在本项目中的职责是提供跨平台的 BLE Adapter、扫描、连接和 GATT API；Casio 特征 UUID、数据包格式、请求顺序和时间编码则由 `gshock` 自己实现。
