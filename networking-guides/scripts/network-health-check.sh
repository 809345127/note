#!/bin/bash

# WSL2 + Docker Desktop + Clash Verge 网络健康检查脚本
# 使用方法: ./network-health-check.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的输出
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 命令未找到，请安装相关包"
        exit 1
    fi
}

# 初始化检查
init_checks() {
    check_command docker
    check_command curl
    check_command ip
    check_command ping
}

# 测试 Docker 基本功能
test_docker_basic() {
    print_header "检查 Docker 基本功能"

    # 检查 Docker 服务
    if systemctl is-active --quiet docker; then
        print_status "Docker 服务正在运行"
    else
        print_error "Docker 服务未运行"
        return 1
    fi

    # 检查版本
    local docker_version=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "Unknown")
    print_status "Docker 版本: $docker_version"

    # 运行测试容器
    if docker run --rm hello-world &>/dev/null; then
        print_status "Docker 运行测试容器成功"
    else
        print_error "Docker 无法运行测试容器"
        return 1
    fi
}

# 测试网络连通性
test_connectivity() {
    print_header "检查网络连通性"

    # 测试 WSL2 DNS
    if curl -s --connect-timeout 5 http://www.google.com > /dev/null; then
        print_status "WSL2 HTTP 连接正常"
    else
        print_error "WSL2 HTTP 连接失败"
    fi

    # 测试容器 DNS
    if docker run --rm alpine nslookup google.com &>/dev/null; then
        print_status "容器 DNS 解析正常"
    else
        print_error "容器 DNS 解析失败"
    fi

    # 测试容器 HTTP
    if docker run --rm alpine wget -q -O- http://www.google.com > /dev/null; then
        print_status "容器 HTTP 访问正常"
    else
        print_error "容器 HTTP 访问失败"
    fi

    # 测试容器 HTTPS
    if docker run --rm alpine wget -q -O- https://www.google.com > /dev/null; then
        print_status "容器 HTTPS 访问正常"
    else
        print_error "容器 HTTPS 访问失败"
    fi
}

# 测试代理配置
test_proxy() {
    print_header "检查代理配置"

    # 检查 Docker 代理环境变量
    local proxy_config=$(docker run --rm alpine env | grep -i proxy || echo "No proxy config found")
    if [[ "$proxy_config" =~ "host.docker.internal" ]]; then
        print_status "检测到代理配置"
        echo "配置: $proxy_config"
    else
        print_warning "未检测到代理配置（可能使用系统代理）"
    fi

    # 测试通过代理访问（如果配置）
    if docker run --rm alpine wget -q -O- http://httpbin.org/ip > /dev/null 2>&1; then
        print_status "代理访问测试成功"
    else
        print_error "代理访问测试失败"
    fi
}

# 测试端口转发
test_port_forwarding() {
    print_header "检查端口转发"

    # 启动测试容器
    local container_name="health-check-nginx-$$"
    docker run -d -p 18080:80 --name "$container_name" nginx:alpine &>/dev/null

    # 等待容器启动
    sleep 3

    # 测试从 WSL2 访问
    if curl -s http://localhost:18080 > /dev/null; then
        print_status "WSL2 → 容器端口转发正常"
    else
        print_error "WSL2 → 容器端口转发失败"
    fi

    # 测试从容器访问
    if docker exec "$container_name" wget -q -O- http://localhost > /dev/null; then
        print_status "容器内部访问正常"
    else
        print_error "容器内部访问失败"
    fi

    # 清理
    docker stop "$container_name" &>/dev/null
    docker rm "$container_name" &>/dev/null
}

# 测试 host.docker.internal
test_host_access() {
    print_header "检查 host.docker.internal 访问"

    # 测试 DNS 解析
    if docker run --rm alpine nslookup host.docker.internal &>/dev/null; then
        print_status "host.docker.internal 解析正常"
    else
        print_error "host.docker.internal 解析失败"
    fi

    # 测试连通性
    if docker run --rm alpine ping -c 2 host.docker.internal &>/dev/null; then
        print_status "host.docker.internal 访问正常"
    else
        print_warning "host.docker.internal 可能无法访问（某些服务可能被防火墙阻止）"
    fi
}

