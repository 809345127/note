# WSL2 + Docker Desktop + Clash Verge 网络配置指南

## 📋 文档说明

本文档记录了在 WSL2 环境中使用 Docker Desktop 和 Clash Verge TUN 模式的最佳实践和已知问题的解决方案。

**环境信息**
- 创建时间: 2025-11-29
- WSL2 版本: 2.6.1.0 (Kernel 6.6.87.2)
- Docker Desktop: 4.53.0 (211793)
- 网络模式: NAT (默认)

## 📁 文档结构

- [wsl2-docker-clash-setup.md](./wsl2-docker-clash-setup.md) - 完整配置指南
- [quick-reference.md](./quick-reference.md) - 快速参考命令
- [troubleshooting.md](./troubleshooting.md) - 故障排除手册
- [scripts/](./scripts/) - 实用脚本集合

## 🎯 快速开始

如果这是第一次配置，请按以下步骤操作：

1. **阅读完整配置指南**：[wsl2-docker-clash-setup.md](./wsl2-docker-clash-setup.md)
2. **执行健康检查**：运行 `scripts/network-health-check.sh`
3. **遇到问题？** 查看 [troubleshooting.md](./troubleshooting.md)

## 💡 关键配置要点

### 网络配置
- **WSL2 IP**: 172.24.245.37/20
- **DNS**: 10.255.255.254 (Clash TUN 虚拟网关)
- **Docker DNS**: 192.168.65.7 (Docker 内置)

### 代理设置
Docker Desktop → Proxies:
- HTTP_PROXY: `http://host.docker.internal:7890`
- HTTPS_PROXY: `http://host.docker.internal:7890`
- NO_PROXY: `localhost,127.0.0.1,.docker.internal,192.168.0.0/16,10.0.0.0/8,172.16.0.0/12`

## 🚨 已知问题

| 问题 | 状态 | 解决方案 |
|------|------|---------|
| Docker pull 超时 | ✅ 已解决 | 使用 host.docker.internal 而非 127.0.0.1 |
| Mirrored 模式 HTTPS 失败 | ⚠️ 潜在风险 | 设置 MTU=1500 |
| 端口转发问题 | ✅ 已解决 | Docker 自动处理 |
| DNS 解析冲突 | ✅ 已解决 | Docker 使用独立 DNS |

## 📞 参考链接

- [Docker Desktop WSL2 文档](https://docs.docker.com/desktop/features/wsl/)
- [Clash Verge 官方文档](https://github.com/clash-verge-rev/clash-verge-rev)
- [WSL2 网络架构](https://learn.microsoft.com/en-us/windows/wsl/networking)

## 🔄 更新日志

- 2025-11-29: 初始文档创建
- 环境测试通过：WSL2 + Docker Desktop + Clash Verge TUN
