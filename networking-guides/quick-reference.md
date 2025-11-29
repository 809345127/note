# 快速参考手册

## 📋 目录

- [基础命令](#基础命令)
- [网络测试](#网络测试)
- [代理设置](#代理设置)
- [故障排查](#故障排查)
- [性能优化](#性能优化)

## 基础命令

### WSL2 管理

```bash
# 查看 WSL 状态
wsl --status

# 列出所有发行版
wsl -l -v

# 关闭 WSL2
wsl --shutdown

# 导出 WSL2 发行版
wsl --export Ubuntu backup.tar

# 导入 WSL2 发行版
wsl --import Ubuntu-New ./new-location backup.tar
```

### Docker 管理

```bash
# Docker 版本信息
docker version
docker info

# 容器管理
docker ps              # 查看运行中容器
docker ps -a          # 查看所有容器
docker images         # 查看镜像
docker logs <container>  # 查看日志
docker exec -it <container> bash  # 进入容器

# 清理资源
docker container prune  # 删除停止的容器
docker image prune      # 删除无用镜像
docker system prune     # 清理所有
```

### Clash Verge 管理

```bash
# 在 Windows PowerShell 中

# 检查 Clash 进程
Get-Process | Where-Object {$_.ProcessName -like "*clash*"}

# 检查端口占用
netstat -ano | findstr "7890"
tasklist /FI "PID eq <端口号>"

# 重启 Clash 服务
Restart-Service -Name "Clash Verge Service"
```

## 网络测试

### 基础连通性

```bash
# 测试外网连通
ping -c 4 1.1.1.1
curl -I http://www.google.com

# 测试 DNS
docker run --rm alpine nslookup google.com
docker run --rm alpine nslookup google.com 10.255.255.254

# 检查路由
ip route
netstat -rn
traceroute 8.8.8.8
```

### 代理测试

```bash
# 检查容器内代理设置
docker run --rm alpine env | grep -i proxy

# 测试通过代理访问
docker run --rm alpine wget -O- http://httpbin.org/ip
docker run --rm alpine wget -O- https://httpbin.org/ip

# 查看当前公网 IP
curl -s http://whatismyip.akamai.com/
wget -q -O- http://whatismyip.akamai.com/
```

### Docker 网络测试

```bash
# 启动测试容器
docker run -d -p 8080:80 --name test-nginx nginx:alpine

# 从 WSL2 测试
curl http://localhost:8080

# 查看容器 IP
docker inspect test-nginx | grep IPAddress

# 从容器内测试
docker exec test-nginx wget -O- http://localhost

# 清理
docker stop test-nginx && docker rm test-nginx
```

### MTU 测试

```bash
# 检查当前 MTU
ip link show eth0 | grep mtu

# 测试大包（测试 MTU 1500）
ping -c 3 -s 1472 8.8.8.8

# 如果失败，测试小包
docker run --rm alpine ping -c 3 -s 1400 8.8.8.8

# 修改 MTU（Mirrored 模式需要）
sudo ip link set eth0 mtu 1500
```

## 代理设置

### Docker Desktop GUI 设置

1. 打开 Settings → Resources → Proxies
2. 配置：
```
HTTP Proxy: http://host.docker.internal:7890
HTTPS Proxy: http://host.docker.internal:7890
No Proxy: localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12
```

### Docker 服务配置

```bash
# 创建配置目录
sudo mkdir -p /etc/systemd/system/docker.service.d

# 创建配置文件
sudo nano /etc/systemd/system/docker.service.d/proxy.conf

# 添加内容
[Service]
Environment="HTTP_PROXY=http://host.docker.internal:7890"
Environment="HTTPS_PROXY=http://host.docker.internal:7890"
Environment="NO_PROXY=localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12"

# 重启 Docker
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 临时容器代理

```bash
# 一次性设置
docker run -e HTTP_PROXY=http://host.docker.internal:7890 \
           -e HTTPS_PROXY=http://host.docker.internal:7890 \
           -e NO_PROXY=localhost,127.0.0.1,.docker.internal \
           your-image
```

### Docker Compose 代理

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

## 故障排查

### 问题 1: Docker pull 失败

**症状**: `proxyconnect tcp: dial tcp 127.0.0.1:7890: connect: connection refused`

**快速修复**:
```bash
# 停止 Docker Desktop
wsl --shutdown
# 重新启动 Docker Desktop

# 如果仍失败，检查代理设置
docker run --rm alpine wget -O- http://host.docker.internal:7890
```

### 问题 2: Mirrored 模式 HTTPS 失败

**症状**: `wget: error getting response: No route to host`

**快速修复**:
```bash
# 在 WSL2 中执行
sudo ip link set eth0 mtu 1500

# 永久修复
sudo systemctl enable mtu-fix.service  # 如果已创建服务
```

### 问题 3: 容器无法上网

**症状**: Docker 容器 ping 不通外网

**检查步骤**:
```bash
# 1. 检查容器 DNS
docker run --rm alpine cat /etc/resolv.conf

# 2. 测试 DNS 解析
docker run --rm alpine nslookup google.com

# 3. 测试网络连通
docker run --rm alpine ping -c 2 8.8.8.8

# 4. 检查 Docker 网络
docker network ls
docker network inspect bridge

# 5. 重启 Docker
docker system prune -f
wsl --shutdown
# 重新启动 Docker Desktop
```

### 问题 4: Windows 无法访问容器端口

**症状**: 访问 localhost:8080 失败

**检查步骤**:
```bash
# 1. 检查容器是否运行
docker ps | grep your-container

# 2. 检查端口映射
docker port your-container

# 3. 从 WSL2 测试
curl http://localhost:8080

# 4. 检查 Windows 防火墙（PowerShell）
netsh advfirewall firewall show rule name="Docker"

# 5. 临时禁用防火墙测试（不推荐长期使用）
netsh advfirewall set allprofiles state off
```

### 问题 5: Clash 与 Docker 冲突

**症状**: Clash 无法启动或端口被占用

**检查步骤**:
```bash
# 1. 检查端口占用（Windows PowerShell）
netstat -ano | findstr "7890"
netstat -ano | findstr "7891"

# 2. 查找进程
tasklist /FI "PID eq <进程号>"

# 3. 重启 Clash
# 结束任务后重新启动 Clash Verge

# 4. 修改 Clash 端口（如果必要）
# 编辑配置文件，修改端口
```

## 性能优化

### 1. WSL2 资源配置

```bash
# 编辑 ~/.wslconfig（Windows 用户目录）
[wsl2]
memory=8GB              # 根据系统调整
processors=4            # 根据 CPU 调整
swap=8GB

# 重启 WSL
wsl --shutdown
```

### 2. Docker 构建优化

```bash
# 使用 BuildKit
export DOCKER_BUILDKIT=1

# 多阶段构建
docker build --target production -t myapp:prod .
```

### 3. 镜像加速（中国用户）

```bash
# Docker Desktop → Settings → Docker Engine
{
  "registry-mirrors": [
    "https://mirror.gcr.io",
    "https://docker.mirrors.ustc.edu.cn"
  ]
}
```

### 4. 清理无用资源

```bash
# 清理停止的容器
docker container prune -f

# 清理无标签镜像
docker image prune -f

# 清理所有未使用资源
docker system prune -af

# 清理构建缓存
docker builder prune -af
```

## 🔍 调试命令

### 查看网络详情

```bash
# 查看 Docker 网络
docker network inspect bridge
docker network ls

# 查看容器网络配置
docker exec <container> ip addr
docker exec <container> ip route

# 查看宿主机网络
ip addr show
ip route show
```

### 查看系统资源

```bash
# 查看内存使用
free -h

# 查看磁盘使用
df -h
du -sh /var/lib/docker

# 实时系统监控
top
htop  # 如果已安装
```

### 查看 Docker 日志

```bash
# 查看 Docker 守护进程日志
sudo journalctl -u docker.service -f

# 查看容器实时日志
docker logs -f <container>

# 查看特定时间的日志
docker logs --since "2024-01-01" <container>
```

## 📝 环境信息模板

遇到问题时，请提供以下信息：

```bash
echo "=== 环境信息 ==="
echo "WSL2 IP: $(ip addr show eth0 | grep 'inet ' | awk '{print $2}')"
echo "WSL2 DNS: $(cat /etc/resolv.conf | grep nameserver)"
echo "Docker Version: $(docker version --format {{.Server.Version}})"
echo "网络模式: $(grep -i 'networkingmode' /etc/wsl.conf 2>/dev/null || echo 'NAT(默认)')"
echo "MTU: $(ip link show eth0 | grep -oP 'mtu \K\d+')"
echo "==============="
```

## 🚀 一键诊断脚本

```bash
#!/bin/bash
echo "===== Docker 网络诊断 ====="

echo "1. 检查 Docker 版本:"
docker version --format '{{.Server.Version}}'

echo "2. 检查 WSL2 网络:"
ip addr show eth0 | grep "inet "

echo "3. 检查 DNS:"
cat /etc/resolv.conf | grep nameserver

echo "4. 测试 Docker 网络:"
docker run --rm alpine ping -c 2 1.1.1.1 > /dev/null && echo "✅ 外网正常" || echo "❌ 外网失败"

echo "5. 测试 Docker 代理:"
docker run --rm alpine wget -q -O- http://httpbin.org/ip > /dev/null && echo "✅ 代理正常" || echo "❌ 代理失败"

echo "===== 诊断完成 ====="
```

保存为 `quick-diagnose.sh` 并执行:
```bash
chmod +x quick-diagnose.sh
./quick-diagnose.sh
```