# 检查 MTU 配置
check_mtu() {
    print_header "检查 MTU 配置"

    local mtu=$(ip link show eth0 | grep -oP 'mtu \K\d+')
    print_status "当前 MTU: $mtu"

    if [ "$mtu" = "1500" ]; then
        print_status "MTU 配置正常"
    else
        print_warning "MTU 不是标准值 1500（如果使用 Mirrored 模式，建议设置为 1500）"
    fi

    # 测试大包
    if docker run --rm alpine ping -c 2 -s 1400 8.8.8.8 &>/dev/null; then
        print_status "大包测试通过"
    else
        print_warning "大包测试失败（可能 MTU 设置不当）"
    fi
}

# 检查 DNS 配置
check_dns() {
    print_header "检查 DNS 配置"

    # WSL2 DNS
    local wsl2_dns=$(cat /etc/resolv.conf | grep nameserver | awk '{print $2}')
    print_status "WSL2 DNS: $wsl2_dns"

    # Docker DNS
    local docker_dns=$(docker run --rm alpine cat /etc/resolv.conf 2>/dev/null | grep nameserver | awk '{print $2}' || echo "Unknown")
    print_status "Docker DNS: $docker_dns"

    # 检查 DNS 一致性
    if [ "$wsl2_dns" = "10.255.255.254" ]; then
        print_status "WSL2 使用 Clash TUN DNS（正常）"
    else
        print_warning "WSL2 DNS 不是 Clash TUN 地址"
    fi
}

# 检查系统资源
check_resources() {
    print_header "检查系统资源"

    # 内存使用
    local mem_info=$(free -h | grep Mem)
    print_status "内存使用: $mem_info"

    # 磁盘使用
    local disk_info=$(df -h / | tail -1)
    print_status "根目录磁盘: $disk_info"

    # Docker 磁盘使用
    local docker_disk=$(du -sh /var/lib/docker 2>/dev/null || echo "Unknown")
    print_status "Docker 数据目录: $docker_disk"
}

# 生成报告
generate_report() {
    print_header "健康检查报告"

    cat <<EOF
检查项目：
1. $(docker version --format '{{.Server.Version}}' >/dev/null 2>&1 && echo "✅ Docker 服务正常" || echo "❌ Docker 服务异常")
2. $(docker run --rm hello-world >/dev/null 2>&1 && echo "✅ Docker 运行正常" || echo "❌ Docker 运行异常")
3. $(curl -s --connect-timeout 5 http://www.google.com >/dev/null && echo "✅ WSL2 网络正常" || echo "❌ WSL2 网络异常")
4. $(docker run --rm alpine ping -c 2 1.1.1.1 >/dev/null 2>&1 && echo "✅ 容器网络正常" || echo "❌ 容器网络异常")
5. $(docker run --rm alpine wget -q -O- https://www.google.com >/dev/null 2>&1 && echo "✅ HTTPS 访问正常" || echo "❌ HTTPS 访问异常")

环境信息：
- WSL2 IP: $(ip addr show eth0 2>/dev/null | grep "inet " | awk '{print $2}')
- Docker 版本: $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "Unknown")
- 当前时间: $(date)

建议：
- 如果所有检查都通过，说明网络配置正常
- 如果有检查失败，请参考 troubleshooting.md
- 对于 DNS 问题，检查 /etc/resolv.conf
- 对于代理问题，检查 Docker 代理配置
EOF
}

# 主函数
main() {
    print_header "WSL2 + Docker + Clash Verge 网络健康检查"

    # 初始化检查
    init_checks

    # 执行各项检查
    test_docker_basic
    test_connectivity
    test_proxy
    test_port_forwarding
    test_host_access
    check_mtu
    check_dns
    check_resources

    # 生成报告
    generate_report

    print_status "检查完成！"
    echo ""
    echo "如需更详细的故障排除信息，请查看:"
    echo "- troubleshooting.md"
    echo "- quick-reference.md"
}

# 如果直接执行脚本（不是被 source）
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
