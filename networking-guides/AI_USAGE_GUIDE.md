# 给后续 AI 的说明

如果你作为 AI 正在查看这个目录，说明用户遇到了 WSL2 + Docker Desktop + Clash Verge 相关的问题。

## 🎯 快速入门

1. **先阅读** [README.md](./README.md) - 了解整体情况
2. **运行健康检查**: `cd scripts && ./network-health-check.sh`
3. **查找问题**: [troubleshooting.md](./troubleshooting.md)
4. **快速参考**: [quick-reference.md](./quick-reference.md)
5. **完整配置**: [wsl2-docker-clash-setup.md](./wsl2-docker-clash-setup.md)

## 📝 环境快照

当前环境配置（2025-11-29）:
- **WSL2**: NAT 模式（非 Mirrored）
- **Docker Desktop**: 4.53.0
- **Clash Verge**: TUN 模式已启用
- **网络状态**: ✅ 测试通过，无冲突

## 🔧 常见问题快速修复

### Docker pull 失败（connection refused）
```bash
# 原因：Docker 使用了 127.0.0.1 作为代理地址
# 修复：改为 host.docker.internal
HTTP_PROXY=http://host.docker.internal:7890
```

### HTTPS 访问失败（Mirrored 模式）
```bash
# 原因：MTU 不匹配（Clash TUN=9000, WSL2=1500）
# 修复：
sudo ip link set eth0 mtu 1500
```

### DNS 解析超时
```bash
# 检查 Docker 是否使用独立 DNS
docker run --rm alpine cat /etc/resolv.conf
docker run --rm alpine nslookup google.com 192.168.65.7
```

## 🧪 验证配置

使用健康检查脚本可以快速验证当前状态：
```bash
cd /home/shize/note/networking-guides/scripts
./network-health-check.sh
```

## 💡 用户习惯

用户倾向于：
1. 使用 NAT 模式（避免 Mirrored 模式的复杂性）
2. 使用 Docker Desktop GUI 配置代理
3. 需要时可切换到 Mirrored 模式（已准备解决方案）
4. 偏好一键诊断工具（scripts/network-health-check.sh）

## 📞 重要参考

- [GitHub Issue #3540](https://github.com/clash-verge-rev/clash-verge-rev/issues/3540) - Docker pull 连接拒绝问题
- [V2EX 讨论](https://v2ex.com/t/1000081) - Mirrored 模式与 MTU 问题
- [Docker Desktop WSL2 文档](https://docs.docker.com/desktop/features/wsl/)

## 📋 TODO 列表模板

当需要解决网络问题时：
1. [ ] 运行健康检查脚本
2. [ ] 识别失败项
3. [ ] 查看 troubleshooting.md 对应部分
4. [ ] 实施修复方案
5. [ ] 重新运行健康检查验证