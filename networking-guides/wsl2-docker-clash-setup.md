# WSL2 + Docker Desktop + Clash Verge 完整配置指南

## 📋 目录

- [环境准备](#环境准备)
- [网络架构理解](#网络架构理解)
- [配置步骤](#配置步骤)
- [代理设置](#代理设置)
- [验证配置](#验证配置)
- [常见问题](#常见问题)

## 环境准备

### 系统要求
- Windows 10 版本 2004 及更高版本，或 Windows 11
- WSL2 版本 2.1.5 或更高
- Docker Desktop 4.53.0 或更高
- Clash Verge Rev v1.x.x 或更高

### 当前环境配置
```
操作系统: Windows 11 (Build 26200.7171)
WSL2 版本: 2.6.1.0
WSL2 内核: 6.6.87.2-microsoft-standard-WSL2
Docker Desktop: 4.53.0 (211793)
Docker Engine: 29.0.1
Clash Verge: TUN 模式已启用
```

### 网络信息
```bash
# WSL2 网络
IP 地址: 172.24.245.37/20
默认网关: 172.24.240.1
DNS 服务器: 10.255.255.254 (Clash TUN)

# Docker 网络
Docker 网桥: 172.17.0.0/16
Docker DNS: 192.168.65.7

# Clash TUN
虚拟网关: 10.255.255.254
代理端口: 7890 (HTTP)/7891 (SOCKS)
```

## 网络架构理解

### 1. 三层网络结构

```
┌─────────────────────────────────────────┐
│          Windows 主机                    │
│  • Clash Verge TUN (10.255.255.254)    │
│  • Docker Desktop 后端                    │
│  • WSL2 虚拟交换机                        │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│           WSL2 虚拟机                    │
│  • eth0 (172.24.245.37/20)              │
│  • Docker 客户端                         │
│  • 你的开发环境                          │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         Docker 容器网络                  │
│  • 桥接网络 (172.17.0.0/16)            │
│  • 独立 DNS (192.168.65.7)             │
│  • 端口映射到宿主机                      │
└─────────────────────────────────────────┘
```

### 2. DNS 解析流程

```
容器内部
   ↓ (查询 docker0 的 DNS 服务)
192.168.65.7 (Docker DNS)
   ↓ (如果不在 Docker 网络，向上查询)
10.255.255.254 (Clash TUN)
   ↓ (根据规则分流)
  8.8.8.8 或 223.5.5.5
```

### 3. 流量走向

**无代理情况**: 容器 → Docker 网桥 → WSL2 eth0 → Windows → 互联网

**使用代理**: 容器 → Docker 网桥 → WSL2 eth0 → Clash TUN (10.255.255.254) → 代理服务器

## 配置步骤

### 步骤 1: 验证 WSL2 安装

```bash
# 检查 WSL 版本，确保是 WSL2
wsl --status

# 列出已安装的发行版
wsl -l -v

# 确保你的发行版是版本 2
# 如果不是，转换到 WSL2
wsl --set-version <发行版名称> 2
```

### 步骤 2: 安装 Docker Desktop

1. **下载 Docker Desktop**
   - 从官网下载最新版: https://www.docker.com/products/docker-desktop/

2. **安装注意事项**
   ```
   ✓ Use WSL 2 instead of Hyper-V (recommended)
   ✗ Allow Windows Containers... (除非你明确需要)
   ✓ Add shortcut to desktop (可选)
   ```

3. **安装后配置**
   - 启动 Docker Desktop
   - 进入 Settings → General
   - 确认 "Use WSL 2 based engine" 已启用
   - 进入 Settings → Resources → WSL Integration
   - 启用你的 WSL2 发行版
   - 点击 Apply & Restart

### 步骤 3: 配置 Clash Verge

1. **以管理员身份运行 Clash Verge**
   - 右键点击 Clash Verge → "以管理员身份运行"
   - **这是必须的**，否则无法创建 TUN 虚拟网卡

2. **启用 TUN 模式**
   - 进入 Settings → TUN Settings
   - 开启 "Enable TUN Mode"
   - 如果提示安装驱动，点击安装

3. **配置代理端口**
   - 确保代理端口设置：
     - HTTP: 7890
     - SOCKS: 7891
     - Mixed: 7892 (可选)

4. **关闭系统代理**
   - 在主页面上关闭 "System Proxy"
   - **重要**：避免与 TUN 模式冲突

5. **验证 TUN 模式**
   ```bash
   # 在 Windows PowerShell 中
   ipconfig
   # 应该看到一个 "TAP-Windows Adapter" 或类似接口
   # IP 应该是 198.18.x.x
   ```

### 步骤 4: WSL2 网络配置

#### 方案 A: NAT 模式（当前配置，推荐）

无需额外配置，WSL2 默认使用 NAT 模式。

```bash
# 验证当前模式
# 检查是否有 /etc/wsl.conf 文件
cat /etc/wsl.conf 2>/dev/null || echo "No wsl.conf found"
```

#### 方案 B: Mirrored 模式（如需启用）

如果需要从局域网访问 WSL2 服务，可启用 Mirrored 模式：

```ini
# 在 Windows 用户目录创建 .wslconfig
# 路径: C:\Users\<你的用户名>\.wslconfig

[wsl2]
networkingMode = mirrored

dnsTunneling = true
firewall = true
autoProxy = false

[experimental]
autoMemoryReclaim = gradual
```

**⚠️ 重要提醒**

启用 Mirrored 模式后，需要修复 MTU：

```bash
# 在 WSL2 中执行
sudo ip link set eth0 mtu 1500

# 验证修改
ip link show eth0 | grep mtu
```

## 代理设置

### 方法 1: Docker Desktop GUI 配置（推荐新手）

1. 打开 Docker Desktop
2. 进入 Settings → Resources → Proxies
3. 选择 "Manual proxy configuration"
4. 填写以下信息：

```
Web Server (HTTP): http://host.docker.internal:7890
Secure Web Server (HTTPS): http://host.docker.internal:7890

No Proxy:
localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12
```

5. 点击 Apply & Restart

### 方法 2: Docker 服务配置（高级）

如果 Docker Desktop 设置无效，可直接配置 Docker 服务：

```bash
# 创建配置目录
sudo mkdir -p /etc/systemd/system/docker.service.d

# 创建代理配置文件
sudo nano /etc/systemd/system/docker.service.d/proxy.conf
```

添加以下内容：

```ini
[Service]
Environment="HTTP_PROXY=http://host.docker.internal:7890"
Environment="HTTPS_PROXY=http://host.docker.internal:7890"
Environment="NO_PROXY=localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12"
```

重新加载并重启 Docker：

```bash
# 重载 systemd 配置
sudo systemctl daemon-reload

# 重启 Docker 服务
sudo systemctl restart docker

# 验证配置
sudo systemctl show --property=Environment docker
```

### 方法 3: 容器级别代理（临时）

对于特定容器，可以使用环境变量设置代理：

```bash
docker run -e HTTP_PROXY=http://host.docker.internal:7890 \
           -e HTTPS_PROXY=http://host.docker.internal:7890 \
           -e NO_PROXY=localhost,127.0.0.1,.docker.internal \
           your-image
```

或者使用 Docker Compose：

```yaml
version: '3.8'
services:
  app:
    image: your-image
    environment:
      - HTTP_PROXY=http://host.docker.internal:7890
      - HTTPS_PROXY=http://host.docker.internal:7890
      - NO_PROXY=localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12
```

## 验证配置

### 1. Docker 基础功能测试

```bash
# 检查 Docker 版本
docker version

# 应该显示 Client 和 Server 信息
# Server 应该显示: Docker Desktop 4.53.0
```

### 2. 网络连通性测试

```bash
# 测试 DNS 解析
docker run --rm alpine nslookup google.com

# 应该返回类似：
# Server:		192.168.65.7
# Name:		google.com

# 测试 HTTP 访问
docker run --rm alpine wget -O- http://httpbin.org/ip

# 应该成功获取你的公网 IP

# 测试 HTTPS 访问
docker run --rm alpine wget -O- https://httpbin.org/ip

# 应该成功（通过代理）
```

### 3. 代理功能测试

```bash
# 检查 Docker 环境变量
docker run --rm alpine env | grep -i proxy

# 应该显示：
# HTTPS_PROXY=http://host.docker.internal:7890
# HTTP_PROXY=http://host.docker.internal:7890

# 测试通过代理访问
docker run --rm alpine wget -O- https://www.google.com

# 应该成功返回内容
```

### 4. 端口转发测试

```bash
# 启动测试容器
docker run -d -p 8080:80 --name test-nginx nginx:alpine

# 从 WSL2 访问
curl http://localhost:8080
# 应该返回 Nginx 欢迎页

# 从 Windows 访问（在 PowerShell）
# curl http://localhost:8080
# 或者直接在浏览器访问 http://localhost:8080

# 清理
docker stop test-nginx
docker rm test-nginx
```

### 5. 性能测试

```bash
# 测试 Docker 拉取速度
time docker pull alpine:latest

# 测试容器内部网络速度
docker run --rm alpine sh -c "time wget -O /dev/null http://cachefly.cachefly.net/100mb.test"
```

### 6. Clash TUN 验证

在 Windows PowerShell 中：

```powershell
# 检查 TUN 接口
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TAP*"}

# 检查路由表
Get-NetRoute -DestinationPrefix 0.0.0.0/0

# 查看 Clash 端口占用
netstat -ano | findstr "7890"
```

## 常见问题

### 问题 1: Docker pull 失败，提示连接被拒绝

**症状**:
```
Error response from daemon: Get https://registry-1.docker.io/v2/:
proxyconnect tcp: dial tcp 127.0.0.1:7890: connect: connection refused
```

**原因**: Docker 使用了 127.0.0.1 作为代理地址，但在 WSL2 中 127.0.0.1 指向的是 WSL2 本身，而不是 Windows 宿主机。

**解决方案**:
使用 `host.docker.internal` 替代 `127.0.0.1` 或 `localhost`。

```bash
# 错误的配置
HTTP_PROXY=http://127.0.0.1:7890  # ❌

# 正确的配置
HTTP_PROXY=http://host.docker.internal:7890  # ✅
```

### 问题 2: Mirrored 模式下 HTTPS 网站无法访问

**症状**:
```
wget: error getting response: No route to host
# 或者 SSL 握手失败
```

**原因**: Clash Verge TUN 模式默认 MTU=9000，而 WSL2 Mirrored 模式期望 MTU=1500，导致大包无法传输。

**解决方案**:
```bash
# 临时修复（重启后失效）
sudo ip link set eth0 mtu 1500

# 永久修复（推荐）
# 创建 systemd 服务自动执行
cat <<'EOF' | sudo tee /etc/systemd/system/mtu-fix.service
[Unit]
Description=Fix MTU for WSL2 Mirrored Mode
After=network-online.target

[Service]
Type=oneshot
ExecStart=/sbin/ip link set eth0 mtu 1500

[Install]
WantedBy=multi-user.target
EOF

# 启用服务
sudo systemctl enable mtu-fix.service
```

### 问题 3: 容器内无法解析 .local 域名

**症状**:
```
ping: mydevice.local: Name or service not known
```

**原因**: Docker 的 DNS 配置可能不包含 mDNS 解析。

**解决方案**:
```bash
# 在 docker run 时添加主机映射
docker run --add-host=mydevice.local:<设备IP> your-image

# 或者在 docker-compose.yml 中添加
extra_hosts:
  - "mydevice.local:192.168.1.100"
```

### 问题 4: Windows 防火墙阻止 Docker 网络

**症状**: Windows 杀毒软件或防火墙阻止 Docker 网络连接。

**解决方案**:

在 PowerShell (管理员权限):

```powershell
# 获取 WSL2 子网
$wslIp = (wsl hostname -I).Trim()
$wslSubnet = ($wslIp -split '\.')[0..2] -join '.'

# 创建 WSL2 子网入站规则
New-NetFirewallRule -DisplayName "WSL2 Subnet" `
  -Direction Inbound -Action Allow -Protocol TCP `
  -RemoteAddress "$wslSubnet.0/20" -Enabled True

# 创建 Docker 桥接网络入站规则
New-NetFirewallRule -DisplayName "Docker Bridge Network" `
  -Direction Inbound -Action Allow -Protocol TCP `
  -RemoteAddress "172.17.0.0/16" -Enabled True

# 创建 Docker Backend 出站规则
New-NetFirewallRule -DisplayName "Docker Backend Outbound" `
  -Direction Outbound -Action Allow -Program "${env:ProgramFiles}\Docker\Docker\resources\com.docker.backend.exe" `
  -Enabled True
```

### 问题 5: Clash 与 Docker 端口冲突

**症状**: Clash 无法启动或 Docker 容器无法绑定端口。

**常见冲突端口**:
- 7890 (HTTP 代理)
- 7891 (SOCKS 代理)
- 9090 (Clash Dashboard)

**解决方案**:

```bash
# 检查端口占用
# Windows 中
netstat -ano | findstr "7890"

# 修改 Clash 端口
# 编辑 Clash 配置文件
port: 7899  # 修改为其他端口
socks-port: 7898
dashboard-port: 9091
```

### 问题 6: WSL2 内存不足导致 Docker 崩溃

**症状**: Docker 容器频繁 OOM (Out of Memory) 或被终止。

**解决方案**:

```ini
# 在 %USERPROFILE%\.wslconfig 中添加
[wsl2]
memory=8GB              # 根据系统内存调整
processors=4            # 根据 CPU 核心数调整
swap=8GB                # 交换文件大小
swapFile=D:\wsl2-swap.vhdx   # 可选：指定交换文件位置
```

然后重启 WSL:
```powershell
wsl --shutdown
```

## 🛡️ 安全加固

### 1. 限制容器权限

```bash
# 以非 root 用户运行容器
docker run --user $(id -u):$(id -g) your-image

# 只读根文件系统
docker run --read-only your-image

# 限制系统调用
docker run --security-opt no-new-privileges your-image
```

### 2. 网络隔离

```bash
# 创建自定义桥接网络
docker network create --driver bridge my-network

# 在自定义网络中运行容器
docker run --network my-network your-image
```

### 3. 定期更新

```bash
# 更新 Docker 镜像
docker pull image:tag

# 清理无用资源
docker system prune -a

# 更新 WSL2 发行版
sudo apt update && sudo apt upgrade -y
```

## 📊 性能优化

### 1. Docker 镜像优化

```dockerfile
# 使用多阶段构建
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

### 2. WSL2 性能调优

```ini
# ~/.wslconfig
[wsl2]
# 限制 WSL2 内存使用，避免占用过多 Windows 内存
memory=8GB

# 限制 CPU 核心数
processors=4

# 禁用页表压缩，提升性能
pageReporting=false

# 禁用 GUI 应用
localhostforwarding=true

[experimental]
# 自动回收内存
autoMemoryReclaim=gradual

# 稀疏 VHD，节省磁盘空间
sparseVhd=true
```

### 3. Docker 构建加速

```bash
# 配置镜像加速器（中国用户适用）
# 在 Docker Desktop → Settings → Docker Engine
{
  "registry-mirrors": [
    "https://mirror.gcr.io",
    "https://docker.mirrors.ustc.edu.cn"
  ]
}
```

## 📈 监控和日志

### 1. Docker 监控

```bash
# 实时查看容器资源使用
docker stats

# 查看容器日志
docker logs -f container-name

# 查看 Docker 系统事件
docker events
```

### 2. WSL2 监控

```bash
# WSL2 内存使用
free -h

# 实时系统监控
top

# 网络流量监控
tcpdump -i eth0
```

### 3. Clash Verge 日志

在 Clash Verge Dashboard 中：
- 查看实时日志
- 检查连接状态
- 验证规则匹配

## 🔄 备份和恢复

### 备份 WSL2 发行版

```bash
# 导出 WSL2 发行版
wsl --export Ubuntu D:\backup\wsl-ubuntu.tar

# 导入 WSL2 发行版
wsl --import Ubuntu-Restored D:\wsl2-imported D:\backup\wsl-ubuntu.tar
```

### 备份 Docker 数据

```bash
# 备份所有镜像
docker save $(docker images -q) -o docker-images-backup.tar

# 恢复镜像
docker load -i docker-images-backup.tar
```
