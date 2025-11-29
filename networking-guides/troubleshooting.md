# 故障排除指南

## 📋 目录

- [基础检查清单](#基础检查清单)
- [已知问题详解](#已知问题详解)
- [网络调试](#网络调试)
- [性能问题](#性能问题)
- [疑难解答](#疑难解答)
- [获取帮助](#获取帮助)

## 基础检查清单

当遇到网络问题时，请按以下顺序检查：

### 1. WSL2 状态检查

```bash
# 检查 WSL2 是否运行
wsl -l -v
# 确保你的发行版是 Running 状态

# 检查网络接口
ip link show eth0
# 应该显示 UP 状态

# 检查 IP 地址
ip addr show eth0 | grep inet
# 应该有 172.x.x.x 的 IP
```

### 2. Docker 状态检查

```bash
# 检查 Docker 服务
sudo systemctl status docker
# 应该显示 active (running)

# 检查 Docker 版本
docker version
# Client 和 Server 都应该有返回

# 尝试运行测试容器
docker run --rm hello-world
# 应该成功执行并输出欢迎信息
```

### 3. Clash Verge 状态检查

在 Windows PowerShell 中：

```powershell
# 检查 Clash 进程
Get-Process | Where-Object {$_.ProcessName -like "*clash*"}

# 检查 TUN 接口
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TUN*"}

# 检查端口监听
netstat -ano | findstr "7890"
```

### 4. 网络连通性检查

```bash
# 测试 WSL2 网络
ping -c 3 1.1.1.1
curl -I http://www.google.com

# 测试 Docker 容器网络
docker run --rm alpine ping -c 3 1.1.1.1
docker run --rm alpine curl -I http://www.google.com

# 测试 DNS 解析
docker run --rm alpine nslookup google.com
docker run --rm alpine getent hosts google.com
```

## 已知问题详解

### 问题 1: Docker pull 提示 "proxyconnect tcp: connection refused"

**详细症状**:
```
Error response from daemon: Get https://registry-1.docker.io/v2/:
proxyconnect tcp: dial tcp 127.0.0.1:7890: connect: connection refused
```

**根本原因**: Docker 容器内使用 127.0.0.1 指代容器本身，而不是 Windows 宿主机。

**诊断步骤**:

```bash
# 1. 检查当前的代理配置
docker run --rm alpine env | grep -i proxy

# 2. 测试从容器访问宿主机的代理
docker run --rm alpine wget -O- http://127.0.0.1:7890
# 应该失败

docker run --rm alpine wget -O- http://host.docker.internal:7890
# 应该成功（如果代理工作正常）
```

**解决方案**:

方法 1: 更新 Docker Desktop 代理设置
```
# 错误的设置
HTTP_PROXY=http://127.0.0.1:7890

# 正确的设置
HTTP_PROXY=http://host.docker.internal:7890
```

方法 2: 修改 systemd 配置
```bash
sudo nano /etc/systemd/system/docker.service.d/proxy.conf

# 确保配置使用 host.docker.internal
[Service]
Environment="HTTP_PROXY=http://host.docker.internal:7890"
Environment="HTTPS_PROXY=http://host.docker.internal:7890"
Environment="NO_PROXY=localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12"
```

方法 3: 直接在环境变量中设置（临时）
```bash
export HTTP_PROXY=http://host.docker.internal:7890
export HTTPS_PROXY=http://host.docker.internal:7890
export NO_PROXY="localhost,127.0.0.1,.docker.internal"
docker pull your-image:tag
```

**验证修复**: 修改后重启 Docker 服务
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 问题 2: Mirrored 模式下 HTTPS 连接失败

**详细症状**:
```bash
# 尝试访问任何 HTTPS 网站
curl https://www.google.com
# 返回: curl: (7) Failed to connect to www.google.com port 443: No route to host

# 或者在 WSL2 中执行 apt update
sudo apt update
# 返回: Err:1 http://archive.ubuntu.com/ubuntu jammy InRelease
#       Cannot initiate the connection to archive.ubuntu.com:80
```

**根本原因**: Clash Verge TUN 模式默认 MTU=9000，与 WSL2 Mirrored 模式不兼容。

**诊断步骤**:

```bash
# 1. 检查当前 MTU
ip link show eth0 | grep mtu
# 输出: eth0: <BROADCAST...> mtu 1500 qdisc...

# 2. 测试大包 ping
ping -c 3 -s 1472 8.8.8.8
# 应该会失败，显示 "Message too long"

# 3. 测试小包 ping
ping -c 3 -s 1400 8.8.8.8
# 应该会成功

# 4. 检查 TUN 接口
# 在 Windows PowerShell 中
Get-NetAdapter | Where-Object {$_.InterfaceAlias -like "*Clash*"} | Get-NetIPConfiguration
# 查看 MTU 值
```

**解决方案**:

方法 1: 临时修改 MTU（立即生效，重启后失效）
```bash
sudo ip link set eth0 mtu 1500
```

方法 2: 创建 systemd 服务（永久生效）
```bash
sudo nano /etc/systemd/system/mtu-fix.service
```

添加以下内容：
```ini
[Unit]
Description=Fix MTU for WSL2 with Clash TUN
After=network-online.target

[Service]
Type=oneshot
ExecStartPre=/bin/sleep 5
ExecStart=/sbin/ip link set eth0 mtu 1500
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

启用并启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable mtu-fix.service
sudo systemctl start mtu-fix.service
```

方法 3: 使用 wsl.conf 自动配置（推荐）
```bash
sudo nano /etc/wsl.conf
```

添加：
```ini
[network]
command = ip link set eth0 mtu 1500
```

重启 WSL:
```powershell
wsl --shutdown
```

**验证修复**:
测试 HTTPS 连接
```bash
curl https://www.google.com
# 应该返回内容而不是错误
```

### 问题 3: DNS 解析超时或失败

**详细症状**:
```bash
# 容器内 DNS 超时
docker run --rm alpine nslookup google.com
# Server:		192.168.65.7
# Address:	192.168.65.7:53
#
# ** server can't find google.com:Timed out
```

**可能原因**:
1. Docker 内置 DNS 转发器故障
2. WSL2 DNS 配置错误
3. Clash DNS 设置问题

**诊断步骤**:

方法一：测试不同 DNS 服务器
```bash
# 测试容器内 DNS
docker run --rm alpine nslookup google.com

# 直接测试 Docker DNS
docker run --rm alpine nslookup google.com 192.168.65.7

# 测试 WSL2 DNS
docker run --rm alpine nslookup google.com 10.255.255.254

# 测试外部 DNS
docker run --rm alpine nslookup google.com 8.8.8.8
```

方法二：检查 DNS 配置
```bash
# 在 WSL2 中
cat /etc/resolv.conf
# 应该显示: nameserver 10.255.255.254

# 检查 Docker DNS 配置
docker run --rm alpine cat /etc/resolv.conf
# 应该显示: nameserver 192.168.65.7
```

方法三：检查 Clash DNS
打开 Clash Verge Dashboard，查看 DNS 配置：
- DNS 服务器是否可用
- 是否有大量查询失败
- Fake-IP 配置是否正常

**解决方案**:

方案 1: 更换 Docker DNS 配置
```bash
# 编辑 Docker daemon 配置
sudo nano /etc/docker/daemon.json
```

添加：
```json
{
  "dns": ["8.8.8.8", "8.8.4.4"],
  "dns-opts": ["single-request"]
}
```

重启 Docker：
```bash
sudo systemctl restart docker
```

方案 2: 修改 WSL2 DNS 生成
```bash
# 阻止自动更新 DNS
sudo nano /etc/wsl.conf
```

添加：
```ini
[network]
generateResolvConf = false
```

手动配置 DNS：
```bash
sudo nano /etc/resolv.conf
```

添加：
```
nameserver 8.8.8.8
nameserver 8.8.4.4
```

重启 WSL：
```powershell
wsl --shutdown
```

方案 3: 为容器设置自定义 DNS
```bash
docker run --rm --dns=8.8.8.8 alpine nslookup google.com
```

### 问题 4: 从 Windows 无法访问容器端口

**详细症状**:
- 容器内服务正常运行
- `docker ps` 显示端口已映射
- 从 WSL2 可以访问 `curl http://localhost:8080`
- 从 Windows 浏览器访问 `http://localhost:8080` 失败

**可能原因**:
1. Windows Defender / 防火墙阻止
2. Docker Desktop 的端口转发失败
3. WSL2 与 Windows 之间的通信问题

**诊断步骤**:

第一步：在 Windows 中检查
```powershell
# 1. 检查容器是否在监听端口
netstat -ano | findstr "8080"

# 2. 检查 Docker Desktop 进程
Get-Process | Where-Object {$_.ProcessName -like "*docker*"}

# 3. 尝试访问 WSL2 IP
# 获取 WSL2 IP
$wslIp = (wsl hostname -I).Trim()
# 在浏览器访问 http://$wslIp:8080
```

第二步：在 WSL2 中检查
```bash
# 1. 检查端口监听
netstat -tuln | grep 8080
# 或
ss -tuln | grep 8080

# 2. 检查服务是否绑定到 0.0.0.0
docker port <container-name>
# 应该返回 0.0.0.0:8080->80/tcp

# 3. 检查是否可以访问容器 IP
docker inspect <container-name> | grep IPAddress
wget -O- http://<container-ip>:80
```

第三步：检查日志
```bash
# Docker Desktop 日志
docker logs <container-name>

# 查看 Docker 服务日志
sudo journalctl -u docker.service -n 50
```

**解决方案**:

方案 1: 创建 Windows 防火墙规则
```powershell
# 以管理员身份运行 PowerShell

# 获取 WSL2 子网
$wslIp = (wsl hostname -I).Trim()
$wslSubnet = ($wslIp -split '\.')[0..2] -join '.'

# 创建入站规则（允许 WSL2 子网）
New-NetFirewallRule -DisplayName "Allow WSL2 Subnet" `
  -Direction Inbound -Action Allow -Protocol TCP `
  -RemoteAddress "$wslSubnet.0/20" -Enabled True

# 创建 Docker 桥接网络规则
New-NetFirewallRule -DisplayName "Allow Docker Bridge" `
  -Direction Inbound -Action Allow -Protocol TCP `
  -RemoteAddress "172.17.0.0/16" -Enabled True
```

方案 2: 重置 Docker Desktop 网络
```powershell
# 停止 Docker Desktop
Stop-Process -Name "Docker Desktop"

# 停止 WSL2
wsl --shutdown

# 清理网络接口
Get-NetAdapter | Where-Object {$_.InterfaceAlias -like "*Docker*"} | Disable-NetAdapter -Confirm:$false
Get-NetAdapter | Where-Object {$_.InterfaceAlias -like "*Docker*"} | Enable-NetAdapter

# 重新启动 Docker Desktop
Start-Process "${env:ProgramFiles}\Docker\Docker\Docker Desktop.exe"
```

方案 3: 使用端口代理（临时方案）
```powershell
# 获取 WSL2 IP
$wslIp = (wsl hostname -I).Trim()

# 创建端口代理（将 Windows 的 8080 转发到 WSL2 的 8080）
netsh interface portproxy add v4tov4 `
  listenport=8080 listenaddress=0.0.0.0 `
  connectport=8080 connectaddress=$wslIp

# 创建防火墙规则
netsh advfirewall firewall add rule name="Allow WSL2 Port 8080" `
  dir=in action=allow protocol=TCP localport=8080
```

### 问题 5: WSL2 启动时网络配置未应用

**详细症状**:
- 手动执行命令后网络正常
- 但 WSL2 重启后问题重现
- 自定义配置没有持久化

**根本原因**: WSL2 的网络配置不像传统 Linux 那样持久，需要在启动时重新应用。

**解决方案**:

方案 1: 使用 systemd 服务（推荐用于 Mirrored 模式）
```bash
# 创建服务目录
sudo mkdir -p /etc/systemd/system/multi-user.target.wants

# 创建启动脚本
sudo nano /usr/local/bin/wsl2-network-fix.sh
```

添加内容：
```bash
#!/bin/bash
# Fix WSL2 network settings on startup

# 等待网络就绪
sleep 5

# 修复 MTU（如果使用的是 Mirrored 模式）
ip link set eth0 mtu 1500

# 刷新 DNS
echo "nameserver 10.255.255.254" > /etc/resolv.conf

# 重启 Docker（确保使用正确的配置）
systemctl restart docker
```

设置可执行：
```bash
sudo chmod +x /usr/local/bin/wsl2-network-fix.sh
```

创建 systemd 服务：
```bash
sudo nano /etc/systemd/system/wsl2-network-fix.service
```

添加内容：
```ini
[Unit]
Description=Fix WSL2 Network on Startup
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/wsl2-network-fix.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

启用服务：
```bash
sudo systemctl enable wsl2-network-fix.service
```

方案 2: 使用 .bashrc（简单但不够优雅）
```bash
# 编辑 ~/.bashrc
echo '
# Fix WSL2 network on shell startup
if [ -n "$WSL_INTEROP" ]; then
  # 仅在 WSL2 中执行
  sudo ip link set eth0 mtu 1500 2>/dev/null
fi
' >> ~/.bashrc
```

方案 3: Windows 启动时运行（最灵活）
```powershell
# 创建 PowerShell 脚本 Fix-WSL2Network.ps1
$wslScript = @"
#!/bin/bash
# Fix WSL2 network
ip link set eth0 mtu 1500
systemctl restart docker
"@

# 保存脚本
$wslScript | Out-File -FilePath "$env:APPDATA\wsl2-network-fix.sh" -Encoding utf8

# 创建启动任务
$action = New-ScheduledTaskAction -Execute 'wsl' -Argument '-d Ubuntu -e bash /mnt/c/Users/<你的用户名>/AppData/Roaming/wsl2-network-fix.sh'
$trigger = New-ScheduledTaskTrigger -AtLogOn
Register-ScheduledTask -TaskName "Fix WSL2 Network" -Action $action -Trigger $trigger -RunLevel Highest
```

## 网络调试

### 使用 tcpdump 抓包

```bash
# 在 WSL2 中安装 tcpdump
sudo apt install tcpdump -y

# 监听 WSL2 接口
sudo tcpdump -i eth0 -n host 10.255.255.254

# 监听 Docker 接口
sudo tcpdump -i docker0 -n

# 保存到文件
sudo tcpdump -i eth0 -w /tmp/capture.pcap

# 在 Windows 中用 Wireshark 打开分析
```

### 使用 netstat/ss 查看连接

```bash
# 查看所有监听端口
sudo ss -tuln

# 查看连接到 Clash 的客户端
sudo ss -tnp | grep 7890

# 查看 Docker 网络连接
sudo ss -tnp | grep docker
```

### DNS 调试

```bash
# 使用 dig 进行详细 DNS 查询
sudo apt install dnsutils -y

# 查询通过 Docker DNS
dig @192.168.65.7 google.com

# 查询通过 WSL2 DNS
dig @10.255.255.254 google.com

# 使用 nslookup
docker run --rm -it alpine sh -c "nslookup google.com && nslookup -query=debug google.com"
```

### 路由调试

```bash
# 查看路由表
ip route show

# 追踪路由路径
docker run --rm -it alpine traceroute 8.8.8.8

# 查看 ARP 表
ip neigh show
```

## 性能问题

### Docker 构建速度慢

**诊断**:

```bash
# 使用 buildx 查看构建性能
docker buildx build --progress=plain .

# 检查磁盘 I/O
sudo iostat -x 1

# 检查网络带宽
docker run --rm alpine sh -c "wget -O /dev/null http://ipv4.download.thinkbroadband.com/100MB.zip"
```

**优化**:

1. 使用 BuildKit
```bash
export DOCKER_BUILDKIT=1
docker build --progress=plain .
```

2. 使用缓存
```bash
docker build --cache-from=type=local,src=/tmp/.buildx-cache \
             --cache-to=type=local,dest=/tmp/.buildx-cache \
             .
```

3. 使用中国镜像源（国内用户）
```json
// Docker Desktop → Settings → Docker Engine
{
  "registry-mirrors": [
    "https://mirror.gcr.io",
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com"
  ]
}
```

### 容器启动慢

**可能原因**:
1. 镜像过大
2. 网络初始化慢
3. 存储性能差

**解决方案**:

1. 使用多阶段构建减小镜像
2. 使用 Docker Compose 的 `depends_on` 和健康检查
3. 将 WSL2 移到 SSD

### 网络延迟高

**诊断**:

```bash
# 测试延迟
docker run --rm alpine sh -c "for i in {1..10}; do time wget -O /dev/null http://httpbin.org/delay/1; done"

# 检查 MTU
ip link show eth0
sudo tcpdump -i eth0 -nn -v | grep -v " [|]"  # 查找分片包
```

## 疑难解答

### 当所有方法都无效时

最后的手段：完全重置

1. **备份重要数据**:
   ```bash
   # 备份容器
docker commit container-name backup-image
docker save backup-image -o backup-image.tar

   # 备份 WSL2 数据
   wsl --export Ubuntu backup-wsl2.tar
   ```

2. **重置 Docker Desktop**:
   - 打开 Docker Desktop → Troubleshoot → Reset to factory defaults

3. **重置 WSL2**:
   ```powershell
   # 在 PowerShell (管理员)
   wsl --unregister Ubuntu
   wsl --install -d Ubuntu
   ```

4. **重新配置**:
   - 按照 [wsl2-docker-clash-setup.md](./wsl2-docker-clash-setup.md) 逐步配置

### 提交 Issue 前准备

如果需要向社区求助，准备以下信息：

```bash
# 创建诊断信息文件
cat > /tmp/diagnostic-info.txt <<EOF
=== 环境信息 ===
WSL 版本: $(wsl --version 2>/dev/null || echo "Unknown")
Docker 版本: $(docker version --format '{{.Server.Version}}')
Clash 版本: $(grep -o '"version":"[^"]*"' "/mnt/c/Program Files/Clash Verge/resources/clash-verge.exe.config" 2>/dev/null || echo "Unknown")

=== WSL2 网络信息 ===
IP: $(ip addr show eth0 | grep inet | awk '{print $2}')
Gateway: $(ip route | grep default | awk '{print $3}')
DNS: $(cat /etc/resolv.conf | grep nameserver)
MTU: $(ip link show eth0 | grep -oP 'mtu \K\d+')

=== Docker 网络信息 ===
Docker 版本: $(docker version --format '{{.Server.Version}}')
网络模式: $(docker network ls | grep bridge)
Docker DNS: $(docker run --rm alpine cat /etc/resolv.conf | grep nameserver)

=== 代理设置 ===
docker run --rm alpine env | grep -i proxy

=== 测试结果 ===
docker run --rm alpine ping -c 1 1.1.1.1 && echo "✓ WSL2 网络正常" || echo "✗ WSL2 网络异常"
docker run --rm alpine wget -q -O- http://httpbin.org/ip && echo "✓ 代理正常" || echo "✗ 代理异常"

EOF
cat /tmp/diagnostic-info.txt
```

## 获取帮助

### 官方资源

- [Docker Desktop 文档](https://docs.docker.com/desktop/)
- [WSL 文档](https://learn.microsoft.com/en-us/windows/wsl/)
- [Clash Verge GitHub](https://github.com/clash-verge-rev/clash-verge-rev)

### 社区支持

- [Docker Community Forums](https://forums.docker.com/)
- [Stack Overflow - Docker](https://stackoverflow.com/questions/tagged/docker)
- [WSL GitHub Issues](https://github.com/microsoft/WSL/issues)
- [V2EX 网络代理板块](https://v2ex.com/tag/mihoyo)

### 报告问题时包含的信息

1. **环境信息**
   - Windows 版本: `winver` 命令输出
   - WSL2 版本: `wsl --version`
   - Docker Desktop 版本
   - Clash Verge 版本

2. **网络配置**
   - WSL2 网络模式: NAT / Mirrored
   - IP 地址和路由表: `ip addr` 和 `ip route`
   - DNS 配置: `cat /etc/resolv.conf`

3. **错误信息**
   - 完整的错误消息
   - 相关日志: `docker logs`, `journalctl`

4. **已尝试的解决方案**
   - 哪些方法有效？哪些无效？
   - 完整的重现步骤
